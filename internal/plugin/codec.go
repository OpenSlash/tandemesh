// Package plugin loads out-of-process adapters over gRPC.
//
// The protocol is JSON over gRPC (custom codec), so a plugin does not need
// protobuf tooling: both sides exchange plain structs. A plugin service is
// /orca.plugin.v1.Runner with a single unary Run method.
package plugin

import (
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/encoding"
)

// Codec is the JSON codec used by both the loader and plugin servers.
type Codec struct{}

// Name implements encoding.Codec.
func (Codec) Name() string { return "json" }

// Marshal implements encoding.Codec.
func (Codec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal implements encoding.Codec.
func (Codec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func init() {
	encoding.RegisterCodec(Codec{})
}

// RunRequest is sent to the plugin.
type RunRequest struct {
	RunID   string            `json:"runId"`
	Repo    string            `json:"repo"`
	Branch  string            `json:"branch"`
	Workdir string            `json:"workdir"`
	Prompt  string            `json:"prompt"`
	AgentID string            `json:"agentId"`
	Env     map[string]string `json:"env,omitempty"`
}

// Event is one streamed update returned by a plugin.
type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

// RunResponse is the terminal result plus any events the plugin produced.
type RunResponse struct {
	Status  string  `json:"status"`
	Code    string  `json:"code,omitempty"`
	Message string  `json:"message,omitempty"`
	Events  []Event `json:"events,omitempty"`
}

// Validate checks the terminal status.
func (r RunResponse) Validate() error {
	switch r.Status {
	case "succeeded", "failed", "cancelled":
		return nil
	}
	return fmt.Errorf("plugin: invalid status %q", r.Status)
}
