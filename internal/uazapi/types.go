package uazapi

import "encoding/json"

// Account holds the per-inbox connection details needed to call the UAZAPI gateway.
type Account struct {
	// BaseURL is the gateway's root, e.g. https://free.uazapi.com or a self-hosted origin. No trailing slash.
	BaseURL string
	// InstanceToken authenticates regular (non-admin) calls via the "token" header.
	InstanceToken string
	// AdminToken authenticates instance-management calls via the "admintoken" header. Only needed to create an instance.
	AdminToken string
}

// Instance mirrors UAZAPI's instance object, returned by create/connect/status.
type Instance struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Token         string `json:"token"`
	Status        string `json:"status"` // disconnected | connecting | connected | hibernated
	QRCode        string `json:"qrcode"`
	PairCode      string `json:"paircode"`
	ProfileName   string `json:"profileName"`
	ProfilePicURL string `json:"profilePicUrl"`
	Owner         string `json:"owner"`

	LastDisconnect       string `json:"lastDisconnect"`
	LastDisconnectReason string `json:"lastDisconnectReason"`
}

// ConnectionStatus mirrors UAZAPI's nested "status" object.
type ConnectionStatus struct {
	Connected bool   `json:"connected"`
	LoggedIn  bool   `json:"loggedIn"`
	JID       string `json:"jid"`
}

// InstanceResponse is the common response shape for create/connect/status.
type InstanceResponse struct {
	Instance Instance         `json:"instance"`
	Status   ConnectionStatus `json:"status"`
}

// Message mirrors the fields UAZAPI returns from a send call and includes in "messages"/"messages_update" webhooks.
type Message struct {
	ID               string          `json:"id"`
	MessageID        string          `json:"messageid"`
	ChatID           string          `json:"chatid"`
	Sender           string          `json:"sender"`
	SenderName       string          `json:"senderName"`
	SenderPN         string          `json:"sender_pn"`
	SenderLID        string          `json:"sender_lid"`
	FromMe           bool            `json:"fromMe"`
	WasSentByApi     bool            `json:"wasSentByApi"`
	IsGroup          bool            `json:"isGroup"`
	MessageType      string          `json:"messageType"`
	Text             string          `json:"text"`
	Content          json.RawMessage `json:"content"`
	Quoted           string          `json:"quoted"`
	FileURL          string          `json:"fileURL"`
	MimeType         string          `json:"mimetype"`
	MessageTimestamp int64           `json:"messageTimestamp"`
	Status           string          `json:"status"`
	TrackSource      string          `json:"track_source"`
	TrackID          string          `json:"track_id"`
}

// SendResponse is returned by /send/text, /send/media and /send/menu. The schema is an allOf merge of Message
// with a "response" wrapper, so every field that matters (id, messageid, status...) is already at the top
// level; "response.message" is redundant and its shape varies by endpoint (an object for text/media, a plain
// confirmation string for menu), so it is decoded loosely and never relied on.
type SendResponse struct {
	Message
	Response struct {
		Status  string          `json:"status"`
		Message json.RawMessage `json:"message"`
	} `json:"response"`
}

// SourceID returns the gateway's message id for status correlation.
func (r SendResponse) SourceID() string {
	return r.MessageID
}

// DownloadResponse is returned by /message/download.
type DownloadResponse struct {
	FileURL       string `json:"fileURL"`
	MimeType      string `json:"mimetype"`
	Base64Data    string `json:"base64Data"`
	Transcription string `json:"transcription"`
}

// MarkReadResult is one entry of /message/markread's "results" array.
type MarkReadResult struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	Error     string `json:"error"`
}

// webhookEnvelope carries the fields common to every UAZAPI webhook delivery.
type webhookEnvelope struct {
	EventType    string `json:"EventType"`
	Owner        string `json:"owner"`
	Token        string `json:"token"`
	BaseURL      string `json:"BaseUrl"`
	InstanceName string `json:"instanceName"`
}

// Event types as sent in the "EventType" field of a webhook delivery.
const (
	EventMessages       = "messages"
	EventMessagesUpdate = "messages_update"
	EventConnection     = "connection"
)

// MessagesEvent is the "messages" webhook: one inbound or outbound message.
type MessagesEvent struct {
	webhookEnvelope
	Message Message         `json:"message"`
	Chat    json.RawMessage `json:"chat"`
}

// MessagesUpdateEvent is the "messages_update" webhook: a delivery/read receipt.
type MessagesUpdateEvent struct {
	webhookEnvelope
	Type  string `json:"type"`  // ReadReceipt | GroupReceipts
	State string `json:"state"` // Delivered | Read | Played
	Event struct {
		Chat       string   `json:"Chat"`
		ChatID     string   `json:"chatid"`
		Sender     string   `json:"Sender"`
		MessageIDs []string `json:"MessageIDs"`
		Timestamp  int64    `json:"Timestamp"`
		Type       string   `json:"Type"`
		IsFromMe   bool     `json:"IsFromMe"`
		IsGroup    bool     `json:"IsGroup"`
		SenderPN   string   `json:"sender_pn"`
		SenderLID  string   `json:"sender_lid"`
	} `json:"event"`
}

// ConnectionEvent is the "connection" webhook: the instance's socket state changed.
type ConnectionEvent struct {
	webhookEnvelope
	EventID  string `json:"event_id"`
	Instance struct {
		Name                 string `json:"name"`
		Status               string `json:"status"`
		LastDisconnect       string `json:"lastDisconnect"`
		LastDisconnectReason string `json:"lastDisconnectReason"`
	} `json:"instance"`
	Type         string `json:"type"` // "Disconnected" | "TemporaryBan"
	TemporaryBan bool   `json:"temporaryBan"`
}
