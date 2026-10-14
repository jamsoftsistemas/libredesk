package uazapi

import (
	"encoding/json"
	"fmt"
)

// ParsePayload decodes a raw UAZAPI webhook delivery into its typed event, based on "EventType".
// It returns one of *MessagesEvent, *MessagesUpdateEvent or *ConnectionEvent.
func ParsePayload(body []byte) (any, error) {
	var envelope webhookEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decoding uazapi webhook envelope: %w", err)
	}

	switch envelope.EventType {
	case EventMessages:
		var ev MessagesEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			return nil, fmt.Errorf("decoding uazapi messages event: %w", err)
		}
		return &ev, nil
	case EventMessagesUpdate:
		var ev MessagesUpdateEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			return nil, fmt.Errorf("decoding uazapi messages_update event: %w", err)
		}
		return &ev, nil
	case EventConnection:
		var ev ConnectionEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			return nil, fmt.Errorf("decoding uazapi connection event: %w", err)
		}
		return &ev, nil
	default:
		return nil, fmt.Errorf("unknown uazapi event type %q", envelope.EventType)
	}
}

// Delivery states carried in MessagesUpdateEvent.State.
const (
	StateDelivered = "Delivered"
	StateRead      = "Read"
	StatePlayed    = "Played"
)

// deliveryRank orders delivery states so a status update is only ever applied forward.
var deliveryRank = map[string]int{
	StateDelivered: 1,
	StateRead:      2,
	StatePlayed:    3,
}

// DeliveryRank returns state's position in the delivery lifecycle, or 0 for an unrecognized state.
func DeliveryRank(state string) int { return deliveryRank[state] }
