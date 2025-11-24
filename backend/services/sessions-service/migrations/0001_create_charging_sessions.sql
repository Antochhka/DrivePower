CREATE TABLE IF NOT EXISTS stations (
    id TEXT PRIMARY KEY,
    name TEXT,
    location TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS charging_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    station_id TEXT NOT NULL,
    connector_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    energy_kwh DOUBLE PRECISION DEFAULT 0,
    transaction_id TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_station FOREIGN KEY (station_id) REFERENCES stations(id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON charging_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON charging_sessions(status);

