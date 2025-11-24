package db

import (
	"database/sql"

	libdb "drivepower/backend/libs/db"
)

// NewPostgres returns shared DB connection.
func NewPostgres(dsn string) (*sql.DB, error) {
	return libdb.NewPostgresDB(dsn)
}

