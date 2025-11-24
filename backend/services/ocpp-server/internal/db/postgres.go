package db

import (
	"database/sql"

	libdb "drivepower/backend/libs/db"
)

// NewPostgres reuses shared DB initializer.
func NewPostgres(dsn string) (*sql.DB, error) {
	return libdb.NewPostgresDB(dsn)
}

