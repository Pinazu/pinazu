package utils

import (
	"github.com/google/uuid"
)

func GenerateTraceID() string {
	if traceID, err := uuid.NewV7(); err == nil {
		return traceID.String()
	}
	return uuid.New().String() // Fallback to v4 if v7 fails
}