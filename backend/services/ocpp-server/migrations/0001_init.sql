CREATE TABLE IF NOT EXISTS charging_stations (
    id TEXT PRIMARY KEY,
    vendor TEXT,
    model TEXT,
    firmware_version TEXT,
    status TEXT NOT NULL DEFAULT 'Available',
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ocpp_messages (
    id BIGSERIAL PRIMARY KEY,
    station_id TEXT NOT NULL,
    direction TEXT NOT NULL,
    message_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

