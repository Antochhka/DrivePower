package ocpp

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type CommandStatus string

var idGenerator = generateID

const (
	CommandStatusQueued   CommandStatus = "queued"
	CommandStatusPending  CommandStatus = "pending"
	CommandStatusAccepted CommandStatus = "accepted"
	CommandStatusRejected CommandStatus = "rejected"
	CommandStatusFailed   CommandStatus = "failed"
	CommandStatusTimeout  CommandStatus = "timeout"
)

type CommandResult struct {
	CommandID  string
	MessageID  string
	Status     CommandStatus
	Attempts   int
	Payload    map[string]any
	Err        error
	OccurredAt time.Time
	StationID  string
	Action     string
}

type CommandSnapshot struct {
	ID            string         `json:"id"`
	StationID     string         `json:"stationId"`
	Action        string         `json:"action"`
	Status        CommandStatus  `json:"status"`
	Attempts      int            `json:"attempts"`
	MaxAttempts   int            `json:"maxAttempts"`
	LastMessageID string         `json:"lastMessageId"`
	LastError     string         `json:"lastError,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	Payload       map[string]any `json:"payload"`
	LastResponse  map[string]any `json:"lastResponse,omitempty"`
}

type CommandCallback func(CommandResult)

type CommandManagerConfig struct {
	Timeout     time.Duration
	MaxAttempts int
	Logger      *log.Logger
}

type Command struct {
	mu            sync.Mutex
	id            string
	stationID     string
	action        string
	payload       map[string]any
	status        CommandStatus
	attempts      int
	maxAttempts   int
	timeout       time.Duration
	createdAt     time.Time
	updatedAt     time.Time
	lastError     string
	lastMessageID string
	lastResponse  map[string]any
	timer         *time.Timer
	callback      CommandCallback
}

func newCommand(stationID, action string, payload map[string]any, timeout time.Duration, maxAttempts int) *Command {
	now := time.Now().UTC()
	return &Command{
		id:          idGenerator(),
		stationID:   stationID,
		action:      action,
		payload:     payload,
		status:      CommandStatusQueued,
		attempts:    0,
		maxAttempts: maxAttempts,
		timeout:     timeout,
		createdAt:   now,
		updatedAt:   now,
	}
}

func (c *Command) snapshot() CommandSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	return CommandSnapshot{
		ID:            c.id,
		StationID:     c.stationID,
		Action:        c.action,
		Status:        c.status,
		Attempts:      c.attempts,
		MaxAttempts:   c.maxAttempts,
		LastMessageID: c.lastMessageID,
		LastError:     c.lastError,
		CreatedAt:     c.createdAt,
		UpdatedAt:     c.updatedAt,
		Payload:       cloneMap(c.payload),
		LastResponse:  cloneMap(c.lastResponse),
	}
}

func (c *Command) updateStatus(status CommandStatus, messageID string, response map[string]any, errMsg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = status
	if messageID != "" {
		c.lastMessageID = messageID
	}
	if response != nil {
		c.lastResponse = response
	}
	c.lastError = errMsg
	c.updatedAt = time.Now().UTC()
}

func (c *Command) markSent(messageID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = CommandStatusPending
	c.lastMessageID = messageID
	c.attempts++
	c.updatedAt = time.Now().UTC()
}

func (c *Command) resetForRetry() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = CommandStatusQueued
	c.lastMessageID = ""
	c.lastResponse = nil
	c.updatedAt = time.Now().UTC()
}

func (c *Command) setTimer(timer *time.Timer) {
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
	}
	c.timer = timer
	c.mu.Unlock()
}

func (c *Command) setCallback(cb CommandCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callback = cb
}

func (c *Command) getCallback() CommandCallback {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.callback
}

func (c *Command) stopTimer() {
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}

func (c *Command) attemptsInfo() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.attempts, c.maxAttempts
}

func (c *Command) actionAndPayload() (string, map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	action := c.action
	payloadCopy := make(map[string]any, len(c.payload))
	for k, v := range c.payload {
		payloadCopy[k] = v
	}
	return action, payloadCopy
}

type wsConn interface {
	WriteJSON(v any) error
	Close() error
}

type stationSession struct {
	stationID string
	manager   *CommandManager

	mu      sync.Mutex
	conn    wsConn
	queue   []*Command
	pending map[string]*Command
}

type CommandManager struct {
	mu       sync.Mutex
	sessions map[string]*stationSession
	commands map[string]*Command
	timeout  time.Duration
	attempts int
	logger   *log.Logger
}

func NewCommandManager(cfg CommandManagerConfig) *CommandManager {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	attempts := cfg.MaxAttempts
	if attempts <= 0 {
		attempts = 3
	}
	return &CommandManager{
		sessions: make(map[string]*stationSession),
		commands: make(map[string]*Command),
		timeout:  timeout,
		attempts: attempts,
		logger:   cfg.Logger,
	}
}

func (m *CommandManager) logf(format string, args ...any) {
	if m.logger != nil {
		m.logger.Printf(format, args...)
	}
}

func (m *CommandManager) getOrCreateSessionLocked(stationID string) *stationSession {
	sess, ok := m.sessions[stationID]
	if !ok {
		sess = &stationSession{
			stationID: stationID,
			manager:   m,
			queue:     make([]*Command, 0),
			pending:   make(map[string]*Command),
		}
		m.sessions[stationID] = sess
	}
	return sess
}

func (m *CommandManager) AttachConnection(stationID string, conn wsConn) {
	m.mu.Lock()
	sess := m.getOrCreateSessionLocked(stationID)
	sess.mu.Lock()
	oldConn := sess.conn
	sess.conn = conn
	sess.mu.Unlock()
	m.mu.Unlock()

	if oldConn != nil && oldConn != conn {
		_ = oldConn.Close()
	}

	sess.flushQueue()
}

func (m *CommandManager) DetachConnection(stationID string, conn wsConn) {
	m.mu.Lock()
	sess, ok := m.sessions[stationID]
	if !ok {
		m.mu.Unlock()
		return
	}
	sess.mu.Lock()
	if sess.conn == conn {
		sess.conn = nil
	}
	pending := make([]*Command, 0, len(sess.pending))
	for _, cmd := range sess.pending {
		pending = append(pending, cmd)
	}
	sess.pending = make(map[string]*Command)
	sess.mu.Unlock()
	m.mu.Unlock()

	if len(pending) == 0 {
		return
	}

	for _, cmd := range pending {
		if cmd == nil {
			continue
		}
		cmd.stopTimer()
		cmd.resetForRetry()
		cmd.updateStatus(CommandStatusQueued, "", nil, "connection lost")
		sess.requeueFront(cmd)
	}
}

func (m *CommandManager) EnqueueCommand(stationID, action string, payload map[string]any, cb CommandCallback) (CommandSnapshot, error) {
	stationID = strings.TrimSpace(stationID)
	action = strings.TrimSpace(action)
	if stationID == "" {
		return CommandSnapshot{}, errors.New("station id is required")
	}
	if action == "" {
		return CommandSnapshot{}, errors.New("action is required")
	}
	if payload == nil {
		payload = make(map[string]any)
	}

	cmd := newCommand(stationID, action, payload, m.timeout, m.attempts)
	if cb != nil {
		cmd.setCallback(cb)
	}

	m.mu.Lock()
	m.commands[cmd.id] = cmd
	sess := m.getOrCreateSessionLocked(stationID)
	m.mu.Unlock()

	sess.enqueueCommand(cmd)
	m.logf("command queued: station=%s action=%s commandId=%s", stationID, action, cmd.id)

	return cmd.snapshot(), nil
}

func (m *CommandManager) GetCommandSnapshot(commandID string) (CommandSnapshot, bool) {
	m.mu.Lock()
	cmd, ok := m.commands[commandID]
	m.mu.Unlock()
	if !ok {
		return CommandSnapshot{}, false
	}
	return cmd.snapshot(), true
}

func (m *CommandManager) HandleCallResult(stationID, messageID string, payload map[string]any) {
	sess := m.getSession(stationID)
	if sess == nil {
		m.logf("call result: no session for station=%s messageId=%s", stationID, messageID)
		return
	}
	cmd := sess.takePending(messageID)
	if cmd == nil {
		m.logf("call result: no command pending for station=%s messageId=%s", stationID, messageID)
		return
	}
	cmd.stopTimer()

	status := strings.TrimSpace(getStatus(payload))
	var finalStatus CommandStatus
	var errMsg string
	var cbErr error
	switch strings.ToLower(status) {
	case "accepted":
		finalStatus = CommandStatusAccepted
	case "rejected":
		finalStatus = CommandStatusRejected
	case "":
		finalStatus = CommandStatusAccepted
		status = "Accepted"
	default:
		finalStatus = CommandStatusFailed
		errMsg = fmt.Sprintf("unexpected status: %s", status)
		cbErr = errors.New(errMsg)
	}

	cmd.updateStatus(finalStatus, messageID, payload, errMsg)
	snap := cmd.snapshot()
	m.logf("command completed: station=%s action=%s commandId=%s status=%s attempts=%d", stationID, snap.Action, snap.ID, finalStatus, snap.Attempts)

	if cb := cmd.getCallback(); cb != nil {
		result := CommandResult{
			CommandID:  snap.ID,
			MessageID:  messageID,
			Status:     finalStatus,
			Attempts:   snap.Attempts,
			Payload:    payload,
			Err:        cbErr,
			OccurredAt: time.Now().UTC(),
			StationID:  stationID,
			Action:     snap.Action,
		}
		go cb(result)
	}

	sess.flushQueue()
}

func (m *CommandManager) HandleCallError(stationID, messageID, errorCode, description string, details map[string]any) {
	sess := m.getSession(stationID)
	if sess == nil {
		m.logf("call error: no session for station=%s messageId=%s", stationID, messageID)
		return
	}
	cmd := sess.takePending(messageID)
	if cmd == nil {
		m.logf("call error: no command pending for station=%s messageId=%s", stationID, messageID)
		return
	}
	cmd.stopTimer()

	errMsg := fmt.Sprintf("%s: %s", errorCode, description)
	cmd.updateStatus(CommandStatusFailed, messageID, details, errMsg)
	snap := cmd.snapshot()
	m.logf("command failed: station=%s action=%s commandId=%s error=%s", stationID, snap.Action, snap.ID, errMsg)

	if cb := cmd.getCallback(); cb != nil {
		result := CommandResult{
			CommandID:  snap.ID,
			MessageID:  messageID,
			Status:     CommandStatusFailed,
			Attempts:   snap.Attempts,
			Payload:    details,
			Err:        errors.New(errMsg),
			OccurredAt: time.Now().UTC(),
			StationID:  stationID,
			Action:     snap.Action,
		}
		go cb(result)
	}

	sess.flushQueue()
}

func (m *CommandManager) handleTimeout(stationID, messageID string) {
	sess := m.getSession(stationID)
	if sess == nil {
		return
	}

	cmd := sess.takePending(messageID)
	if cmd == nil {
		return
	}

	cmd.stopTimer()
	cmd.updateStatus(CommandStatusQueued, "", nil, "timeout waiting for response")
	attempts, maxAttempts := cmd.attemptsInfo()

	if attempts >= maxAttempts {
		cmd.updateStatus(CommandStatusTimeout, messageID, nil, "maximum attempts reached")
		snap := cmd.snapshot()
		if cb := cmd.getCallback(); cb != nil {
			result := CommandResult{
				CommandID:  snap.ID,
				MessageID:  messageID,
				Status:     CommandStatusTimeout,
				Attempts:   snap.Attempts,
				Payload:    nil,
				Err:        fmt.Errorf("command timeout after %d attempts", snap.Attempts),
				OccurredAt: time.Now().UTC(),
				StationID:  stationID,
				Action:     snap.Action,
			}
			go cb(result)
		}
		return
	}

	cmd.resetForRetry()
	sess.requeueFront(cmd)
	nextAttempt := attempts + 1
	snap := cmd.snapshot()
	m.logf("command timeout: station=%s action=%s commandId=%s retry=%d/%d", stationID, snap.Action, snap.ID, nextAttempt, maxAttempts)
}

func (m *CommandManager) getSession(stationID string) *stationSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[stationID]
}

func (s *stationSession) enqueueCommand(cmd *Command) {
	s.mu.Lock()
	s.queue = append(s.queue, cmd)
	s.mu.Unlock()
	s.flushQueue()
}

func (s *stationSession) flushQueue() {
	for {
		cmd, conn := s.nextCommand()
		if cmd == nil || conn == nil {
			return
		}
		if err := s.sendCommand(conn, cmd); err != nil {
			s.handleSendError(conn, cmd, err)
			return
		}
	}
}

func (s *stationSession) nextCommand() (*Command, wsConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil, nil
	}
	if len(s.pending) > 0 {
		return nil, nil
	}
	if len(s.queue) == 0 {
		return nil, nil
	}
	cmd := s.queue[0]
	s.queue = s.queue[1:]
	return cmd, s.conn
}

func (s *stationSession) sendCommand(conn wsConn, cmd *Command) error {
	messageID := idGenerator()
	action, payloadData := cmd.actionAndPayload()
	payload := []any{float64(2), messageID, action, payloadData}
	if err := conn.WriteJSON(payload); err != nil {
		return err
	}

	cmd.markSent(messageID)
	timer := time.AfterFunc(cmd.timeout, func() {
		s.manager.handleTimeout(s.stationID, messageID)
	})
	cmd.setTimer(timer)

	s.mu.Lock()
	if s.pending == nil {
		s.pending = make(map[string]*Command)
	}
	s.pending[messageID] = cmd
	s.mu.Unlock()

	snap := cmd.snapshot()
	s.manager.logf("command sent: station=%s action=%s messageId=%s attempt=%d", s.stationID, snap.Action, messageID, snap.Attempts)
	return nil
}

func (s *stationSession) takePending(messageID string) *Command {
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd, ok := s.pending[messageID]
	if ok {
		delete(s.pending, messageID)
	}
	return cmd
}

func (s *stationSession) requeueFront(cmd *Command) {
	s.mu.Lock()
	s.queue = append([]*Command{cmd}, s.queue...)
	s.mu.Unlock()
	s.flushQueue()
}

func (s *stationSession) handleSendError(conn wsConn, cmd *Command, err error) {
	errMsg := fmt.Sprintf("send command failed: %v", err)
	s.manager.logf("send command failed: station=%s action=%s err=%v", s.stationID, cmd.snapshot().Action, err)
	cmd.stopTimer()
	cmd.updateStatus(CommandStatusQueued, "", nil, errMsg)
	cmd.resetForRetry()

	s.mu.Lock()
	if s.conn == conn {
		s.conn = nil
	}
	s.queue = append([]*Command{cmd}, s.queue...)
	s.mu.Unlock()

	_ = conn.Close()
}

func getStatus(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload["status"].(string); ok {
		return v
	}
	return ""
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
