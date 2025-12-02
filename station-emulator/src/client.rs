use std::collections::HashMap;
use std::env;

use json::JsonValue;
use url;
use uuid::Uuid;
use ws::util::Token;
use ws::{CloseCode, Error, ErrorKind, Handler, Handshake, Message, Request, Result, Sender};

use crate::requests;

// Timeout events.
const HEARTBEAT: Token = Token(1);
const STOP_TRANSACTION: Token = Token(2);

// Simulation parameters.
const DEFAULT_CONNECTOR_ID: u8 = 1;
const DEFAULT_ID_TAG: &str = "TAG-001";
const DEFAULT_METER_START: i64 = 0;
const DEFAULT_METER_STOP: i64 = 150;
const DEFAULT_STOP_AFTER_MS: u64 = 5_000;

// OCPP message type ids.
const CALL: u8 = 2;
const CALLRESULT: u8 = 3;
const CALLERROR: u8 = 4;

pub struct Client {
    pub out: Sender,
    station_id: String,
    heartbeat_interval_ms: u64,
    active_transaction: Option<String>,
    sent_actions: HashMap<String, String>,
}

impl Client {
    pub fn new(sender: Sender, station_id: String) -> Self {
        Self {
            out: sender,
            station_id,
            heartbeat_interval_ms: 0,
            active_transaction: None,
            sent_actions: HashMap::new(),
        }
    }

    fn send_and_track(&mut self, action: &str, payload: String) -> Result<()> {
        let msg_id = Uuid::new_v4().to_string();
        let frame = requests::wrap_call(&msg_id, action, &payload);
        self.sent_actions.insert(msg_id.clone(), action.to_string());
        self.out.send(frame)
    }

    fn schedule_stop_transaction(&mut self) -> Result<()> {
        if self.active_transaction.is_some() {
            self.out.timeout(DEFAULT_STOP_AFTER_MS, STOP_TRANSACTION)?;
        }
        Ok(())
    }
}

impl Handler for Client {
    /// Add protocol to initial handshake request.
    fn build_request(&mut self, url: &url::Url) -> Result<Request> {
        let mut req = Request::from_url(url).unwrap();
        req.add_protocol("ocpp2.0");
        Ok(req)
    }

    fn on_open(&mut self, _: Handshake) -> Result<()> {
        println!("Opening connection, sending BootNotification...");

        let model = env::var("MODEL").unwrap_or_else(|_| "Model".to_string());
        let vendor = env::var("VENDOR_NAME").unwrap_or_else(|_| "Vendor name".to_string());
        let serial = env::var("SERIAL_NUMBER").ok();

        // BootNotification with station id included.
        let boot_payload =
            requests::boot_notification_payload("PowerUp", &model, &vendor, serial, &self.station_id);
        let boot_id = Uuid::new_v4().to_string();
        let boot_frame = requests::wrap_call(&boot_id, "BootNotification", &boot_payload);
        self.sent_actions
            .insert(boot_id.clone(), "BootNotification".to_string());
        self.out.send(boot_frame)?;

        Ok(())
    }

    fn on_message(&mut self, msg: Message) -> Result<()> {
        let parsed = json::parse(msg.as_text()?)
            .map_err(|e| Error::new(ErrorKind::Protocol, e.to_string()))?;
        let msg_type = parsed[0].as_u8().unwrap_or(0);
        let msg_id = parsed[1].to_string();

        match msg_type {
            CALLRESULT => {
                let payload: &JsonValue = &parsed[2];
                if let Some(action) = self.sent_actions.remove(&msg_id) {
                    match action.as_str() {
                        "BootNotification" => {
                            if payload["status"].to_string() == "Accepted" {
                                let interval = payload["interval"]
                                    .as_u64()
                                    .unwrap_or(30) * 1000;
                                self.heartbeat_interval_ms = interval;
                                // Heartbeat loop.
                                if self.heartbeat_interval_ms > 0 {
                                    self.out.timeout(self.heartbeat_interval_ms, HEARTBEAT)?;
                                }
                                // Notify Available.
                                let status_payload = requests::status_notification_payload(
                                    &self.station_id,
                                    DEFAULT_CONNECTOR_ID,
                                    "Available",
                                );
                                self.send_and_track("StatusNotification", status_payload)?;

                                // Start transaction.
                                let tx_id = Uuid::new_v4().to_string();
                                self.active_transaction = Some(tx_id.clone());
                                let start_payload = requests::start_transaction_payload(
                                    &self.station_id,
                                    DEFAULT_CONNECTOR_ID,
                                    &tx_id,
                                    DEFAULT_ID_TAG,
                                    DEFAULT_METER_START,
                                );
                                self.send_and_track("StartTransaction", start_payload)?;
                            }
                        }
                        "StartTransaction" => {
                            // If backend returns transactionId, store it.
                            if let Some(id) = payload["transactionId"].as_str() {
                                self.active_transaction = Some(id.to_string());
                            }
                            self.schedule_stop_transaction()?;
                        }
                        _ => {}
                    }
                }
            }
            CALLERROR => {
                let code = parsed[2].to_string();
                let description = parsed[3].to_string();
                println!("CALLERROR {} - {}", code, description);
            }
            _ => {
                // ignore CALL for now (no CSMS-initiated commands handled)
            }
        }
        Ok(())
    }

    fn on_timeout(&mut self, event: Token) -> Result<()> {
        match event {
            HEARTBEAT => {
                let payload = "{}".to_string();
                self.send_and_track("Heartbeat", payload)?;
                if self.heartbeat_interval_ms > 0 {
                    self.out.timeout(self.heartbeat_interval_ms, HEARTBEAT)?;
                }
                Ok(())
            }
            STOP_TRANSACTION => {
                if let Some(tx_id) = self.active_transaction.clone() {
                    let payload = requests::stop_transaction_payload(
                        &self.station_id,
                        &tx_id,
                        DEFAULT_ID_TAG,
                        DEFAULT_METER_STOP,
                        "Local",
                    );
                    self.send_and_track("StopTransaction", payload)?;
                    // reset
                    self.active_transaction = None;
                }
                Ok(())
            }
            _ => Err(Error::new(
                ErrorKind::Internal,
                "Invalid timeout token encountered!",
            )),
        }
    }

    fn on_close(&mut self, code: CloseCode, reason: &str) {
        println!("WebSocket closing for ({:?}) {}", code, reason);
        let _ = self.out.shutdown();
    }

    fn on_error(&mut self, err: Error) {
        println!("WebSocket error: {}", err);
        let _ = self.out.shutdown();
    }
}
