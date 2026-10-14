// Package uazapi is a client for the UAZAPI gateway (https://uazapi.com), an unofficial
// WhatsApp API connected by scanning a QR code with a regular or WhatsApp Business phone.
package uazapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zerodha/logf"
)

const (
	defaultTimeout = 30 * time.Second
	mediaTimeout   = 2 * time.Minute
)

// APIError is returned for any non-2xx response from the gateway.
type APIError struct {
	StatusCode   int
	ProviderCode int
	Body         string
	RetryAfter   time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("uazapi: status %d: %s", e.StatusCode, e.Body)
}

// IsRateLimited reports whether the gateway asked the caller to back off (HTTP 429).
func (e *APIError) IsRateLimited() bool { return e.StatusCode == http.StatusTooManyRequests }

// IsReachoutTimelock reports WhatsApp error 463: a temporary cap on starting new chats.
func (e *APIError) IsReachoutTimelock() bool { return e.ProviderCode == 463 }

type Client struct {
	httpClient      *http.Client
	mediaHTTPClient *http.Client
	lo              *logf.Logger
}

func New(lo *logf.Logger) *Client {
	return &Client{
		httpClient:      &http.Client{Timeout: defaultTimeout},
		mediaHTTPClient: &http.Client{Timeout: mediaTimeout},
		lo:              lo,
	}
}

// SendOpts are the optional fields shared by /send/text and /send/media.
type SendOpts struct {
	ReplyID     string
	Delay       int
	ReadChat    bool
	TrackSource string
	TrackID     string
}

// CreateInstance provisions a new instance on the gateway. Requires acc.AdminToken.
func (c *Client) CreateInstance(ctx context.Context, acc Account, name string) (Instance, error) {
	var out struct {
		Instance Instance `json:"instance"`
		Token    string   `json:"token"`
		Name     string   `json:"name"`
	}
	body, err := c.do(ctx, http.MethodPost, acc, true, "/instance/create", map[string]any{"name": name}, c.httpClient)
	if err != nil {
		return Instance{}, err
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return Instance{}, fmt.Errorf("decoding instance/create response: %w", err)
	}
	inst := out.Instance
	if inst.Token == "" {
		inst.Token = out.Token
	}
	if inst.Name == "" {
		inst.Name = out.Name
	}
	return inst, nil
}

// Connect starts (or resumes) the WhatsApp session for the instance. With no phone it returns a QR code;
// with a phone number it returns a pair code. Poll Status afterwards for the code and the connected state.
func (c *Client) Connect(ctx context.Context, acc Account, phone string) (InstanceResponse, error) {
	payload := map[string]any{}
	if phone != "" {
		payload["phone"] = phone
	}
	body, err := c.do(ctx, http.MethodPost, acc, false, "/instance/connect", payload, c.httpClient)
	if err != nil {
		return InstanceResponse{}, err
	}
	var out InstanceResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return InstanceResponse{}, fmt.Errorf("decoding instance/connect response: %w", err)
	}
	return out, nil
}

// Status returns the instance's current connection state, including qrcode/paircode while connecting.
func (c *Client) Status(ctx context.Context, acc Account) (InstanceResponse, error) {
	body, err := c.do(ctx, http.MethodGet, acc, false, "/instance/status", nil, c.httpClient)
	if err != nil {
		return InstanceResponse{}, err
	}
	var out InstanceResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return InstanceResponse{}, fmt.Errorf("decoding instance/status response: %w", err)
	}
	return out, nil
}

// Disconnect logs the instance out. Connect must be called again (a fresh QR code) to use it afterwards.
func (c *Client) Disconnect(ctx context.Context, acc Account) error {
	_, err := c.do(ctx, http.MethodPost, acc, false, "/instance/disconnect", nil, c.httpClient)
	return err
}

// SetWebhook points the instance's single webhook at url for the given events, best-effort filtering out
// group messages and echoes of messages this app itself sent.
func (c *Client) SetWebhook(ctx context.Context, acc Account, url string, events, excludeMessages []string) error {
	payload := map[string]any{
		"enabled":         true,
		"url":             url,
		"events":          events,
		"excludeMessages": excludeMessages,
	}
	_, err := c.do(ctx, http.MethodPost, acc, false, "/webhook", payload, c.httpClient)
	return err
}

// SendText sends a free-form text message.
func (c *Client) SendText(ctx context.Context, acc Account, number, text string, opts SendOpts) (SendResponse, error) {
	payload := map[string]any{"number": number, "text": text}
	applySendOpts(payload, opts)
	return c.send(ctx, acc, "/send/text", payload)
}

// SendMedia sends an image/video/document/audio/ptt message. file is a URL or base64 payload.
func (c *Client) SendMedia(ctx context.Context, acc Account, number, mediaType, file, caption, docName string, opts SendOpts) (SendResponse, error) {
	payload := map[string]any{"number": number, "type": mediaType, "file": file}
	if caption != "" {
		payload["text"] = caption
	}
	if docName != "" {
		payload["docName"] = docName
	}
	applySendOpts(payload, opts)
	return c.send(ctx, acc, "/send/media", payload)
}

// SendButtonMenu sends a single interactive URL button (/send/menu, type "button"). Currently used to link
// out to a web page from a plain WhatsApp message, e.g. a CSAT survey, since UAZAPI has no template mechanism.
func (c *Client) SendButtonMenu(ctx context.Context, acc Account, number, text, buttonText, buttonURL string, opts SendOpts) (SendResponse, error) {
	payload := map[string]any{
		"number":  number,
		"type":    "button",
		"text":    text,
		"choices": []string{fmt.Sprintf("%s|%s", buttonText, buttonURL)},
	}
	applySendOpts(payload, opts)
	return c.send(ctx, acc, "/send/menu", payload)
}

func applySendOpts(payload map[string]any, opts SendOpts) {
	if opts.ReplyID != "" {
		payload["replyid"] = opts.ReplyID
	}
	if opts.Delay > 0 {
		payload["delay"] = opts.Delay
	}
	if opts.ReadChat {
		payload["readchat"] = true
	}
	if opts.TrackSource != "" {
		payload["track_source"] = opts.TrackSource
	}
	if opts.TrackID != "" {
		payload["track_id"] = opts.TrackID
	}
}

func (c *Client) send(ctx context.Context, acc Account, path string, payload map[string]any) (SendResponse, error) {
	body, err := c.do(ctx, http.MethodPost, acc, false, path, payload, c.httpClient)
	if err != nil {
		return SendResponse{}, err
	}
	var out SendResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return SendResponse{}, fmt.Errorf("decoding %s response: %w", path, err)
	}
	return out, nil
}

// DownloadMedia resolves a message's media to a short-lived CDN URL (and optionally transcribes audio).
func (c *Client) DownloadMedia(ctx context.Context, acc Account, messageID string, generateMP3 bool) (DownloadResponse, error) {
	payload := map[string]any{"id": messageID, "generate_mp3": generateMP3}
	body, err := c.do(ctx, http.MethodPost, acc, false, "/message/download", payload, c.mediaHTTPClient)
	if err != nil {
		return DownloadResponse{}, err
	}
	var out DownloadResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return DownloadResponse{}, fmt.Errorf("decoding message/download response: %w", err)
	}
	return out, nil
}

// FetchMedia downloads the file at a UAZAPI-issued CDN URL (from DownloadResponse.FileURL), capped at maxBytes.
func (c *Client) FetchMedia(ctx context.Context, fileURL string, maxBytes int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.mediaHTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &APIError{StatusCode: resp.StatusCode, Body: "media download failed"}
	}
	var reader io.Reader = resp.Body
	if maxBytes > 0 {
		reader = io.LimitReader(resp.Body, maxBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return nil, "", &APIError{StatusCode: http.StatusRequestEntityTooLarge, Body: "media exceeds configured size limit"}
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// MarkRead marks the given message IDs as read.
func (c *Client) MarkRead(ctx context.Context, acc Account, messageIDs []string) error {
	_, err := c.do(ctx, http.MethodPost, acc, false, "/message/markread", map[string]any{"id": messageIDs}, c.httpClient)
	return err
}

func (c *Client) do(ctx context.Context, method string, acc Account, useAdminToken bool, path string, payload any, httpClient *http.Client) ([]byte, error) {
	if acc.BaseURL == "" {
		return nil, fmt.Errorf("uazapi: base_url is not configured")
	}
	var reqBody io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	endpoint := strings.TrimRight(acc.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if useAdminToken {
		req.Header.Set("admintoken", acc.AdminToken)
	} else {
		req.Header.Set("token", acc.InstanceToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uazapi request to %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading uazapi response from %s: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(body)}
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				apiErr.RetryAfter = time.Duration(secs) * time.Second
			}
		}
		apiErr.ProviderCode = extractProviderCode(body)
		return nil, apiErr
	}

	return body, nil
}

// extractProviderCode best-effort pulls WhatsApp's own error code (e.g. 463) out of an error body.
func extractProviderCode(body []byte) int {
	var probe struct {
		ProviderCode int `json:"provider_code"`
		Details      struct {
			ProviderCode int `json:"provider_code"`
		} `json:"details"`
	}
	if json.Unmarshal(body, &probe) != nil {
		return 0
	}
	if probe.ProviderCode != 0 {
		return probe.ProviderCode
	}
	return probe.Details.ProviderCode
}
