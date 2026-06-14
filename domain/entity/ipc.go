package entity

import "encoding/json"

// CommandRequest is the envelope sent by the CLI over the IPC socket.
type CommandRequest struct {
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Payload json.RawMessage `json:"payload"`
}

// CommandResponse is the envelope returned by the daemon over the IPC socket.
type CommandResponse struct {
	ID    string          `json:"id"`
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *ErrorDetail    `json:"error,omitempty"`
}

// ErrorDetail carries a safe error code and message back to the CLI.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
