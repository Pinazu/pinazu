package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type dummyEventMessage struct {
	Data string `json:"data"`
}

const dummyTestSubject EventSubject = "test"

func (m *dummyEventMessage) Validate() error {
	return nil
}

func (m *dummyEventMessage) Subject() EventSubject {
	return dummyTestSubject
}

func Test_NewEventErrorEvent(t *testing.T) {
	headers := &EventHeaders{
		UserID: uuid.New(),
	}
	metadata := &EventMetadata{
		Timestamp: time.Now(),
	}
	this_err := fmt.Errorf("test error")

	event := NewErrorEvent[*dummyEventMessage](headers, metadata, this_err)
	_, err := event.toByte()
	assert.NoError(t, err, "Failed to convert event to byte")
}
