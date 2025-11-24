package service

import "sync"

// ConnectorState holds minimal connector info.
type ConnectorState struct {
	Status string
}

// StationRuntimeState keeps runtime info per station.
type StationRuntimeState struct {
	Status     string
	Connectors map[int]ConnectorState
}

// StationState keeps track of in-memory station data for quick lookups.
type StationState struct {
	mu        sync.RWMutex
	stations  map[string]*StationRuntimeState
}

// NewStationState returns state store.
func NewStationState() *StationState {
	return &StationState{
		stations: make(map[string]*StationRuntimeState),
	}
}

// UpdateStation updates station status.
func (s *StationState) UpdateStation(stationID, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.stations[stationID]
	if !ok {
		state = &StationRuntimeState{Connectors: make(map[int]ConnectorState)}
		s.stations[stationID] = state
	}
	state.Status = status
}

// UpdateConnector updates connector-level status.
func (s *StationState) UpdateConnector(stationID string, connectorID int, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.stations[stationID]
	if !ok {
		state = &StationRuntimeState{Connectors: make(map[int]ConnectorState)}
		s.stations[stationID] = state
	}
	state.Connectors[connectorID] = ConnectorState{Status: status}
}

// Snapshot returns a copy of current state map.
func (s *StationState) Snapshot() map[string]StationRuntimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]StationRuntimeState, len(s.stations))
	for id, st := range s.stations {
		copyState := StationRuntimeState{
			Status:     st.Status,
			Connectors: make(map[int]ConnectorState, len(st.Connectors)),
		}
		for cid, conn := range st.Connectors {
			copyState.Connectors[cid] = conn
		}
		result[id] = copyState
	}
	return result
}

