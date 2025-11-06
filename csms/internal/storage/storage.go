package storage

import (
	"context"
	"time"
)

type StationBootInfo struct {
	StationID string
	Vendor    string
	Model     string
	Reason    string
	Time      time.Time
}

type StationRepository interface {
	UpsertBoot(ctx context.Context, info StationBootInfo) error
	UpdateLastSeen(ctx context.Context, stationID string, ts time.Time) error
}
