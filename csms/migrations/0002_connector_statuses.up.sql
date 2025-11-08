CREATE TABLE IF NOT EXISTS station_connector_statuses (
    station_id TEXT NOT NULL,
    evse_id INTEGER NOT NULL,
    connector_id INTEGER NOT NULL,
    connector_status TEXT NOT NULL,
    evse_status TEXT,
    connector_type TEXT,
    reason_code TEXT,
    vendor_id TEXT,
    vendor_description TEXT,
    status_timestamp TIMESTAMPTZ NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (station_id, evse_id, connector_id)
);

CREATE INDEX IF NOT EXISTS station_connector_statuses_recorded_at_idx
    ON station_connector_statuses (recorded_at DESC);
