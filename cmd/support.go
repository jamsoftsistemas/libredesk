package main

import (
	"encoding/json"
	"fmt"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const internalSupportWidgetSessionTTL = 5 * time.Minute

// internalSupportWidgetSessionResp is returned to logged-in agents so they can open the
// internal support widget, authenticated as a contact of the internal support inbox.
type internalSupportWidgetSessionResp struct {
	InboxUUID string `json:"inbox_uuid"`
	JWT       string `json:"jwt"`
}

// handleGetInternalSupportWidgetSession mints a short-lived JWT, signed with the internal
// support inbox's own secret, that identifies the logged-in agent as a contact of that inbox.
// This lets the same chat widget used by external customers be embedded inside the helpdesk
// for agents to reach internal support.
func handleGetInternalSupportWidgetSession(r *fastglue.Request) error {
	var app = r.Context.(*App)

	user, ok := r.RequestCtx.UserValue("user").(amodels.User)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	inboxIDRaw, err := app.setting.Get("app.internal_support_inbox_id")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var id int
	if err := json.Unmarshal(inboxIDRaw, &id); err != nil {
		id = 0
	}
	if id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("globals.messages.notFound"), nil, envelope.InputError)
	}

	inbox, err := app.inbox.GetDBRecord(id)
	if err != nil {
		app.lo.Error("error fetching internal support inbox", "inbox_id", id, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("validation.notFoundInbox"), nil, envelope.InputError)
	}
	if !inbox.Enabled || inbox.Channel != livechat.ChannelLiveChat || !inbox.Secret.Valid || inbox.Secret.String == "" {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("validation.notFoundInbox"), nil, envelope.InputError)
	}

	now := time.Now()
	claims := Claims{
		ExternalUserID: fmt.Sprintf("agent-%d", user.ID),
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(internalSupportWidgetSessionTTL)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(inbox.Secret.String))
	if err != nil {
		app.lo.Error("error signing internal support widget JWT", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}

	return r.SendEnvelope(internalSupportWidgetSessionResp{
		InboxUUID: inbox.UUID,
		JWT:       signed,
	})
}
