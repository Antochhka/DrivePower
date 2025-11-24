package protocol

// MessageType values as per OCPP spec.
const (
	MessageTypeCall       = 2
	MessageTypeCallResult = 3
	MessageTypeCallError  = 4
)

// Actions supported by MVP.
const (
	ActionBootNotification  = "BootNotification"
	ActionStatusNotification = "StatusNotification"
	ActionStartTransaction   = "StartTransaction"
	ActionStopTransaction    = "StopTransaction"
)

// Registration status values.
const (
	RegistrationAccepted = "Accepted"
	RegistrationRejected = "Rejected"
)

// StatusNotification status values (subset).
const (
	ConnectorAvailable     = "Available"
	ConnectorUnavailable   = "Unavailable"
	ConnectorCharging      = "Charging"
	ConnectorFinishing     = "Finishing"
	ConnectorPreparing     = "Preparing"
	ConnectorFaulted       = "Faulted"
	ConnectorReserved      = "Reserved"
)

