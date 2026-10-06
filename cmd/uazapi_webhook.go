package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	uazapiChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/uazapi"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/uazapi"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	uazapiDefaultContactName  = "Contact"
	uazapiMediaAttemptTimeout = 3 * time.Minute
)

// uazapiConversationLocks serializes the open-conversation lookup + create per contact and inbox.
var uazapiConversationLocks = &keyedLock{entries: make(map[string]*keyedLockEntry)}

func lockUazapiConversation(contactID, inboxID int) func() {
	return uazapiConversationLocks.lock(strconv.Itoa(contactID) + ":" + strconv.Itoa(inboxID))
}

// handleUazapiWebhookEvent accepts the gateway's single webhook delivery per instance, authenticated by a
// per-inbox secret in the path (UAZAPI signs nothing itself).
func handleUazapiWebhookEvent(r *fastglue.Request) error {
	app := r.Context.(*App)

	inboxID, err := inboxIDFromPath(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid inbox id", nil, envelope.InputError)
	}
	secret, _ := r.RequestCtx.UserValue("secret").(string)

	cfg, err := uazapiConfigForInbox(app, inboxID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "inbox not found", nil, envelope.NotFoundError)
	}
	if cfg.WebhookSecret == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(cfg.WebhookSecret)) != 1 {
		app.lo.Warn("uazapi webhook rejected: secret mismatch", "inbox_id", inboxID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "invalid secret", nil, envelope.PermissionError)
	}

	body := append([]byte(nil), r.RequestCtx.PostBody()...)

	ing := app.uazapiIngesterInstance()
	if ing == nil {
		if err := ensureUazapiIngester(app); err != nil {
			app.lo.Error("error starting uazapi ingester on demand", "inbox_id", inboxID, "error", err)
		}
		ing = app.uazapiIngesterInstance()
	}
	if ing == nil {
		app.lo.Error("uazapi ingester not initialized", "inbox_id", inboxID)
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "uazapi ingester unavailable", nil, envelope.GeneralError)
	}
	if err := ing.Enqueue(inboxID, body); err != nil {
		app.lo.Error("error enqueuing uazapi webhook to durable stream, asking the gateway to retry", "inbox_id", inboxID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "busy, retry shortly", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]string{"status": "ok"})
}

// processUazapiEvent dispatches one parsed webhook delivery.
func processUazapiEvent(ctx context.Context, app *App, inboxID int, event any) error {
	switch ev := event.(type) {
	case *uazapi.MessagesEvent:
		return ingestUazapiMessage(ctx, app, inboxID, ev)
	case *uazapi.MessagesUpdateEvent:
		return applyUazapiStatus(app, ev)
	case *uazapi.ConnectionEvent:
		applyUazapiConnectionEvent(app, inboxID, ev)
		return nil
	default:
		return nil
	}
}

func ingestUazapiMessage(ctx context.Context, app *App, inboxID int, ev *uazapi.MessagesEvent) error {
	m := ev.Message
	if m.MessageID == "" || m.ChatID == "" {
		return fmt.Errorf("missing message id or chat id")
	}
	// Echoes of our own sends and group chats are filtered server-side (excludeMessages), but a
	// misconfigured webhook or a race on save must not be allowed to poison a conversation.
	if m.FromMe || m.WasSentByApi || m.IsGroup {
		return nil
	}

	number := uazapiNumberFromJID(m.ChatID)
	if number == "" {
		// @lid and other non-phone JIDs have no stable identity to thread on.
		return nil
	}

	if iq := app.uazapiIngesterInstance(); iq != nil {
		unlock := iq.lockSender(number)
		defer unlock()
	}

	inbRec, cfg, err := uazapiInboxRecord(app, inboxID)
	if err != nil {
		app.lo.Warn("dropping uazapi message: inbox unavailable", "inbox_id", inboxID, "error", err)
		return nil
	}

	contactID, err := upsertUazapiContact(app, m, number)
	if err != nil {
		return fmt.Errorf("resolving contact: %w", err)
	}
	contact, err := app.user.GetContactOrVisitor(contactID, "")
	if err != nil {
		return fmt.Errorf("checking contact: %w", err)
	}
	if !contact.Enabled {
		return nil
	}

	if exists, err := app.conversation.MessageExists(m.MessageID); err != nil {
		return fmt.Errorf("checking duplicate: %w", err)
	} else if exists {
		return nil
	}

	attachments, err := fetchUazapiAttachments(ctx, app, cfg, m)
	if err != nil {
		return fmt.Errorf("downloading uazapi media: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	defer lockUazapiConversation(contactID, inboxID)()

	isNewConversation := false
	conversationID, conversationUUID, err := app.conversation.GetLatestOpenConversationForContact(contactID, inboxID)
	if err != nil && inbRec.ReopenWindowHours > 0 {
		conversationID, conversationUUID, err = app.conversation.GetReopenableConversationForContact(contactID, inboxID, inbRec.ReopenWindowHours)
	}
	if err != nil {
		conversationID, conversationUUID, err = app.conversation.CreateConversation(
			contactID,
			inboxID,
			uazapiTextPreview(m),
			time.Now(),
			"",
			false,
			nil,
			nil,
			0,
			0,
		)
		if err != nil {
			return fmt.Errorf("creating conversation: %w", err)
		}
		isNewConversation = true
	}

	content := uazapiTextPreview(m)
	if m.Text == "" && len(attachments) > 0 {
		content = ""
	}

	msg := cmodels.Message{
		Channel:          uazapiChannel.ChannelUazapi,
		ConversationID:   conversationID,
		ConversationUUID: conversationUUID,
		SenderID:         contactID,
		SenderType:       cmodels.SenderTypeContact,
		Type:             cmodels.MessageIncoming,
		Status:           cmodels.MessageStatusReceived,
		InboxID:          inboxID,
		Content:          content,
		ContentType:      cmodels.ContentTypeText,
		SourceID:         null.StringFrom(m.MessageID),
		Attachments:      attachments,
	}

	if _, err := app.conversation.ProcessIncomingWhatsAppMessage(msg, isNewConversation, time.UnixMilli(m.MessageTimestamp)); err != nil {
		return fmt.Errorf("processing uazapi message: %w", err)
	}
	return nil
}

// applyUazapiStatus updates a message's delivery status. Out-of-order deliveries can regress the status;
// documented as a known limitation, matching the same best-effort behavior as the WhatsApp Cloud API channel.
func applyUazapiStatus(app *App, ev *uazapi.MessagesUpdateEvent) error {
	var status string
	switch ev.State {
	case uazapi.StateDelivered:
		status = "delivered"
	case uazapi.StateRead:
		status = "read"
	default:
		return nil
	}
	eventAt := time.Now()
	if ev.Event.Timestamp > 0 {
		eventAt = time.Unix(ev.Event.Timestamp, 0)
	}
	var errs []error
	for _, id := range ev.Event.MessageIDs {
		if err := app.conversation.ApplyWhatsAppStatus(id, status, eventAt, ""); err != nil {
			app.lo.Warn("error applying uazapi message status", "message_id", id, "status", status, "error", err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("applying uazapi status: %v", errs)
	}
	return nil
}

func applyUazapiConnectionEvent(app *App, inboxID int, ev *uazapi.ConnectionEvent) {
	if ev.Type == "TemporaryBan" || ev.TemporaryBan {
		app.lo.Error("uazapi instance temporarily banned by WhatsApp", "inbox_id", inboxID, "instance", ev.Instance.Name)
		return
	}
	app.lo.Info("uazapi connection event", "inbox_id", inboxID, "status", ev.Instance.Status, "reason", ev.Instance.LastDisconnectReason)
}

// fetchUazapiAttachments returns (nil, nil) when there is nothing to fetch or the download permanently fails.
func fetchUazapiAttachments(ctx context.Context, app *App, cfg uazapiChannel.Config, m uazapi.Message) (attachment.Attachments, error) {
	if app.uazapiClient == nil {
		return nil, nil
	}
	acc := cfg.Account()

	fileURL := m.FileURL
	mimeType := m.MimeType
	if fileURL == "" {
		if !uazapiMessageHasMedia(m.MessageType) {
			return nil, nil
		}
		dlCtx, cancel := context.WithTimeout(ctx, uazapiMediaAttemptTimeout)
		info, err := app.uazapiClient.DownloadMedia(dlCtx, acc, m.MessageID, true)
		cancel()
		if err != nil {
			app.lo.Warn("uazapi media unavailable, inserting placeholder", "message_id", m.MessageID, "error", err)
			return nil, nil
		}
		fileURL = info.FileURL
		if mimeType == "" {
			mimeType = info.MimeType
		}
	}
	if fileURL == "" {
		return nil, nil
	}

	maxBytes := int64(app.consts.Load().(*constants).MaxFileUploadSizeMB) * 1024 * 1024
	dlCtx, cancel := context.WithTimeout(ctx, uazapiMediaAttemptTimeout)
	body, contentType, err := app.uazapiClient.FetchMedia(dlCtx, fileURL, maxBytes)
	cancel()
	if err != nil {
		app.lo.Warn("error downloading uazapi media, inserting placeholder", "message_id", m.MessageID, "error", err)
		return nil, nil
	}
	if len(body) == 0 {
		return nil, nil
	}
	if mimeType == "" {
		mimeType = contentType
	}

	return attachment.Attachments{
		attachment.Attachment{
			Name:        uazapiDefaultMediaFilename(m.MessageType, mimeType),
			ContentType: mimeType,
			Content:     body,
			Size:        len(body),
			Disposition: attachment.DispositionAttachment,
		},
	}, nil
}

func uazapiMessageHasMedia(messageType string) bool {
	t := strings.ToLower(messageType)
	for _, kind := range []string{"image", "video", "audio", "document", "sticker", "ptt"} {
		if strings.Contains(t, kind) {
			return true
		}
	}
	return false
}

func uazapiDefaultMediaFilename(messageType, mime string) string {
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	ext := "bin"
	if i := strings.LastIndex(mime, "/"); i >= 0 && i+1 < len(mime) {
		ext = mime[i+1:]
	}
	t := strings.ToLower(messageType)
	switch {
	case strings.Contains(t, "image"):
		return "image." + ext
	case strings.Contains(t, "video"):
		return "video." + ext
	case strings.Contains(t, "audio"), strings.Contains(t, "ptt"):
		return "audio." + ext
	case strings.Contains(t, "sticker"):
		return "sticker." + ext
	}
	return "document." + ext
}

func uazapiTextPreview(m uazapi.Message) string {
	if m.Text != "" {
		return m.Text
	}
	if uazapiMessageHasMedia(m.MessageType) {
		return "[" + strings.ToLower(m.MessageType) + "]"
	}
	return "[message]"
}

// uazapiNumberFromJID returns the phone number out of an individual chat JID, or "" for a group/@lid JID.
func uazapiNumberFromJID(jid string) string {
	if strings.HasSuffix(jid, "@g.us") || strings.HasSuffix(jid, "@lid") {
		return ""
	}
	number, _, _ := strings.Cut(jid, "@")
	number = strings.TrimSpace(number)
	for _, r := range number {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return number
}

func upsertUazapiContact(app *App, m uazapi.Message, number string) (int, error) {
	first, last := splitName(m.SenderName)
	contact := umodels.User{
		Type:      umodels.UserTypeContact,
		FirstName: first,
		LastName:  last,
	}
	id, err := app.user.UpsertContactByChannelIdentity(uazapiChannel.ChannelUazapi, number, &contact)
	if err != nil {
		return 0, err
	}
	if err := app.user.SetContactPhoneIfMissing(id, number, ""); err != nil {
		app.lo.Error("error setting uazapi contact phone", "user_id", id, "error", err)
	}
	if m.SenderName != "" {
		if err := app.user.UpdateContactNameIfDefault(id, first, last, uazapiDefaultContactName); err != nil {
			app.lo.Error("error updating uazapi contact name", "user_id", id, "error", err)
		}
	}
	return id, nil
}

func uazapiConfigForInbox(app *App, inboxID int) (uazapiChannel.Config, error) {
	if inb, err := app.inbox.Get(inboxID); err == nil {
		if ua, ok := inb.(interface{ Config() uazapiChannel.Config }); ok {
			return ua.Config(), nil
		}
	}
	rec, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return uazapiChannel.Config{}, err
	}
	return uazapiConfigFromRecord(rec)
}

func uazapiConfigFromRecord(rec imodels.Inbox) (uazapiChannel.Config, error) {
	if rec.Channel != uazapiChannel.ChannelUazapi {
		return uazapiChannel.Config{}, fmt.Errorf("inbox %d is not a uazapi inbox", rec.ID)
	}
	var cfg uazapiChannel.Config
	if err := json.Unmarshal(rec.Config, &cfg); err != nil {
		return uazapiChannel.Config{}, fmt.Errorf("decoding uazapi inbox config: %w", err)
	}
	return cfg, nil
}

// markUazapiMessageRead sends a read receipt to the gateway for an inbound message. Best-effort, logs and swallows failures.
func markUazapiMessageRead(app *App, inboxID int, sourceID string) {
	if app.uazapiClient == nil || sourceID == "" {
		return
	}
	cfg, err := uazapiConfigForInbox(app, inboxID)
	if err != nil {
		app.lo.Error("error fetching inbox config for uazapi read receipt", "inbox_id", inboxID, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), uazapiCallTimeout)
	defer cancel()
	if err := app.uazapiClient.MarkRead(ctx, cfg.Account(), []string{sourceID}); err != nil {
		app.lo.Warn("error marking uazapi message read", "inbox_id", inboxID, "source_id", sourceID, "error", err)
	}
}

func uazapiInboxRecord(app *App, inboxID int) (imodels.Inbox, uazapiChannel.Config, error) {
	rec, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return imodels.Inbox{}, uazapiChannel.Config{}, err
	}
	cfg, err := uazapiConfigFromRecord(rec)
	if err != nil {
		return imodels.Inbox{}, uazapiChannel.Config{}, err
	}
	if !rec.Enabled {
		return imodels.Inbox{}, uazapiChannel.Config{}, fmt.Errorf("inbox %d is disabled", inboxID)
	}
	return rec, cfg, nil
}
