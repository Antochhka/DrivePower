package ocpp

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeConn struct {
	mu       sync.Mutex
	messages [][]any
	writeErr error
	closed   bool
}

func newFakeConn() *fakeConn {
	return &fakeConn{}
}

func (f *fakeConn) WriteJSON(v any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.writeErr != nil {
		return f.writeErr
	}

	if payload, ok := v.([]any); ok {
		copyPayload := make([]any, len(payload))
		copy(copyPayload, payload)
		f.messages = append(f.messages, copyPayload)
	}

	return nil
}

func (f *fakeConn) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}

func (f *fakeConn) setWriteErr(err error) {
	f.mu.Lock()
	f.writeErr = err
	f.mu.Unlock()
}

func (f *fakeConn) messageCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.messages)
}

func (f *fakeConn) messageAt(index int) []any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if index < 0 || index >= len(f.messages) {
		return nil
	}
	payload := make([]any, len(f.messages[index]))
	copy(payload, f.messages[index])
	return payload
}

func (f *fakeConn) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func TestCommandManagerSendAndAcknowledge(t *testing.T) {
	originalGenerator := idGenerator
	ids := []string{"cmd-1", "msg-1"}
	idGenerator = func() string {
		if len(ids) == 0 {
			return originalGenerator()
		}
		id := ids[0]
		ids = ids[1:]
		return id
	}
	t.Cleanup(func() { idGenerator = originalGenerator })

	manager := NewCommandManager(CommandManagerConfig{Timeout: time.Second, MaxAttempts: 2})

	fake := newFakeConn()
	snapshot, err := manager.EnqueueCommand("station-1", "RemoteStartTransaction", map[string]any{"connectorId": 1}, nil)
	if err != nil {
		t.Fatalf("enqueue command: %v", err)
	}
	if snapshot.Status != CommandStatusQueued {
		t.Fatalf("expected queued status, got %s", snapshot.Status)
	}

	manager.AttachConnection("station-1", fake)

	waitFor(t, 200*time.Millisecond, func() bool { return fake.messageCount() == 1 })

	snap, ok := manager.GetCommandSnapshot(snapshot.ID)
	if !ok {
		t.Fatalf("command snapshot not found")
	}
	if snap.Status != CommandStatusPending {
		t.Fatalf("expected pending status after send, got %s", snap.Status)
	}
	if snap.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", snap.Attempts)
	}

	msgPayload := fake.messageAt(0)
	if len(msgPayload) < 2 {
		t.Fatalf("unexpected payload format: %v", msgPayload)
	}
	messageID, _ := msgPayload[1].(string)
	if messageID == "" {
		t.Fatalf("missing message id in payload: %v", msgPayload)
	}

	manager.HandleCallResult("station-1", messageID, map[string]any{"status": "Accepted"})

	snap, ok = manager.GetCommandSnapshot(snapshot.ID)
	if !ok {
		t.Fatalf("command snapshot not found after ack")
	}
	if snap.Status != CommandStatusAccepted {
		t.Fatalf("expected accepted status, got %s", snap.Status)
	}
	if snap.Attempts != 1 {
		t.Fatalf("expected attempts to remain 1, got %d", snap.Attempts)
	}
	if snap.LastMessageID != messageID {
		t.Fatalf("expected last message id %s, got %s", messageID, snap.LastMessageID)
	}
}

func TestCommandManagerRetriesAndTimesOut(t *testing.T) {
	originalGenerator := idGenerator
	ids := []string{"cmd-timeout", "msg-1", "msg-2"}
	idGenerator = func() string {
		if len(ids) == 0 {
			return originalGenerator()
		}
		id := ids[0]
		ids = ids[1:]
		return id
	}
	t.Cleanup(func() { idGenerator = originalGenerator })

	timeout := 20 * time.Millisecond
	manager := NewCommandManager(CommandManagerConfig{Timeout: timeout, MaxAttempts: 2})

	fake := newFakeConn()
	snapshot, err := manager.EnqueueCommand("station-7", "RemoteStopTransaction", map[string]any{"transactionId": "42"}, nil)
	if err != nil {
		t.Fatalf("enqueue command: %v", err)
	}

	manager.AttachConnection("station-7", fake)

	waitFor(t, 200*time.Millisecond, func() bool { return fake.messageCount() == 1 })

	waitFor(t, 400*time.Millisecond, func() bool { return fake.messageCount() == 2 })

	waitFor(t, 400*time.Millisecond, func() bool {
		snap, ok := manager.GetCommandSnapshot(snapshot.ID)
		return ok && snap.Status == CommandStatusTimeout
	})

	snap, ok := manager.GetCommandSnapshot(snapshot.ID)
	if !ok {
		t.Fatalf("command snapshot not found after timeout")
	}
	if snap.Attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", snap.Attempts)
	}
	if snap.Status != CommandStatusTimeout {
		t.Fatalf("expected timeout status, got %s", snap.Status)
	}
}

func TestCommandManagerHandlesSendFailure(t *testing.T) {
	originalGenerator := idGenerator
	ids := []string{"cmd-fail", "msg-fail"}
	idGenerator = func() string {
		if len(ids) == 0 {
			return originalGenerator()
		}
		id := ids[0]
		ids = ids[1:]
		return id
	}
	t.Cleanup(func() { idGenerator = originalGenerator })

	manager := NewCommandManager(CommandManagerConfig{Timeout: time.Second, MaxAttempts: 1})

	fake := newFakeConn()
	fake.setWriteErr(errors.New("boom"))

	snapshot, err := manager.EnqueueCommand("station-9", "ChangeAvailability", map[string]any{"evseId": 3}, nil)
	if err != nil {
		t.Fatalf("enqueue command: %v", err)
	}

	manager.AttachConnection("station-9", fake)

	waitFor(t, 200*time.Millisecond, fake.isClosed)

	snap, ok := manager.GetCommandSnapshot(snapshot.ID)
	if !ok {
		t.Fatalf("command snapshot not found after send failure")
	}
	if snap.Status != CommandStatusQueued {
		t.Fatalf("expected command to remain queued, got %s", snap.Status)
	}
	if snap.Attempts != 0 {
		t.Fatalf("expected attempts to remain 0, got %d", snap.Attempts)
	}
	if !strings.Contains(snap.LastError, "send command failed") {
		t.Fatalf("expected error to mention send failure, got %s", snap.LastError)
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
