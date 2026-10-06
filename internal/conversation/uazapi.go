package conversation

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/countries"
	"github.com/abhinavxd/libredesk/internal/envelope"
	uazapiChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/uazapi"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// prepareUazapiOutbound resolves the recipient number and writes it into metaMap. Unlike WhatsApp's Cloud API,
// UAZAPI has no customer-service window and no template requirement: any free-form text or media can be sent
// at any time, so this is a much smaller version of prepareWhatsAppOutbound.
func (m *Manager) prepareUazapiOutbound(inboxRecord imodels.Inbox, conversationUUID string, content string, hasAttachments bool, metaMap map[string]any) (string, error) {
	var conv struct {
		InboxID   int `db:"inbox_id"`
		ContactID int `db:"contact_id"`
	}
	if err := m.q.GetConversationInboxContact.Get(&conv, conversationUUID); err != nil {
		m.lo.Error("error fetching conversation inbox and contact", "conversation_uuid", conversationUUID, "error", err)
		return content, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if conv.InboxID != inboxRecord.ID {
		return content, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	toNumber, err := m.userStore.GetChannelIdentity(conv.ContactID, uazapiChannel.ChannelUazapi)
	if err != nil {
		return content, err
	}
	if toNumber == "" {
		contact, err := m.userStore.Get(conv.ContactID, "", nil)
		if err != nil {
			return content, err
		}
		if contact.PhoneNumber.String == "" {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.contactNoPhone"), nil)
		}
		dialCode := countries.DialCodeForISO(contact.PhoneNumberCountryCode.String)
		if dialCode == "" {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.contactCountryCodeInvalid"), nil)
		}
		var matchesCountry bool
		toNumber, matchesCountry = stringutil.WhatsAppPhoneForDialCode(contact.PhoneNumber.String, dialCode)
		if !matchesCountry || toNumber == "" {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.contactCountryCodeInvalid"), nil)
		}
		// Link the identity now so the contact's reply threads back to this contact instead of forking a duplicate.
		linkedID, err := m.userStore.LinkChannelIdentity(conv.ContactID, uazapiChannel.ChannelUazapi, toNumber)
		if err != nil {
			return content, err
		}
		if linkedID != conv.ContactID {
			return content, envelope.NewError(envelope.ConflictError, m.i18n.T("conversation.whatsapp.error.numberLinkedToAnotherContact"), nil)
		}
	}

	if err := m.validateUazapiContent(content, hasAttachments); err != nil {
		return content, err
	}

	send := uazapiChannel.SendMeta{ToNumber: toNumber}
	// Set by sendUazapiCSAT: sends this message as a single interactive URL button instead of plain text.
	if buttonURL, ok := metaMap["uazapi_button_url"].(string); ok && buttonURL != "" {
		send.ButtonURL = buttonURL
		send.ButtonText, _ = metaMap["uazapi_button_text"].(string)
	}
	encoded, err := json.Marshal(send)
	if err != nil {
		return content, err
	}
	metaMap["uazapi"] = json.RawMessage(encoded)
	return content, nil
}

// sendUazapiCSAT sends a CSAT request as a single "Rate us" button linking to the CSAT page. UAZAPI has no
// 24h window and no templates, so unlike WhatsApp's Cloud API this can be sent as a native message any time.
func (m *Manager) sendUazapiCSAT(actorUserID int, conversation models.Conversation, csatUUID, csatURL string) error {
	meta := map[string]any{
		"is_csat":            true,
		"is_automated":       true,
		"csat_uuid":          csatUUID,
		"uazapi_button_url":  csatURL,
		"uazapi_button_text": m.i18n.T("globals.terms.rateUs"),
	}
	content := m.i18n.T("globals.messages.pleaseRateConversation")
	if _, err := m.QueueReply(nil, conversation.InboxID, actorUserID, conversation.ContactID, conversation.UUID, content, nil, nil, nil, meta); err != nil {
		m.lo.Error("error sending uazapi CSAT", "conversation_uuid", conversation.UUID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func (m *Manager) validateUazapiContent(content string, hasAttachments bool) error {
	if strings.TrimSpace(content) == "" && !hasAttachments {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.messageOrAttachmentRequired"), nil)
	}
	if utf8.RuneCountInString(stringutil.HTML2WhatsApp(content)) > whatsAppMaxTextLength {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("conversation.whatsapp.error.tooLong", "limit", strconv.Itoa(whatsAppMaxTextLength)), nil)
	}
	return nil
}
