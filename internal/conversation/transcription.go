package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
)

const (
	transcriptionQueueSize = 1000
	transcriptionTimeout   = 3 * time.Minute

	TranscriptionStatusDone   = "done"
	TranscriptionStatusFailed = "failed"
)

// ErrTranscriptionDisabled is returned by a Transcriber when no provider is configured; the job is dropped silently.
var ErrTranscriptionDisabled = errors.New("transcription disabled")

// Transcriber turns an audio blob into text.
type Transcriber interface {
	TranscribeAudio(ctx context.Context, data []byte, contentType string) (string, error)
}

type transcriptionJob struct {
	MessageUUID string
	MediaUUID   string
	ContentType string
}

// SetTranscriber sets the audio transcriber; when nil, audio attachments are not transcribed.
func (m *Manager) SetTranscriber(t Transcriber) {
	m.transcriber = t
}

// RunTranscriptionWorkers starts workers that transcribe audio attachments until ctx is done.
func (m *Manager) RunTranscriptionWorkers(ctx context.Context, workers int) {
	for range workers {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-m.transcriptionQueue:
					m.transcribe(ctx, job)
				}
			}
		}()
	}
}

// enqueueTranscriptions queues every audio attachment of the message for transcription.
func (m *Manager) enqueueTranscriptions(message *models.Message) {
	if m.transcriber == nil {
		return
	}
	for _, att := range message.Attachments {
		if att.UUID == "" || !strings.HasPrefix(strings.ToLower(att.ContentType), "audio/") {
			continue
		}
		select {
		case m.transcriptionQueue <- transcriptionJob{MessageUUID: message.UUID, MediaUUID: att.UUID, ContentType: att.ContentType}:
		default:
			m.lo.Warn("transcription queue full, skipping audio", "message_uuid", message.UUID, "media_uuid", att.UUID)
		}
	}
}

func (m *Manager) transcribe(ctx context.Context, job transcriptionJob) {
	blob, err := m.mediaStore.GetBlob(job.MediaUUID)
	if err != nil {
		m.lo.Error("error fetching audio blob for transcription", "media_uuid", job.MediaUUID, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, transcriptionTimeout)
	defer cancel()

	result := map[string]any{"updated_at": time.Now().UTC().Format(time.RFC3339)}
	text, err := m.transcriber.TranscribeAudio(ctx, blob, job.ContentType)
	switch {
	case errors.Is(err, ErrTranscriptionDisabled):
		return
	case err != nil:
		m.lo.Error("error transcribing audio", "message_uuid", job.MessageUUID, "media_uuid", job.MediaUUID, "error", err)
		result["status"] = TranscriptionStatusFailed
	default:
		result["status"] = TranscriptionStatusDone
		result["text"] = text
	}

	b, err := json.Marshal(result)
	if err != nil {
		return
	}
	var row struct {
		UUID             string          `db:"uuid"`
		ConversationUUID string          `db:"conversation_uuid"`
		Meta             json.RawMessage `db:"meta"`
	}
	if err := m.q.MergeMessageTranscription.Get(&row, job.MessageUUID, job.MediaUUID, b); err != nil {
		m.lo.Error("error saving transcription", "message_uuid", job.MessageUUID, "error", err)
		return
	}
	m.BroadcastMessageUpdate(row.ConversationUUID, row.UUID, map[string]any{"meta": stripCSATUUID(row.Meta)})
}
