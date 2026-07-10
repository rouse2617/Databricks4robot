package models

import "time"

// TagRegistryEntry is a persisted managed tag definition (CYB-3246 Phase 2).
// It mirrors config.TagDef plus the key and audit columns; the DB is the
// runtime source of truth for managed tags, while unregistered keys remain
// open-vocabulary free-form string tags (Phase 1) and are not stored here.
type TagRegistryEntry struct {
	Key         string    `json:"key"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // "enum" | "string"
	Values      []string  `json:"values"`
	MaxLength   int       `json:"max_length"`
	Propagation string    `json:"propagation"` // "none" | "descendants"
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Managed is true for DB-backed (editable) definitions and false for
	// read-only YAML-baseline entries. Computed by the list handler; not stored.
	Managed bool `json:"managed"`
}
