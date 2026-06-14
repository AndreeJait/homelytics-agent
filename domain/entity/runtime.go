package entity

// RuntimeStatus reports the containerd runtime connection state.
type RuntimeStatus struct {
	Connected bool   `json:"connected"`
	Version   string `json:"version,omitempty"`
	Revision  string `json:"revision,omitempty"`
	Error     string `json:"error,omitempty"`
}
