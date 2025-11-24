package ocpp

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Message represents parsed OCPP frame.
type Message struct {
	MessageType int
	UniqueID    string
	Action      string
	Payload     json.RawMessage
}

// Parser decodes raw JSON OCPP frames.
type Parser struct{}

// NewParser returns parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse decodes []byte into Message struct.
func (p *Parser) Parse(data []byte) (*Message, error) {
	var array []json.RawMessage
	if err := json.Unmarshal(data, &array); err != nil {
		return nil, err
	}

	if len(array) < 3 {
		return nil, errors.New("ocpp: malformed frame")
	}

	var msgType int
	if err := json.Unmarshal(array[0], &msgType); err != nil {
		return nil, err
	}

	msg := &Message{MessageType: msgType}

	switch msgType {
	case 2: // CALL
		if len(array) < 4 {
			return nil, errors.New("ocpp: incomplete CALL frame")
		}
		if err := json.Unmarshal(array[1], &msg.UniqueID); err != nil {
			return nil, fmt.Errorf("ocpp: read unique id: %w", err)
		}
		if err := json.Unmarshal(array[2], &msg.Action); err != nil {
			return nil, fmt.Errorf("ocpp: read action: %w", err)
		}
		msg.Payload = array[3]
	default:
		return nil, fmt.Errorf("ocpp: unsupported message type %d", msgType)
	}

	return msg, nil
}

// BuildCallResult builds standard CALLRESULT payload.
func BuildCallResult(uniqueID string, payload interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	frame := []interface{}{3, uniqueID, json.RawMessage(body)}
	return json.Marshal(frame)
}

// BuildCallError builds CALLERROR payload.
func BuildCallError(uniqueID, code, description string) ([]byte, error) {
	frame := []interface{}{4, uniqueID, code, description, map[string]string{}}
	return json.Marshal(frame)
}

