package agents

import (
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/pinazu/core/internal/db"
	"github.com/pinazu/core/internal/service"
)

// StreamEvent interface for stream events that can be used with Service Message Generic type
type StreamEvent interface {
	service.EventMessage
	RawJSON() string
}

// AnthropicStreamEventWrapper wraps anthropic.MessageStreamEventUnion to implement StreamEvent interface
type AnthropicStreamEventWrapper struct {
	anthropic.MessageStreamEventUnion
}

// Validate implements service.EventMessage interface
func (w *AnthropicStreamEventWrapper) Validate() error {
	// Basic validation - MessageStreamEventUnion should always have a type
	if w.Type == "" {
		return fmt.Errorf("stream event type is required")
	}
	return nil
}

// Subject implements service.EventMessage interface
func (w *AnthropicStreamEventWrapper) Subject() service.EventSubject {
	return service.WebsocketResponseEventSubject
}

// RawJSON returns the raw JSON from the underlying MessageStreamEventUnion
func (w *AnthropicStreamEventWrapper) RawJSON() string {
	return w.MessageStreamEventUnion.RawJSON()
}

// NewAnthropicStreamEventWrapper creates a new wrapper for anthropic.MessageStreamEventUnion
func NewAnthropicStreamEventWrapper(event anthropic.MessageStreamEventUnion) *AnthropicStreamEventWrapper {
	return &AnthropicStreamEventWrapper{
		MessageStreamEventUnion: event,
	}
}

// ToWebsocketResponseEventMessage converts anthropic.MessageStreamEventUnion to WebsocketResponseEventMessage
func ToWebsocketResponseEventMessage(event anthropic.MessageStreamEventUnion, provider db.ProviderModel) *service.WebsocketResponseEventMessage {
	return &service.WebsocketResponseEventMessage{
		Message:      event.Message,
		Type:         event.Type,
		Delta:        event.Delta,
		Usage:        event.Usage,
		ContentBlock: event.ContentBlock,
		Index:        event.Index,
		Provider:     provider,
	}
}
