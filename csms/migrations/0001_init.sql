CREATE TABLE IF NOT EXISTS stations (
    station_id TEXT PRIMARY KEY,
    vendor TEXT,
    model TEXT,
    boot_reason TEXT,
    status TEXT,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
