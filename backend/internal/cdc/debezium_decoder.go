package cdc

import (
	"encoding/json"
	"fmt"
)

// DebeziumEnvelope is the small subset of a Debezium-style change message we
// need for the long-term architecture skeleton.
type DebeziumEnvelope struct {
	Payload struct {
		Op     string         `json:"op"`
		Before map[string]any `json:"before"`
		After  map[string]any `json:"after"`
		Source struct {
			Table string `json:"table"`
		} `json:"source"`
	} `json:"payload"`
}

// DecodeDebeziumMessage converts a raw Debezium-style message into the
// transport-agnostic ChangeEvent used by the consumers.
func DecodeDebeziumMessage(raw []byte, key map[string]any) (ChangeEvent, error) {
	var env DebeziumEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return ChangeEvent{}, fmt.Errorf("decode debezium message: %w", err)
	}

	op := Operation(env.Payload.Op)
	switch op {
	case OperationCreate, OperationUpdate, OperationDelete, OperationRead:
	default:
		return ChangeEvent{}, fmt.Errorf("decode debezium message: unsupported op %q", env.Payload.Op)
	}

	return ChangeEvent{
		Table:  env.Payload.Source.Table,
		Op:     op,
		Key:    key,
		Before: env.Payload.Before,
		After:  env.Payload.After,
	}, nil
}
