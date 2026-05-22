package models

import "time"

// Customer is a long-lived business reference row (CYB-1014).
type Customer struct {
	CustomerID     string                 `json:"customer_id"`
	DisplayName    string                 `json:"display_name"`
	LegalName      string                 `json:"legal_name,omitempty"`
	Status         string                 `json:"status"`
	Region         string                 `json:"region,omitempty"`
	SLATier        string                 `json:"sla_tier"`
	AccountOwner   string                 `json:"account_owner,omitempty"`
	ComplianceTags []interface{}          `json:"compliance_tags,omitempty"`
	ExcludeTags    []interface{}          `json:"exclude_tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
	OnboardedAt    *time.Time             `json:"onboarded_at,omitempty"`
	OffboardedAt   *time.Time             `json:"offboarded_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	RowVersion     int64                  `json:"row_version"`
}
