package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultMaxOpenConns = 25
	defaultMaxIdleConns = 5
	defaultConnLifetime = time.Hour
	defaultConnIdleTime = 30 * time.Minute
	defaultPingTimeout  = 5 * time.Second
)

// NewPostgresDB creates a pgx/stdlib backed *sql.DB pool and validates the connection.
func NewPostgresDB(dsn string) (*sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("db: empty DSN")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnLifetime)
	db.SetConnMaxIdleTime(defaultConnIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
