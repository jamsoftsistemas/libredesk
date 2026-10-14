package uazapi

import "testing"

func TestParsePayloadMessagesEvent(t *testing.T) {
	body := []byte(`{
		"EventType": "messages",
		"owner": "5511888888888",
		"message": {
			"id": "r1a2b3c4",
			"messageid": "ABCD1234",
			"chatid": "5511999999999@s.whatsapp.net",
			"sender": "5511999999999@s.whatsapp.net",
			"senderName": "Jane",
			"fromMe": false,
			"wasSentByApi": false,
			"isGroup": false,
			"messageType": "Conversation",
			"text": "hello there"
		}
	}`)

	event, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ev, ok := event.(*MessagesEvent)
	if !ok {
		t.Fatalf("expected *MessagesEvent, got %T", event)
	}
	if ev.Message.MessageID != "ABCD1234" || ev.Message.Text != "hello there" {
		t.Errorf("unexpected message: %+v", ev.Message)
	}
}

func TestParsePayloadMessagesUpdateEvent(t *testing.T) {
	body := []byte(`{
		"EventType": "messages_update",
		"type": "ReadReceipt",
		"state": "Read",
		"event": {
			"chatid": "5511999999999@s.whatsapp.net",
			"MessageIDs": ["ABCD1234", "EFGH5678"],
			"Timestamp": 1700000000
		}
	}`)

	event, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ev, ok := event.(*MessagesUpdateEvent)
	if !ok {
		t.Fatalf("expected *MessagesUpdateEvent, got %T", event)
	}
	if len(ev.Event.MessageIDs) != 2 || ev.State != StateRead {
		t.Errorf("unexpected event: %+v", ev)
	}
}

func TestParsePayloadConnectionEvent(t *testing.T) {
	body := []byte(`{
		"EventType": "connection",
		"instance": {"name": "test", "status": "disconnected", "lastDisconnectReason": "logged out"}
	}`)

	event, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ev, ok := event.(*ConnectionEvent)
	if !ok {
		t.Fatalf("expected *ConnectionEvent, got %T", event)
	}
	if ev.Instance.Status != "disconnected" {
		t.Errorf("unexpected instance status: %+v", ev.Instance)
	}
}

func TestParsePayloadUnknownEventType(t *testing.T) {
	if _, err := ParsePayload([]byte(`{"EventType": "sender"}`)); err == nil {
		t.Fatal("expected an error for unknown event type")
	}
}

func TestDeliveryRankOrdering(t *testing.T) {
	if DeliveryRank(StateDelivered) >= DeliveryRank(StateRead) {
		t.Error("expected Delivered to rank below Read")
	}
	if DeliveryRank(StateRead) >= DeliveryRank(StatePlayed) {
		t.Error("expected Read to rank below Played")
	}
	if DeliveryRank("unknown") != 0 {
		t.Error("expected unknown state to rank 0")
	}
}
