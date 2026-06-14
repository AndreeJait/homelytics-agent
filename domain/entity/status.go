package entity

// AgentStatus is a high-level health/status view of the daemon.
type AgentStatus struct {
	ServiceName          string `json:"service_name"`
	LoggedIn             bool   `json:"logged_in"`
	TSNetAuthKeyPresent  bool   `json:"tsnet_auth_key_present"`
	TSNetConnected       bool   `json:"tsnet_connected"`
	RuntimeConnected     bool   `json:"runtime_connected"`
	RuntimeVersion       string `json:"runtime_version,omitempty"`
}
