package db

import (
	"database/sql"

	libdb "drivepower/backend/libs/db"
)

// NewPostgres connects to Postgres using shared library helper.
func NewPostgres(dsn string) (*sql.DB, error) {
	return libdb.NewPostgresDB(dsn)
}

