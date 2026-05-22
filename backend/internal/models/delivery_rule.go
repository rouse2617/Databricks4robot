package models

import (
	"encoding/json"
	"time"
)

// DeliveryRule is a declarative gate evaluated before delivery commit (CYB-1020).
type DeliveryRule struct {
	RuleID       string          `json:"rule_id"`
	Name         string          `json:"name"`
	Owner        string          `json:"owner"`
	CustomerID   string          `json:"customer_id,omitempty"`
	QueryDSL     json.RawMessage `json:"query_dsl"`
	DSLVersion   string          `json:"dsl_version"`
	EnforceMode  string          `json:"enforce_mode"`
	RatingScope  string          `json:"rating_scope"`
	IsActive     bool            `json:"is_active"`
	Version      int64           `json:"version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}
