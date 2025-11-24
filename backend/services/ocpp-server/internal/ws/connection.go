package ws

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// MessageProcessor handles raw OCPP messages.
type MessageProcessor interface {
	Process(ctx context.Context, stationID string, raw []byte) ([]byte, error)
}

// Connection represents active station WebSocket connection.
type Connection struct {
	stationID    string
	ws           *websocket.Conn
	send         chan []byte
	logger       *zap.Logger
	processor    MessageProcessor
	writeTimeout time.Duration
	onClose      func(stationID string)
}

// NewConnection builds connection wrapper.
func NewConnection(stationID string, ws *websocket.Conn, processor MessageProcessor, writeTimeout time.Duration, logger *zap.Logger, onClose func(string)) *Connection {
	return &Connection{
		stationID:    stationID,
		ws:           ws,
		send:         make(chan []byte, 16),
		logger:       logger,
		processor:    processor,
		writeTimeout: writeTimeout,
		onClose:      onClose,
	}
}

// StationID returns identifier.
func (c *Connection) StationID() string {
	return c.stationID
}

// Start launches read/write pumps.
func (c *Connection) Start(ctx context.Context) {
	go c.writePump(ctx)
	c.readPump(ctx)
}

func (c *Connection) readPump(ctx context.Context) {
	defer c.cleanup()
	c.ws.SetReadLimit(1024 * 1024)
	c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := c.ws.ReadMessage()
		if err != nil {
			c.logger.Info("connection read closed", zap.String("station_id", c.stationID), zap.Error(err))
			return
		}

		response, err := c.processor.Process(ctx, c.stationID, message)
		if err != nil {
			c.logger.Warn("failed to process message", zap.String("station_id", c.stationID), zap.Error(err))
			continue
		}
		if response != nil {
			c.Send(response)
		}
	}
}

func (c *Connection) writePump(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				_ = c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}

// Send enqueues a message for writing.
func (c *Connection) Send(msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Warn("attempted to send on closed channel", zap.String("station_id", c.stationID))
		}
	}()
	select {
	case c.send <- msg:
	default:
		c.logger.Warn("dropping outgoing message, buffer full", zap.String("station_id", c.stationID))
	}
}

// Ping sends ping.
func (c *Connection) Ping() error {
	return c.write(websocket.PingMessage, []byte("ping"))
}

func (c *Connection) write(messageType int, data []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	return c.ws.WriteMessage(messageType, data)
}

func (c *Connection) cleanup() {
	close(c.send)
	_ = c.ws.Close()
	if c.onClose != nil {
		c.onClose(c.stationID)
	}
}
