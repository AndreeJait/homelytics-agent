package entity

import (
	"encoding/json"
	"time"
)

// AgentIdentity identifies this agent to the control plane.
type AgentIdentity struct {
	AgentID  string `json:"agent_id"`
	Hostname string `json:"hostname"`
}

// AgentHeartbeat is the payload sent to /v1/agents/heartbeat.
type AgentHeartbeat struct {
	AgentID   string              `json:"agent_id"`
	Hostname  string              `json:"hostname"`
	TSNetIP   string              `json:"tsnet_ip"`
	Version   string              `json:"version"`
	Timestamp time.Time           `json:"timestamp"`
	Runtime   RuntimeStatus       `json:"runtime"`
	Workloads []WorkloadHeartbeat `json:"workloads,omitempty"`
}

// WorkloadHeartbeat is a single workload entry inside a heartbeat.
type WorkloadHeartbeat struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Image  string `json:"image"`
}

// HeartbeatResponse is returned by the control plane after a heartbeat.
type HeartbeatResponse struct {
	Commands []Command `json:"commands"`
}

// Command is a control-plane instruction for the agent.
type Command struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
