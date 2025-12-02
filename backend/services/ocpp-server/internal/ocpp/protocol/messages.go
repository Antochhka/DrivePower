package protocol

import "time"

// BootNotificationRequest minimal subset.
type BootNotificationRequest struct {
	ChargePointVendor  string `json:"chargePointVendor"`
	ChargePointModel   string `json:"chargePointModel"`
	ChargePointSerial  string `json:"chargePointSerialNumber"`
	ChargeBoxSerial    string `json:"chargeBoxSerialNumber"`
	FirmwareVersion    string `json:"firmwareVersion"`
	StationID          string `json:"stationId"`
}

// BootNotificationResponse minimal response.
type BootNotificationResponse struct {
	CurrentTime time.Time `json:"currentTime"`
	Interval    int       `json:"interval"`
	Status      string    `json:"status"`
}

// StatusNotificationRequest payload.
type StatusNotificationRequest struct {
	ConnectorID       int       `json:"connectorId"`
	ConnectorStatus   string    `json:"status"`
	ErrorCode         string    `json:"errorCode"`
	Info              string    `json:"info"`
	Timestamp         time.Time `json:"timestamp"`
	VendorID          string    `json:"vendorId"`
	StationID         string    `json:"stationId"`
}

// StatusNotificationResponse is empty (ack).
type StatusNotificationResponse struct{}

// StartTransactionRequest payload.
type StartTransactionRequest struct {
	ConnectorID   int       `json:"connectorId"`
	IdTag         string    `json:"idTag"`
	MeterStart    int64     `json:"meterStart"`
	ReservationID int       `json:"reservationId"`
	Timestamp     time.Time `json:"timestamp"`
	StationID     string    `json:"stationId"`
	TransactionID string    `json:"transactionId"`
}

// StartTransactionResponse simplified response.
type StartTransactionResponse struct {
	TransactionID string `json:"transactionId"`
	IdTagInfo     struct {
		Status string `json:"status"`
	} `json:"idTagInfo"`
}

// StopTransactionRequest payload.
type StopTransactionRequest struct {
	TransactionID string    `json:"transactionId"`
	IdTag         string    `json:"idTag"`
	MeterStop     int64     `json:"meterStop"`
	Timestamp     time.Time `json:"timestamp"`
	Reason        string    `json:"reason"`
	StationID     string    `json:"stationId"`
	MeterStart    int64     `json:"meterStart"` // optional for energy calc
}

// MeterValuesRequest payload for telemetry.
type MeterValuesRequest struct {
	StationID     string    `json:"stationId"`
	ConnectorID   int       `json:"connectorId"`
	TransactionID string    `json:"transactionId"`
	MeterValue    float64   `json:"meterValue"`
	Timestamp     time.Time `json:"timestamp"`
}

// StopTransactionResponse ack.
type StopTransactionResponse struct{}

// HeartbeatResponse returns server time.
type HeartbeatResponse struct {
	CurrentTime time.Time `json:"currentTime"`
}
