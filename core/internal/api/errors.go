package api

import "github.com/google/uuid"

type NotFoundError struct {
	Entity string    `json:"entity"`
	ID     uuid.UUID `json:"id"`
}

func (e NotFoundError) Error() string {
	return e.Entity + " with ID " + e.ID.String() + " not found"
}
