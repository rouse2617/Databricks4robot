package models

import "time"

// AlgoEvent represents a row in the asset_algo_events table.
// Each row records a single algorithm status transition on an asset.
type AlgoEvent struct {
	EventID    string    `json:"event_id"`
	AssetID    string    `json:"asset_id"`
	AlgoKey    string    `json:"algo_key"`
	PrevStatus *string   `json:"prev_status"`
	NewStatus  string    `json:"new_status"`
	RunID      *string   `json:"run_id"`
	Reason     *string   `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

// AlgoStatus represents the lifecycle status of an algorithm on an asset.
// It preserves the flat-key semantics documented in
// docs/review/sql.md §4.2 for backward-compatible API hydration.
type AlgoStatus string

const (
	AlgoStatusBlocked AlgoStatus = "blocked"
	AlgoStatusPending AlgoStatus = "pending"
	AlgoStatusRunning AlgoStatus = "running"
	AlgoStatusOk      AlgoStatus = "ok"
	AlgoStatusFailed  AlgoStatus = "failed"
)

// Algo field suffix constants used by the flat-key representation.
// Key pattern: <algo>@<ver>:<field>
const (
	AlgoFieldStatus     = "status"
	AlgoFieldStartedAt  = "started_at"
	AlgoFieldFinishedAt = "finished_at"
	AlgoFieldMethod     = "method"
	AlgoFieldRunID      = "run_id"
	AlgoFieldOutputURI  = "output_uri"
	AlgoFieldReason     = "reason"
)

// ValidAlgoTransitions defines the legal state machine transitions.
// Key = current status (empty string = first time / no prior status).
// Value = list of allowed next statuses.
var ValidAlgoTransitions = map[AlgoStatus][]AlgoStatus{
	"":                {AlgoStatusBlocked, AlgoStatusPending, AlgoStatusRunning},
	AlgoStatusBlocked: {AlgoStatusPending},
	AlgoStatusPending: {AlgoStatusRunning},
	AlgoStatusRunning: {AlgoStatusOk, AlgoStatusFailed},
	AlgoStatusFailed:  {AlgoStatusPending},
	AlgoStatusOk:      {AlgoStatusPending},
}
