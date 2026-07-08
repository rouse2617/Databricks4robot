package dbschema

import "time"

// Customer — `customers` table.
type Customer struct {
	CustomerID     string     `gorm:"column:customer_id;type:text;primaryKey" json:"customer_id"`
	DisplayName    string     `gorm:"column:display_name;type:text;not null" json:"display_name"`
	LegalName      string     `gorm:"column:legal_name;type:text" json:"legal_name,omitempty"`
	Status         string     `gorm:"column:status;type:text;not null;default:'active'" json:"status"`
	Region         string     `gorm:"column:region;type:text" json:"region,omitempty"`
	SLATier        string     `gorm:"column:sla_tier;type:text;not null;default:'standard'" json:"sla_tier"`
	AccountOwner   string     `gorm:"column:account_owner;type:text" json:"account_owner,omitempty"`
	ComplianceTags string     `gorm:"column:compliance_tags;type:jsonb;not null;default:'[]'" json:"compliance_tags"`
	ExcludeTags    string     `gorm:"column:exclude_tags;type:jsonb;not null;default:'[]'" json:"exclude_tags"`
	Metadata       string     `gorm:"column:metadata;type:jsonb;not null;default:'{}'" json:"metadata"`
	Extra          string     `gorm:"column:extra;type:jsonb;not null;default:'{}'" json:"extra"`
	OnboardedAt    *time.Time `gorm:"column:onboarded_at;type:timestamptz" json:"onboarded_at,omitempty"`
	OffboardedAt   *time.Time `gorm:"column:offboarded_at;type:timestamptz" json:"offboarded_at,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (Customer) TableName() string { return "customers" }

// Delivery — `deliveries` table.
type Delivery struct {
	DeliveryID         string     `gorm:"column:delivery_id;type:uuid;primaryKey" json:"delivery_id"`
	CustomerID         string     `gorm:"column:customer_id;type:text;not null" json:"customer_id"`
	Status             string     `gorm:"column:status;type:varchar(16);not null;default:'pending'" json:"status"`
	DeliveredAt        *time.Time `gorm:"column:delivered_at;type:timestamptz" json:"delivered_at,omitempty"`
	IsDeleted          bool       `gorm:"column:is_deleted;type:boolean;not null;default:false" json:"is_deleted"`
	ContractID         string     `gorm:"column:contract_id;type:text" json:"contract_id,omitempty"`
	DeliveryType       string     `gorm:"column:delivery_type;type:text;not null;default:'asset_set'" json:"delivery_type"`
	RequestedBy        string     `gorm:"column:requested_by;type:text" json:"requested_by,omitempty"`
	ApprovedBy         string     `gorm:"column:approved_by;type:text" json:"approved_by,omitempty"`
	DeliveredBy        string     `gorm:"column:delivered_by;type:text" json:"delivered_by,omitempty"`
	ManifestURI        string     `gorm:"column:manifest_uri;type:text" json:"manifest_uri,omitempty"`
	ReplayManifestURI  string     `gorm:"column:replay_manifest_uri;type:text" json:"replay_manifest_uri,omitempty"`
	ItemCount          int64      `gorm:"column:item_count;type:bigint;not null;default:0" json:"item_count"`
	TotalSizeBytes     *int64     `gorm:"column:total_size_bytes;type:bigint" json:"total_size_bytes,omitempty"`
	CompletedAt        *time.Time `gorm:"column:completed_at;type:timestamptz" json:"completed_at,omitempty"`
	Metadata           string     `gorm:"column:metadata;type:jsonb;not null;default:'{}'" json:"metadata"`
	TenantID           string     `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID          string     `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	CreatedAt          time.Time  `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
	Version            int64      `gorm:"column:version;type:bigint;not null;default:1" json:"version"`
}

func (Delivery) TableName() string { return "deliveries" }

// DeliveryItem — `delivery_items` table.
type DeliveryItem struct {
	DeliveryID string    `gorm:"column:delivery_id;type:uuid;primaryKey" json:"delivery_id"`
	AssetID    string    `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (DeliveryItem) TableName() string { return "delivery_items" }

// DeliveryRule — `delivery_rules` table.
type DeliveryRule struct {
	RuleID       string    `gorm:"column:rule_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"rule_id"`
	Name         string    `gorm:"column:name;type:text;not null" json:"name"`
	Owner        string    `gorm:"column:owner;type:text;not null" json:"owner"`
	CustomerID   string    `gorm:"column:customer_id;type:text" json:"customer_id,omitempty"`
	QueryDSL     string    `gorm:"column:query_dsl;type:jsonb;not null" json:"query_dsl"`
	DSLVersion   string    `gorm:"column:dsl_version;type:text;not null;default:'v1'" json:"dsl_version"`
	EnforceMode  string    `gorm:"column:enforce_mode;type:text;not null;default:'block'" json:"enforce_mode"`
	RatingScope  string    `gorm:"column:rating_scope;type:text;not null;default:'current'" json:"rating_scope"`
	IsActive     bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	Version      int64     `gorm:"column:version;type:bigint;not null;default:1" json:"version"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (DeliveryRule) TableName() string { return "delivery_rules" }