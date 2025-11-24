package models

// StationDTO describes station data returned to clients.
type StationDTO struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Connectors map[string]interface{} `json:"connectors,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

