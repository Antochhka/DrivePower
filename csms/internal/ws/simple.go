package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

type Conn struct {
	conn        net.Conn
	rw          *bufio.ReadWriter
	subprotocol string
	mu          sync.Mutex
}

type Upgrader struct {
	Subprotocols []string
	CheckOrigin  func(*http.Request) bool
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*Conn, error) {
	if u.CheckOrigin != nil && !u.CheckOrigin(r) {
		return nil, errors.New("websocket: origin not allowed")
	}

	if !headerContainsToken(r.Header, "Connection", "Upgrade") || !headerContainsToken(r.Header, "Upgrade", "websocket") {
		return nil, errors.New("websocket: not a websocket handshake")
	}

	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return nil, errors.New("websocket: missing Sec-WebSocket-Key")
	}

	var chosenSubprotocol string
	if len(u.Subprotocols) > 0 {
		offered := parseSubprotocols(r.Header.Get("Sec-WebSocket-Protocol"))
		for _, sp := range u.Subprotocols {
			for _, offer := range offered {
				if offer == sp {
					chosenSubprotocol = sp
					break
				}
			}
			if chosenSubprotocol != "" {
				break
			}
		}
		if chosenSubprotocol == "" {
			return nil, errors.New("websocket: required subprotocol not offered")
		}
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("websocket: response writer does not support hijacking")
	}

	netConn, buf, err := hj.Hijack()
	if err != nil {
		return nil, fmt.Errorf("websocket: hijack failed: %w", err)
	}

	accept := computeAcceptKey(key)

	if responseHeader == nil {
		responseHeader = http.Header{}
	}
	responseHeader.Set("Upgrade", "websocket")
	responseHeader.Set("Connection", "Upgrade")
	responseHeader.Set("Sec-WebSocket-Accept", accept)
	if chosenSubprotocol != "" {
		responseHeader.Set("Sec-WebSocket-Protocol", chosenSubprotocol)
	}

	if err := writeHandshake(buf.Writer, responseHeader); err != nil {
		netConn.Close()
		return nil, err
	}

	return &Conn{
		conn:        netConn,
		rw:          bufio.NewReadWriter(buf.Reader, buf.Writer),
		subprotocol: chosenSubprotocol,
	}, nil
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) Subprotocol() string {
	return c.subprotocol
}

func (c *Conn) ReadMessage() (int, []byte, error) {
	for {
		first, err := c.readByte()
		if err != nil {
			return 0, nil, err
		}
		fin := first&0x80 != 0
		opcode := first & 0x0F

		second, err := c.readByte()
		if err != nil {
			return 0, nil, err
		}
		masked := second&0x80 != 0
		if !masked {
			return 0, nil, errors.New("websocket: received unmasked frame from client")
		}
		payloadLen := int(second & 0x7F)
		switch payloadLen {
		case 126:
			buf := make([]byte, 2)
			if _, err := io.ReadFull(c.rw, buf); err != nil {
				return 0, nil, err
			}
			payloadLen = int(binary.BigEndian.Uint16(buf))
		case 127:
			buf := make([]byte, 8)
			if _, err := io.ReadFull(c.rw, buf); err != nil {
				return 0, nil, err
			}
			l := binary.BigEndian.Uint64(buf)
			if l > uint64(int(^uint(0)>>1)) {
				return 0, nil, errors.New("websocket: payload too large")
			}
			payloadLen = int(l)
		}

		mask := make([]byte, 4)
		if _, err := io.ReadFull(c.rw, mask); err != nil {
			return 0, nil, err
		}

		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(c.rw, payload); err != nil {
			return 0, nil, err
		}
		for i := 0; i < payloadLen; i++ {
			payload[i] ^= mask[i%4]
		}

		if !fin {
			return 0, nil, errors.New("websocket: fragmented frames not supported")
		}

		switch opcode {
		case TextMessage, BinaryMessage:
			return int(opcode), payload, nil
		case CloseMessage:
			return int(opcode), payload, io.EOF
		case PingMessage:
			if err := c.WriteControl(PongMessage, payload, time.Now().Add(time.Second)); err != nil {
				return 0, nil, err
			}
			continue
		case PongMessage:
			continue
		default:
			return 0, nil, fmt.Errorf("websocket: unsupported opcode %d", opcode)
		}
	}
}

func (c *Conn) WriteMessage(messageType int, data []byte) error {
	var opcode byte
	switch messageType {
	case TextMessage:
		opcode = TextMessage
	case BinaryMessage:
		opcode = BinaryMessage
	case CloseMessage:
		opcode = CloseMessage
	default:
		return fmt.Errorf("websocket: unsupported message type %d", messageType)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	header := []byte{0x80 | opcode}
	payloadLen := len(data)
	switch {
	case payloadLen < 126:
		header = append(header, byte(payloadLen))
	case payloadLen < 65536:
		header = append(header, 126)
		extended := make([]byte, 2)
		binary.BigEndian.PutUint16(extended, uint16(payloadLen))
		header = append(header, extended...)
	default:
		header = append(header, 127)
		extended := make([]byte, 8)
		binary.BigEndian.PutUint64(extended, uint64(payloadLen))
		header = append(header, extended...)
	}

	if _, err := c.rw.Write(header); err != nil {
		return err
	}
	if _, err := c.rw.Write(data); err != nil {
		return err
	}
	return c.rw.Flush()
}

func (c *Conn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	err := c.WriteMessage(messageType, data)
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return err
	}
	// Reset deadline.
	_ = c.conn.SetWriteDeadline(time.Time{})
	return err
}

func FormatCloseMessage(code int, text string) []byte {
	payload := make([]byte, 2+len(text))
	binary.BigEndian.PutUint16(payload, uint16(code))
	copy(payload[2:], text)
	return payload
}

func (c *Conn) readByte() (byte, error) {
	b, err := c.rw.ReadByte()
	if err != nil {
		return 0, err
	}
	return b, nil
}

func headerContainsToken(h http.Header, key, token string) bool {
	values := h[http.CanonicalHeaderKey(key)]
	if len(values) == 0 {
		values = []string{h.Get(key)}
	}
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func parseSubprotocols(header string) []string {
	if header == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func computeAcceptKey(key string) string {
	const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.Sum([]byte(key + magicGUID))
	return base64.StdEncoding.EncodeToString(h[:])
}

func writeHandshake(w *bufio.Writer, headers http.Header) error {
	if _, err := w.WriteString("HTTP/1.1 101 Switching Protocols\r\n"); err != nil {
		return err
	}
	for k, vals := range headers {
		for _, v := range vals {
			if _, err := w.WriteString(k + ": " + v + "\r\n"); err != nil {
				return err
			}
		}
	}
	if _, err := w.WriteString("\r\n"); err != nil {
		return err
	}
	return w.Flush()
}
