package service

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

//go:generate go run ../../api/generate_event.go

type (
	EventSubject string
	// EventMessage is an interface represents different types of message request fields
	EventMessage interface {
		// Validate checks if the request fields are valid
		Validate() error
		Subject() EventSubject
	}

	// WebSocketEventMessage extends EventMessage for WebSocket-specific events that need user-specific subjects
	WebSocketEventMessage interface {
		EventMessage
		// SubjectWithUser returns the subject with user ID appended for WebSocket routing
		SubjectWithUser(userID uuid.UUID) EventSubject
	}

	// Generic Typed Requests
	Event[T EventMessage] struct {
		H   *EventHeaders  `json:"header"`
		Msg T              `json:"message"`
		M   *EventMetadata `json:"metadata"`
		Err *EventError    `json:"error,omitempty"`
	}

	EventMetadata struct {
		TraceID   string    `json:"trace_id,omitempty"`
		Timestamp time.Time `json:"timestamp"`
	}

	EventHeaders struct {
		UserID       uuid.UUID  `json:"user_id"`
		ThreadID     *uuid.UUID `json:"thread_id,omitempty"`
		TaskID       *string    `json:"task_id,omitempty"`
		ConnectionID *uuid.UUID `json:"connection_id,omitempty"`
	}

	EventError struct {
		Type    string `json:"type"`
		Package string `json:"package"`
		Error   string `json:"error"`
	}

	// ModelProvider represents different AI model providers
	ModelProvider string
)

// WrapError wraps a Go error into an EventError struct
func WrapError(err error) *EventError {
	if err == nil {
		return nil
	}
	reflectType := reflect.TypeOf(err)
	return &EventError{
		Type:    reflectType.Name(),
		Package: reflectType.PkgPath(),
		Error:   err.Error(),
	}
}

// NewEvent creates a new event with the given message, headers, and metadata
func NewEvent[T EventMessage](msg T, headers *EventHeaders, metadata *EventMetadata) *Event[T] {
	return &Event[T]{
		H:   headers,
		Msg: msg,
		M:   metadata,
		Err: nil,
	}
}

// NewErrorEvent creates a new error event with zero message and wrapped error
func NewErrorEvent[T EventMessage](headers *EventHeaders, metadata *EventMetadata, err error) *Event[T] {
	var zero T
	// Check if T is a pointer type
	if reflect.TypeOf(zero).Kind() == reflect.Ptr {
		zero = reflect.New(reflect.TypeOf(zero).Elem()).Interface().(T)
	} else {
		zero = reflect.Zero(reflect.TypeOf(zero)).Interface().(T)
	}
	return &Event[T]{
		H:   headers,
		Msg: zero,
		M:   metadata,
		Err: WrapError(err),
	}
}

// ParseEvent unmarshals and validates an event from JSON bytes
func ParseEvent[T EventMessage](data []byte) (*Event[T], error) {
	var req Event[T]
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}
	// Check for nil message before validation using reflection
	if reflect.ValueOf(req.Msg).IsNil() {
		// Check for error in the request
		if req.Err != nil {
			return &req, fmt.Errorf("Error Type: %s, Error: %s", req.Err.Type, req.Err.Error)
		}
		return nil, fmt.Errorf("message is nil")
	}
	if req.Err != nil {
		return &req, fmt.Errorf("Error Type: %s, Error: %s", req.Err.Type, req.Err.Error)
	}
	// Validation logic
	if err := req.Msg.Validate(); err != nil {
		return &req, fmt.Errorf("invalid message: %w", err)
	}
	return &req, nil
}

// String returns the string representation of an EventSubject
func (s EventSubject) String() string {
	return string(s)
}

// toByte marshals the event to JSON bytes
func (e *Event[T]) toByte() ([]byte, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	return data, nil
}

// Publish publishes the event to NATS using the message's subject
func (e *Event[T]) Publish(n *nats.Conn) error {
	data, err := e.toByte()
	if err != nil {
		return fmt.Errorf("failed to convert event to byte: %w", err)
	}
	err = n.Publish(e.Msg.Subject().String(), data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	return nil
}

// PublishWithUser publishes the event with user-specific subject for WebSocket events
func (e *Event[T]) PublishWithUser(n *nats.Conn, userID uuid.UUID) error {
	data, err := e.toByte()
	if err != nil {
		return fmt.Errorf("failed to convert event to byte: %w", err)
	}

	// Check if the message implements WebSocketEventMessage interface
	if wsMsg, ok := any(e.Msg).(WebSocketEventMessage); ok {
		subject := wsMsg.SubjectWithUser(userID)

		// Publish to WebSocket subject
		err = n.Publish(subject.String(), data)
		if err != nil {
			return fmt.Errorf("failed to publish to WebSocket subject: %w", err)
		}

		// Also publish to SSE subject to prevent duplication issues
		sseSubject := subject.String() + ".sse"
		err = n.Publish(sseSubject, data)
	} else {
		// Fallback to regular subject
		err = n.Publish(e.Msg.Subject().String(), data)
	}

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	return nil
}

// Respond sends the event as a response to a NATS message
func (e *Event[T]) Respond(msg *nats.Msg) error {
	data, err := e.toByte()
	if err != nil {
		return fmt.Errorf("failed to convert event to byte: %w", err)
	}
	return msg.Respond(data)
}

// Request sends a request event and waits for a response event with timeout
func Request[R, T EventMessage](nat *nats.Conn, req *Event[T], timeout time.Duration) (*Event[R], error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := nat.Request(req.Msg.Subject().String(), data, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	resp, err := ParseEvent[R](msg.Data)
	return resp, err
}
