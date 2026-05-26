package models

import "time"

// Allowed action source_type values. Mirrors actions.source_type CHECK semantics
// (validated at the usecase boundary).
const (
	ActionSourceHuman  = "human"
	ActionSourceAlgo   = "algo"
	ActionSourceRule   = "rule"
	ActionSourceSystem = "system"
)

// Action is one row in the `actions` table — a time-bounded annotation inside a
// segment asset (`mcap → seg → action`, the third layer of the business model).
//
// Invariants the persistence layer enforces:
//   - asset_id MUST reference an asset with asset_type='segment'
//   - end_ns >= start_ns (DB CHECK)
//   - actions do NOT participate in lifecycle_state / deliveries / asset list
type Action struct {
	ActionID string `json:"action_id"`
	AssetID  string `json:"asset_id"`

	StartNs     int64 `json:"start_ns"`
	EndNs       int64 `json:"end_ns"`
	ActionIndex *int  `json:"action_index,omitempty"`

	PrimaryLabel string                 `json:"primary_label,omitempty"`
	Labels       []string               `json:"labels"`
	Description  string                 `json:"description,omitempty"`
	Attrs        map[string]interface{} `json:"attrs"`

	SourceType    string   `json:"source_type"`
	SourceName    string   `json:"source_name,omitempty"`
	SourceVersion string   `json:"source_version,omitempty"`
	RunID         string   `json:"run_id,omitempty"`
	Confidence    *float64 `json:"confidence,omitempty"`
	ExternalID    string   `json:"external_id,omitempty"`
	TaskID        string   `json:"task_id,omitempty"`

	TenantID  string `json:"tenant_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`

	IsDeleted bool      `json:"is_deleted"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsValidActionSourceType reports whether s is one of the allowed source_type
// enum values.
func IsValidActionSourceType(s string) bool {
	switch s {
	case ActionSourceHuman, ActionSourceAlgo, ActionSourceRule, ActionSourceSystem:
		return true
	}
	return false
}
