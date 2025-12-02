use chrono::prelude::*;
use json::stringify;

// OCPP constant.
const CALL: u8 = 2;

/// Wrap a CALL message.
pub fn wrap_call(msg_id: &str, action: &str, payload: &str) -> String {
    format!("[{}, \"{}\", \"{}\", {}]", CALL, msg_id, action, payload)
}

pub fn boot_notification_payload(
    reason: &str,
    model: &str,
    vendor_name: &str,
    serial_number: Option<String>,
    station_id: &str,
) -> String {
    let mut payload = object! {
        "reason" => reason,
        "chargingStation" => object!{
            "model" => model,
            "vendorName" => vendor_name,
        },
        "stationId" => station_id,
    };

    if let Some(data) = serial_number {
        payload["chargingStation"]["serialNumber"] = data.into();
    }

    stringify(payload)
}

pub fn status_notification_payload(
    station_id: &str,
    connector_id: u8,
    status: &str,
) -> String {
    let now = Utc::now()
        .with_nanosecond(0)
        .unwrap_or_else(|| Utc::now())
        .to_rfc3339();
    let payload = object! {
        "timestamp" => now,
        "connectorStatus" => status,
        "connectorId" => connector_id,
        "stationId" => station_id,
    };

    stringify(payload)
}

pub fn heartbeat_payload() -> String {
    "{}".to_string()
}

pub fn start_transaction_payload(
    station_id: &str,
    connector_id: u8,
    transaction_id: &str,
    id_tag: &str,
    meter_start: i64,
) -> String {
    let now = Utc::now()
        .with_nanosecond(0)
        .unwrap_or_else(|| Utc::now())
        .to_rfc3339();

    let payload = object! {
        "stationId" => station_id,
        "connectorId" => connector_id,
        "transactionId" => transaction_id,
        "idTag" => id_tag,
        "meterStart" => meter_start,
        "timestamp" => now,
    };

    stringify(payload)
}

pub fn stop_transaction_payload(
    station_id: &str,
    transaction_id: &str,
    id_tag: &str,
    meter_stop: i64,
    reason: &str,
) -> String {
    let now = Utc::now()
        .with_nanosecond(0)
        .unwrap_or_else(|| Utc::now())
        .to_rfc3339();

    let payload = object! {
        "stationId" => station_id,
        "transactionId" => transaction_id,
        "idTag" => id_tag,
        "meterStop" => meter_stop,
        "timestamp" => now,
        "reason" => reason,
    };

    stringify(payload)
}
