package redisstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ActiveSession stored in redis for quick access.
type ActiveSession struct {
	SessionID     int64  `json:"session_id"`
	TransactionID string `json:"transaction_id"`
	StationID     string `json:"station_id"`
	ConnectorID   int    `json:"connector_id"`
	UserID        int64  `json:"user_id"`
}

// Store manages active session cache.
type Store struct {
	client *redis.Client
	ttl    time.Duration
}

// NewStore returns redis-backed store.
func NewStore(client *redis.Client, ttl time.Duration) *Store {
	return &Store{client: client, ttl: ttl}
}

func (s *Store) key(transactionID string) string {
	return fmt.Sprintf("sessions:active:%s", transactionID)
}

// Save caches session.
func (s *Store) Save(ctx context.Context, session ActiveSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(session.TransactionID), data, s.ttl).Err()
}

// Get returns cached session.
func (s *Store) Get(ctx context.Context, transactionID string) (*ActiveSession, error) {
	result, err := s.client.Get(ctx, s.key(transactionID)).Result()
	if err != nil {
		return nil, err
	}
	var session ActiveSession
	if err := json.Unmarshal([]byte(result), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Delete removes cached session.
func (s *Store) Delete(ctx context.Context, transactionID string) error {
	return s.client.Del(ctx, s.key(transactionID)).Err()
}

