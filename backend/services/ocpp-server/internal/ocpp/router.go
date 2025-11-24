package ocpp

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// HandlerFunc processes message payload and returns response body.
type HandlerFunc func(ctx context.Context, stationID string, payload json.RawMessage) (interface{}, error)

// Router dispatches OCPP actions to handlers.
type Router struct {
	handlers map[string]HandlerFunc
}

// NewRouter returns router.
func NewRouter() *Router {
	return &Router{handlers: make(map[string]HandlerFunc)}
}

// Register attaches handler to action.
func (r *Router) Register(action string, handler HandlerFunc) {
	r.handlers[action] = handler
}

// Route executes handler for message.
func (r *Router) Route(ctx context.Context, stationID string, msg *Message) (interface{}, error) {
	handler, ok := r.handlers[msg.Action]
	if !ok {
		return nil, fmt.Errorf("ocpp: unsupported action %s", msg.Action)
	}
	return handler(ctx, stationID, msg.Payload)
}

// Processor ties together parsing, routing, and response encoding.
type Processor struct {
	parser  *Parser
	router  *Router
	logger  *zap.Logger
	logRepo OCPPLogRepository
}

// OCPPLogRepository minimal interface.
type OCPPLogRepository interface {
	Save(ctx context.Context, stationID, direction, messageType string, payload []byte) error
}

// NewProcessor builds Processor.
func NewProcessor(parser *Parser, router *Router, logRepo OCPPLogRepository, logger *zap.Logger) *Processor {
	return &Processor{
		parser:  parser,
		router:  router,
		logRepo: logRepo,
		logger:  logger,
	}
}

// Process handles raw message and returns response frame bytes.
func (p *Processor) Process(ctx context.Context, stationID string, raw []byte) ([]byte, error) {
	msg, err := p.parser.Parse(raw)
	if err != nil {
		return nil, err
	}

	if p.logRepo != nil {
		_ = p.logRepo.Save(ctx, stationID, "incoming", msg.Action, raw)
	}

	responsePayload, err := p.router.Route(ctx, stationID, msg)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("ocpp handler failed", zap.String("action", msg.Action), zap.Error(err))
		}
		return BuildCallError(msg.UniqueID, "InternalError", err.Error())
	}

	if responsePayload == nil {
		return nil, nil
	}

	respBytes, err := BuildCallResult(msg.UniqueID, responsePayload)
	if err != nil {
		if p.logger != nil {
			p.logger.Error("encode ocpp response failed", zap.Error(err))
		}
		return nil, err
	}

	if p.logRepo != nil {
		_ = p.logRepo.Save(ctx, stationID, "outgoing", msg.Action, respBytes)
	}

	return respBytes, nil
}

// Decode convenience helper for handlers.
func Decode[T any](payload json.RawMessage) (T, error) {
	var target T
	if err := json.Unmarshal(payload, &target); err != nil {
		var zero T
		return zero, err
	}
	return target, nil
}
