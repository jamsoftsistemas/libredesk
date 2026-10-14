package uazapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zerodha/logf"
)

func testClient(t *testing.T, handler http.HandlerFunc) (*Client, Account, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	lo := logf.New(logf.Opts{})
	c := New(&lo)
	acc := Account{BaseURL: srv.URL, InstanceToken: "tok", AdminToken: "admintok"}
	return c, acc, srv.Close
}

func TestSendTextUsesInstanceToken(t *testing.T) {
	c, acc, closeSrv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send/text" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("token"); got != "tok" {
			t.Errorf("expected token header 'tok', got %q", got)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["number"] != "5511999999999" || body["text"] != "hello" {
			t.Errorf("unexpected body: %+v", body)
		}
		json.NewEncoder(w).Encode(SendResponse{Message: Message{MessageID: "abc123", Status: "Sent"}})
	})
	defer closeSrv()

	resp, err := c.SendText(context.Background(), acc, "5511999999999", "hello", SendOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SourceID() != "abc123" {
		t.Errorf("expected source id abc123, got %q", resp.SourceID())
	}
}

func TestSendButtonMenuBuildsURLChoice(t *testing.T) {
	c, acc, closeSrv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send/menu" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["type"] != "button" {
			t.Errorf("expected type 'button', got %v", body["type"])
		}
		choices, _ := body["choices"].([]any)
		if len(choices) != 1 || choices[0] != "Rate us|https://example.com/csat/abc" {
			t.Errorf("unexpected choices: %+v", body["choices"])
		}
		json.NewEncoder(w).Encode(SendResponse{Message: Message{MessageID: "m1"}, Response: struct {
			Status  string          `json:"status"`
			Message json.RawMessage `json:"message"`
		}{Status: "success", Message: json.RawMessage(`"Menu sent successfully"`)}})
	})
	defer closeSrv()

	resp, err := c.SendButtonMenu(context.Background(), acc, "5511999999999", "Please rate", "Rate us", "https://example.com/csat/abc", SendOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SourceID() != "m1" {
		t.Errorf("expected source id m1, got %q", resp.SourceID())
	}
}

func TestCreateInstanceUsesAdminToken(t *testing.T) {
	c, acc, closeSrv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("admintoken"); got != "admintok" {
			t.Errorf("expected admintoken header 'admintok', got %q", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"instance": Instance{ID: "inst1", Name: "test", Token: "newtok"},
		})
	})
	defer closeSrv()

	inst, err := c.CreateInstance(context.Background(), acc, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Token != "newtok" {
		t.Errorf("expected token 'newtok', got %q", inst.Token)
	}
}

func TestDoReturnsAPIErrorWithRetryAfterAndProviderCode(t *testing.T) {
	c, acc, closeSrv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"provider_code": 463})
	})
	defer closeSrv()

	_, err := c.SendText(context.Background(), acc, "123", "hi", SendOpts{})
	if err == nil {
		t.Fatal("expected an error")
	}
	var apiErr *APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if !apiErr.IsReachoutTimelock() {
		t.Errorf("expected reachout timelock, got provider code %d", apiErr.ProviderCode)
	}
	if apiErr.RetryAfter.Seconds() != 5 {
		t.Errorf("expected retry-after 5s, got %v", apiErr.RetryAfter)
	}
}

func TestStatusRateLimited(t *testing.T) {
	c, acc, closeSrv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{}`))
	})
	defer closeSrv()

	_, err := c.Status(context.Background(), acc)
	var apiErr *APIError
	if !asAPIError(err, &apiErr) || !apiErr.IsRateLimited() {
		t.Fatalf("expected rate-limited APIError, got %v", err)
	}
}

func asAPIError(err error, target **APIError) bool {
	ae, ok := err.(*APIError)
	if ok {
		*target = ae
	}
	return ok
}
