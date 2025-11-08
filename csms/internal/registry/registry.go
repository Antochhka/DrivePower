package registry

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// StatusUpdate описывает обновление состояния конкретного коннектора,
// которое пришло от зарядной станции в сообщении StatusNotification.
type StatusUpdate struct {
	EVSEID            int
	ConnectorID       int
	ConnectorStatus   string
	EVSEStatus        string
	ConnectorType     string
	Timestamp         time.Time
	ReasonCode        string
	VendorID          string
	VendorDescription string
}

// ConnectorStatus хранит последнюю известную информацию о коннекторе.
type ConnectorStatus struct {
	Status            string
	EVSEStatus        string
	ConnectorType     string
	ReasonCode        string
	VendorID          string
	VendorDescription string
	StatusTimestamp   time.Time
	UpdatedAt         time.Time
}

// StationSnapshot представляет снимок состояния станции в момент времени.
type StationSnapshot struct {
	StationID string
	UpdatedAt time.Time
	EVSEs     map[int]map[int]ConnectorStatus
}

// StatusEvent содержит сведения о событии изменения статуса коннектора.
type StatusEvent struct {
	StationID  string
	Update     StatusUpdate
	Previous   ConnectorStatus
	Current    ConnectorStatus
	RecordedAt time.Time
	Snapshot   StationSnapshot
}

type stationState struct {
	stationID string
	updatedAt time.Time
	evses     map[int]map[int]ConnectorStatus
}

func newStationState(stationID string) *stationState {
	return &stationState{
		stationID: stationID,
		evses:     make(map[int]map[int]ConnectorStatus),
	}
}

func (s *stationState) snapshot() StationSnapshot {
	snapshot := StationSnapshot{
		StationID: s.stationID,
		UpdatedAt: s.updatedAt,
		EVSEs:     make(map[int]map[int]ConnectorStatus, len(s.evses)),
	}

	for evseID, connectors := range s.evses {
		copied := make(map[int]ConnectorStatus, len(connectors))
		for connectorID, status := range connectors {
			copied[connectorID] = status
		}
		snapshot.EVSEs[evseID] = copied
	}

	return snapshot
}

// Registry хранит сведения о статусах коннекторов всех станций.
type Registry struct {
	mu       sync.RWMutex
	stations map[string]*stationState
}

// New создаёт новый реестр состояний.
func New() *Registry {
	return &Registry{stations: make(map[string]*stationState)}
}

// Update фиксирует новое состояние коннектора станции и возвращает событие.
//
// Параметр recordedAt задаёт момент обработки события на стороне CSMS.
func (r *Registry) Update(stationID string, update StatusUpdate, recordedAt time.Time) (StatusEvent, error) {
	if stationID == "" {
		return StatusEvent{}, fmt.Errorf("station id is required")
	}
	if update.EVSEID <= 0 {
		return StatusEvent{}, fmt.Errorf("evse id must be positive")
	}
	if update.ConnectorID <= 0 {
		return StatusEvent{}, fmt.Errorf("connector id must be positive")
	}
	if strings.TrimSpace(update.ConnectorStatus) == "" {
		return StatusEvent{}, fmt.Errorf("connector status is required")
	}

	if update.Timestamp.IsZero() {
		update.Timestamp = recordedAt
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.stations[stationID]
	if !ok {
		state = newStationState(stationID)
		r.stations[stationID] = state
	}

	connectors := state.evses[update.EVSEID]
	if connectors == nil {
		connectors = make(map[int]ConnectorStatus)
		state.evses[update.EVSEID] = connectors
	}

	previous := connectors[update.ConnectorID]
	current := ConnectorStatus{
		Status:            update.ConnectorStatus,
		EVSEStatus:        update.EVSEStatus,
		ConnectorType:     update.ConnectorType,
		ReasonCode:        update.ReasonCode,
		VendorID:          update.VendorID,
		VendorDescription: update.VendorDescription,
		StatusTimestamp:   update.Timestamp,
		UpdatedAt:         recordedAt,
	}

	connectors[update.ConnectorID] = current
	state.updatedAt = recordedAt

	event := StatusEvent{
		StationID:  stationID,
		Update:     update,
		Previous:   previous,
		Current:    current,
		RecordedAt: recordedAt,
		Snapshot:   state.snapshot(),
	}

	return event, nil
}

// Snapshot возвращает копию последнего известного состояния станции.
func (r *Registry) Snapshot(stationID string) (StationSnapshot, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, ok := r.stations[stationID]
	if !ok {
		return StationSnapshot{}, false
	}
	return state.snapshot(), true
}
