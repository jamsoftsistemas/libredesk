package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	uazapiChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/uazapi"
	"github.com/abhinavxd/libredesk/internal/streamqueue"
	"github.com/abhinavxd/libredesk/internal/uazapi"
)

const (
	uazapiStream      = "libredesk:uazapi:inbound"
	uazapiStreamGroup = "libredesk"
	uazapiConsumer    = "ingester"

	// Must exceed the worst-case media-download budget so the reclaimer never re-runs a still-in-flight delivery.
	uazapiReclaimMinIdle = 10 * time.Minute

	uazapiEnqueueTimeout = 5 * time.Second
)

// uazapiJob is the durable envelope persisted to the stream. Body is the raw webhook POST body, parsed in the worker.
type uazapiJob struct {
	InboxID int             `json:"inbox_id"`
	Body    json.RawMessage `json:"body"`
}

// UazapiIngester is the durable inbound pipeline: a Redis-stream work queue plus per-sender serialization.
type UazapiIngester struct {
	queue       *streamqueue.Queue
	sourceLocks *keyedLock
}

func newUazapiIngester(app *App) (*UazapiIngester, error) {
	ing := &UazapiIngester{
		sourceLocks: &keyedLock{entries: make(map[string]*keyedLockEntry)},
	}
	q, err := streamqueue.New(streamqueue.Opts{
		Redis:        app.redis,
		Logger:       app.lo,
		Stream:       uazapiStream,
		Group:        uazapiStreamGroup,
		Consumer:     uazapiConsumer,
		Handler:      ing.handle(app),
		ClaimMinIdle: uazapiReclaimMinIdle,
	})
	if err != nil {
		return nil, err
	}
	ing.queue = q
	return ing, nil
}

// Run consumes the stream until Close is called.
func (i *UazapiIngester) Run() { i.queue.Run() }

// Close stops the queue and waits for in-flight work. Un-acked deliveries stay durable for the next start.
func (i *UazapiIngester) Close() { i.queue.Close() }

// Enqueue durably stores a raw webhook body for the inbox.
func (i *UazapiIngester) Enqueue(inboxID int, body []byte) error {
	data, err := json.Marshal(uazapiJob{InboxID: inboxID, Body: json.RawMessage(body)})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), uazapiEnqueueTimeout)
	defer cancel()
	return i.queue.Enqueue(ctx, data)
}

// lockSender blocks until the per-sender lock is held, returning the release func.
func (i *UazapiIngester) lockSender(from string) func() {
	return i.sourceLocks.lock(from)
}

// handle returns nil for an unparseable job so it is dropped rather than retried forever. A processing error keeps the entry pending for retry.
func (i *UazapiIngester) handle(app *App) streamqueue.Handler {
	return func(ctx context.Context, payload []byte) error {
		var job uazapiJob
		if err := json.Unmarshal(payload, &job); err != nil {
			app.lo.Error("error decoding uazapi stream job, dropping", "error", err)
			return nil
		}
		event, err := uazapi.ParsePayload(job.Body)
		if err != nil {
			app.lo.Error("error parsing uazapi webhook payload from stream, dropping", "inbox_id", job.InboxID, "error", err)
			return nil
		}
		return processUazapiEvent(ctx, app, job.InboxID, event)
	}
}

func (app *App) uazapiIngesterInstance() *UazapiIngester { return app.uazapiIngester.Load() }

func ensureUazapiIngester(app *App) error {
	if app.uazapiIngesterInstance() != nil {
		return nil
	}

	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return fmt.Errorf("listing inboxes: %w", err)
	}
	configured := false
	for _, rec := range inboxes {
		if rec.Channel == uazapiChannel.ChannelUazapi && rec.Enabled {
			configured = true
			break
		}
	}
	if !configured {
		return nil
	}

	app.uazapiIngesterMu.Lock()
	defer app.uazapiIngesterMu.Unlock()
	if app.uazapiIngesterInstance() != nil {
		return nil
	}

	ing, err := newUazapiIngester(app)
	if err != nil {
		return err
	}
	app.uazapiIngester.Store(ing)
	go ing.Run()
	app.lo.Info("uazapi inbound pipeline started")
	return nil
}
