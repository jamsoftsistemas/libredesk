package main

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	uazapiChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/uazapi"
	"github.com/abhinavxd/libredesk/internal/uazapi"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const uazapiCallTimeout = 30 * time.Second

// uazapiWebhookEvents is the minimal event set a helpdesk needs: inbound messages, delivery/read receipts and
// connection state. Groups and the app's own outbound sends are filtered at the gateway via excludeMessages.
var (
	uazapiWebhookEvents          = []string{"messages", "messages_update", "connection"}
	uazapiWebhookExcludeMessages = []string{"wasSentByApi", "isGroupYes"}
)

func uazapiCallbackURLFromRoot(root string, inboxID int, secret string) string {
	if root == "" || secret == "" {
		return ""
	}
	return strings.TrimRight(root, "/") + "/webhooks/uazapi/" + strconv.Itoa(inboxID) + "/" + secret
}

// uazapiWebhookRegistration reports the computed callback URL and whether registering it with the gateway succeeded.
type uazapiWebhookRegistration struct {
	WebhookURL        string `json:"webhook_url,omitempty"`
	WebhookRegistered bool   `json:"webhook_registered"`
	WebhookError      string `json:"webhook_error,omitempty"`
}

// registerUazapiWebhook computes this inbox's callback URL and, if the app's root URL is a public HTTPS
// address the gateway can reach, registers it. It never returns an error: failures are reported in the
// result so the caller can still proceed (e.g. with connecting) and surface the problem to the admin.
func registerUazapiWebhook(ctx context.Context, app *App, id int, cfg uazapiChannel.Config) uazapiWebhookRegistration {
	root, _ := app.setting.GetAppRootURL()
	callbackURL := uazapiCallbackURLFromRoot(root, id, cfg.WebhookSecret)
	reg := uazapiWebhookRegistration{WebhookURL: callbackURL}

	if callbackURL == "" || !isPublicWebhookURL(root) {
		reg.WebhookError = app.i18n.T("admin.inbox.uazapi.error.rootURLNotPublic")
		app.lo.Warn("uazapi webhook not auto-registered: the app root URL must be a public HTTPS URL the gateway can reach", "inbox_id", id, "root_url", root)
		return reg
	}
	if err := app.uazapiClient.SetWebhook(ctx, cfg.Account(), callbackURL, uazapiWebhookEvents, uazapiWebhookExcludeMessages); err != nil {
		app.lo.Warn("uazapi webhook registration failed, otherwise inbound messages will not arrive", "inbox_id", id, "error", err)
		reg.WebhookError = errString(err)
		return reg
	}
	reg.WebhookRegistered = true
	return reg
}

// handleUazapiConnect provisions the instance (if it has no token yet), registers the webhook, and starts
// a connection, returning the QR code/pair code for the admin to scan.
func handleUazapiConnect(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	rec, err := app.inbox.GetDBRecord(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	cfg, err := uazapiConfigFromRecord(rec)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if app.uazapiClient == nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}

	ctx, cancel := context.WithTimeout(context.Background(), uazapiCallTimeout)
	defer cancel()

	if cfg.InstanceToken == "" {
		if cfg.AdminToken == "" || cfg.InstanceName == "" {
			return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.uazapi.error.missingCreateFields"), nil))
		}
		inst, err := app.uazapiClient.CreateInstance(ctx, cfg.Account(), cfg.InstanceName)
		if err != nil || inst.Token == "" {
			app.lo.Error("error creating uazapi instance", "inbox_id", id, "error", err)
			return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("admin.inbox.uazapi.error.connectFailed", "error", errString(err)), nil))
		}
		cfg.InstanceToken = inst.Token
		if err := persistUazapiConfig(app, id, cfg); err != nil {
			return sendErrorEnvelope(r, err)
		}
		if err := reloadInbox(app, id); err != nil {
			app.lo.Error("error reloading inbox after uazapi instance creation", "inbox_id", id, "error", err)
		}
	}

	webhookReg := registerUazapiWebhook(ctx, app, id, cfg)

	resp, err := app.uazapiClient.Connect(ctx, cfg.Account(), "")
	if err != nil {
		app.lo.Error("error connecting uazapi instance", "inbox_id", id, "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("admin.inbox.uazapi.error.connectFailed", "error", errString(err)), nil))
	}
	return r.SendEnvelope(struct {
		uazapi.InstanceResponse
		uazapiWebhookRegistration
	}{resp, webhookReg})
}

// handleUazapiRegisterWebhook (re)registers the webhook using the app's current root URL, without touching
// the WhatsApp session. Useful after fixing the root URL (e.g. pointing it at a tunnel) post-connect.
func handleUazapiRegisterWebhook(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	rec, err := app.inbox.GetDBRecord(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	cfg, err := uazapiConfigFromRecord(rec)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if cfg.InstanceToken == "" || app.uazapiClient == nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.uazapi.error.missingCreateFields"), nil))
	}
	ctx, cancel := context.WithTimeout(context.Background(), uazapiCallTimeout)
	defer cancel()
	return r.SendEnvelope(registerUazapiWebhook(ctx, app, id, cfg))
}

// handleUazapiStatus polls the instance's current connection state, including the QR code/pair code while connecting.
func handleUazapiStatus(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	rec, err := app.inbox.GetDBRecord(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	cfg, err := uazapiConfigFromRecord(rec)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	root, _ := app.setting.GetAppRootURL()
	webhookURL := uazapiCallbackURLFromRoot(root, id, cfg.WebhookSecret)

	if cfg.InstanceToken == "" || app.uazapiClient == nil {
		return r.SendEnvelope(struct {
			Instance   map[string]any `json:"instance"`
			Status     map[string]any `json:"status"`
			WebhookURL string         `json:"webhook_url,omitempty"`
		}{map[string]any{}, map[string]any{"connected": false, "loggedIn": false}, webhookURL})
	}

	ctx, cancel := context.WithTimeout(context.Background(), uazapiCallTimeout)
	defer cancel()
	resp, err := app.uazapiClient.Status(ctx, cfg.Account())
	if err != nil {
		app.lo.Error("error fetching uazapi instance status", "inbox_id", id, "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(struct {
		uazapi.InstanceResponse
		WebhookURL string `json:"webhook_url,omitempty"`
	}{resp, webhookURL})
}

// handleUazapiDisconnect logs the instance's WhatsApp session out. Connect must be used again afterwards.
func handleUazapiDisconnect(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	rec, err := app.inbox.GetDBRecord(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	cfg, err := uazapiConfigFromRecord(rec)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if cfg.InstanceToken == "" || app.uazapiClient == nil {
		return r.SendEnvelope(true)
	}
	ctx, cancel := context.WithTimeout(context.Background(), uazapiCallTimeout)
	defer cancel()
	if err := app.uazapiClient.Disconnect(ctx, cfg.Account()); err != nil {
		app.lo.Error("error disconnecting uazapi instance", "inbox_id", id, "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(true)
}

// persistUazapiConfig saves a config mutation (e.g. a freshly created instance token) back to the inbox row.
func persistUazapiConfig(app *App, inboxID int, cfg uazapiChannel.Config) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := app.inbox.UpdateConfig(inboxID, b); err != nil {
		app.lo.Error("error persisting uazapi config", "inbox_id", inboxID, "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
