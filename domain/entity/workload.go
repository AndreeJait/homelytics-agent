package entity

import (
	"encoding/json"
	"time"
)

// Workload represents a deployed container managed by the agent.
type Workload struct {
	ID        string            `json:"id"`
	Image     string            `json:"image"`
	Status    string            `json:"status"`
	IPAddress string            `json:"ip_address,omitempty"`
	Ports     map[string]string `json:"ports,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// RunWorkloadRequest is the payload to deploy a new container.
type RunWorkloadRequest struct {
	ID          string            `json:"id"`
	Image       string            `json:"image"`
	Command     []string          `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Ports       map[string]string `json:"ports,omitempty"` // key=container port, value=host port
	HostNetwork bool              `json:"host_network"`
}

// WorkloadIDRequest identifies a single workload for stop/delete/status.
type WorkloadIDRequest struct {
	ID string `json:"id"`
}

// WorkloadList is returned by workload.list.
type WorkloadList struct {
	Workloads []Workload `json:"workloads"`
}

// MarshalJSON returns a stable JSON representation for IPC responses.
func (w Workload) MarshalJSON() ([]byte, error) {
	type alias Workload
	return json.Marshal(alias(w))
}
