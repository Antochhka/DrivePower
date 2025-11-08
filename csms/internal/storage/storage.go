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

type ConnectorStatusRecord struct {
	StationID         string
	EVSEID            int
	ConnectorID       int
	ConnectorStatus   string
	EVSEStatus        string
	ConnectorType     string
	ReasonCode        string
	VendorID          string
	VendorDescription string
	StatusTimestamp   time.Time
	RecordedAt        time.Time
}

type StationRepository interface {
	UpsertBoot(ctx context.Context, info StationBootInfo) error
	UpdateLastSeen(ctx context.Context, stationID string, ts time.Time) error
	UpsertConnectorStatus(ctx context.Context, status ConnectorStatusRecord) error
}
