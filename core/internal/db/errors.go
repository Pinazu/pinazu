package db

import (
	"github.com/jackc/pgx/v5/pgconn"
)

func IsConflictError(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error is a pgx error with code 23505 (unique violation)
	if pgxErr, ok := err.(*pgconn.PgError); ok && pgxErr.Code == "23505" {
		return true
	}
	return false
}
