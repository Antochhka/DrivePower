CREATE TABLE IF NOT EXISTS telemetry_data (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL,
    station_id TEXT NOT NULL,
    connector_id INTEGER NOT NULL,
    meter_value DOUBLE PRECISION NOT NULL,
    unit TEXT NOT NULL DEFAULT 'Wh',
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telemetry_session_id ON telemetry_data(session_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_recorded_at ON telemetry_data(recorded_at);

CREATE MATERIALIZED VIEW IF NOT EXISTS session_energy_view AS
SELECT
    session_id,
    COALESCE(MAX(meter_value) - MIN(meter_value), 0) AS total_energy_kwh
FROM telemetry_data
GROUP BY session_id;

