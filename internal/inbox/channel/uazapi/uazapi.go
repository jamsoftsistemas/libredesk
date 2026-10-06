// Package uazapi implements an inbox backed by the UAZAPI gateway, an unofficial
// WhatsApp API connected by scanning a QR code. Unlike the Meta Cloud API channel,
// there is no 24h customer-service window and no template requirement: any message
// can be sent as free-form text or media at any time.
package uazapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/uazapi"
	"github.com/zerodha/logf"
)

const ChannelUazapi = inbox.ChannelUazapi

const CallTimeout = 30 * time.Second

// Covers base64-encoding and uploading a large attachment, then the send call itself.
const attachmentSendTimeout = 2*time.Minute + CallTimeout

const (
	sendMaxAttempts  = 3
	sendRetryBackoff = 2 * time.Second
)

// Config is the per-inbox UAZAPI configuration from the inbox config JSONB, with secrets already decrypted.
type Config struct {
	BaseURL       string `json:"base_url"`
	InstanceName  string `json:"instance_name"`
	InstanceToken string `json:"instance_token"`
	AdminToken    string `json:"admin_token"`
	WebhookSecret string `json:"webhook_secret"`
}

func (c Config) Account() uazapi.Account {
	return uazapi.Account{
		BaseURL:       strings.TrimRight(c.BaseURL, "/"),
		InstanceToken: c.InstanceToken,
		AdminToken:    c.AdminToken,
	}
}

// SendMeta is the per-message metadata threaded through OutboundMessage.Meta. A set ButtonURL sends the
// message as a single interactive URL button instead of plain text (e.g. a CSAT survey link).
type SendMeta struct {
	ToNumber         string `json:"to_number"`
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	ButtonText       string `json:"button_text,omitempty"`
	ButtonURL        string `json:"button_url,omitempty"`
}

// SourceIDUpdater persists the UAZAPI message id for status correlation.
type SourceIDUpdater interface {
	UpdateMessageSourceID(messageUUID, sourceID string) error
}

type Uazapi struct {
	id            int
	name          string
	config        Config
	client        *uazapi.Client
	lo            *logf.Logger
	messageStore  inbox.MessageStore
	sourceUpdater SourceIDUpdater
	retryBackoff  time.Duration
}

type Opts struct {
	ID            int
	Name          string
	Config        Config
	Client        *uazapi.Client
	Lo            *logf.Logger
	SourceUpdater SourceIDUpdater
}

func New(store inbox.MessageStore, opts Opts) (*Uazapi, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("uazapi client is required")
	}
	if opts.Config.BaseURL == "" || opts.Config.InstanceToken == "" {
		return nil, fmt.Errorf("base_url and instance_token are required")
	}
	if opts.Lo == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &Uazapi{
		id:            opts.ID,
		name:          opts.Name,
		config:        opts.Config,
		client:        opts.Client,
		lo:            opts.Lo,
		messageStore:  store,
		sourceUpdater: opts.SourceUpdater,
		retryBackoff:  sendRetryBackoff,
	}, nil
}

func (u *Uazapi) Identifier() int          { return u.id }
func (u *Uazapi) Config() Config           { return u.config }
func (u *Uazapi) Channel() string          { return ChannelUazapi }
func (u *Uazapi) Name() string             { return u.name }
func (u *Uazapi) FromAddress() string      { return "" }
func (u *Uazapi) ReplyToAddress() string   { return "" }
func (u *Uazapi) FromNameTemplate() string { return "" }
func (u *Uazapi) Close() error             { return nil }

// Receive is a no-op. Inbound messages arrive via the webhook handler.
func (u *Uazapi) Receive(ctx context.Context) error { return nil }

// Send retries a 429 or 5xx only, since a send carries no idempotency key and the gateway may already have accepted it.
func (u *Uazapi) Send(message models.OutboundMessage) error {
	var err error
	for attempt := 1; ; attempt++ {
		err = u.send(message)
		if err == nil || attempt >= sendMaxAttempts || !refusedSend(err) {
			return err
		}
		u.lo.Warn("uazapi refused the send, retrying", "message_uuid", message.UUID, "attempt", attempt, "error", err)
		time.Sleep(u.retryBackoff * time.Duration(attempt))
	}
}

func refusedSend(err error) bool {
	var ae *uazapi.APIError
	if !errors.As(err, &ae) {
		return false
	}
	return ae.IsRateLimited() || ae.StatusCode >= http.StatusInternalServerError
}

func (u *Uazapi) send(message models.OutboundMessage) error {
	meta, err := parseSendMeta(message.Meta)
	if err != nil {
		return fmt.Errorf("parsing uazapi send meta: %w", err)
	}
	if meta.ToNumber == "" {
		return fmt.Errorf("missing recipient number on outbound message")
	}

	timeout := CallTimeout
	if len(message.Attachments) > 0 {
		timeout = attachmentSendTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	acc := u.config.Account()
	opts := uazapi.SendOpts{
		ReplyID:     meta.ReplyToMessageID,
		TrackSource: "libredesk",
		TrackID:     message.UUID,
	}

	var resp uazapi.SendResponse
	switch {
	case meta.ButtonURL != "" && len(message.Attachments) == 0:
		resp, err = u.client.SendButtonMenu(ctx, acc, meta.ToNumber, textBody(message), meta.ButtonText, meta.ButtonURL, opts)
	case len(message.Attachments) > 0:
		resp, err = u.sendAttachment(ctx, acc, opts, message)
	case strings.TrimSpace(textBody(message)) != "":
		resp, err = u.client.SendText(ctx, acc, meta.ToNumber, textBody(message), opts)
	default:
		return fmt.Errorf("outbound message has no content")
	}

	if err == nil {
		if sourceID := resp.SourceID(); sourceID != "" && u.sourceUpdater != nil {
			if upErr := u.sourceUpdater.UpdateMessageSourceID(message.UUID, sourceID); upErr != nil {
				u.lo.Error("failed to persist uazapi source id", "message_uuid", message.UUID, "source_id", sourceID, "error", upErr)
			}
		}
	}
	return err
}

// sendAttachment uploads one attachment as a data URI. UAZAPI accepts one file per message.
func (u *Uazapi) sendAttachment(ctx context.Context, acc uazapi.Account, opts uazapi.SendOpts, message models.OutboundMessage) (uazapi.SendResponse, error) {
	if len(message.Attachments) > 1 {
		return uazapi.SendResponse{}, fmt.Errorf("uazapi accepts one attachment per message, got %d", len(message.Attachments))
	}
	meta, err := parseSendMeta(message.Meta)
	if err != nil {
		return uazapi.SendResponse{}, err
	}

	att := message.Attachments[0]
	dataURI := fmt.Sprintf("data:%s;base64,%s", att.ContentType, base64.StdEncoding.EncodeToString(att.Content))
	return u.client.SendMedia(ctx, acc, meta.ToNumber, mediaTypeForAttachment(att), dataURI, strings.TrimSpace(textBody(message)), att.Name, opts)
}

func parseSendMeta(raw json.RawMessage) (SendMeta, error) {
	var meta SendMeta
	if len(raw) == 0 {
		return meta, nil
	}
	// SendMeta lives under a "uazapi" key in message.meta to avoid colliding with other channels' meta keys.
	var envelope struct {
		Uazapi json.RawMessage `json:"uazapi"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Uazapi) > 0 {
		if err := json.Unmarshal(envelope.Uazapi, &meta); err != nil {
			return meta, fmt.Errorf("decoding uazapi meta envelope: %w", err)
		}
		return meta, nil
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

// textBody renders HTML replies as plain text. UAZAPI has no HTML/markup rendering beyond WhatsApp's own markers.
func textBody(m models.OutboundMessage) string {
	if m.ContentType == models.ContentTypeHTML && m.Content != "" {
		return stringutil.HTML2WhatsApp(m.Content)
	}
	if m.TextContent != "" {
		return m.TextContent
	}
	return m.Content
}

func mediaTypeForAttachment(att attachment.Attachment) string {
	mime := strings.ToLower(strings.TrimSpace(att.ContentType))
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	default:
		return "document"
	}
}
