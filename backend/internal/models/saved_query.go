package models

import "time"

type SavedQuery struct {
	SavedQueryID  string                 `json:"saved_query_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Resource      string                 `json:"resource"`
	SchemaVersion string                 `json:"schema_version"`
	QueryIRJSON   map[string]interface{} `json:"query_ir_json"`
	Owner         string                 `json:"owner,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}
