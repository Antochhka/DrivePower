package handlers

import (
	"context"
	"encoding/json"
	"time"

	"drivepower/backend/services/ocpp-server/internal/ocpp"
	"drivepower/backend/services/ocpp-server/internal/ocpp/protocol"
)

// NewHeartbeatHandler returns ack with current time.
func NewHeartbeatHandler() ocpp.HandlerFunc {
	return func(ctx context.Context, stationID string, payload json.RawMessage) (interface{}, error) {
		_ = ctx // unused
		_ = stationID
		return protocol.HeartbeatResponse{
			CurrentTime: time.Now().UTC(),
		}, nil
	}
}
