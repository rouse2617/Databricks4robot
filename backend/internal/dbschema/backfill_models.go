package dbschema

import "time"

// BackfillJob — `backfill_jobs` (043; altered by 054 to add new fields,
// 062 to add created_by, 064 to add notification_sent_at, 063 added
// finished_at earlier).
type BackfillJob struct {
	ID                    string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	Name                  string     `gorm:"column:name;type:text;not null" json:"name"`
	TemplateID            string     `gorm:"column:template_id;type:text;not null" json:"template_id"`
	FilterJSON            string     `gorm:"column:filter_json;type:jsonb" json:"filter_json,omitempty"`
	TotalCount            int        `gorm:"column:total_count;type:integer;not null;default:0" json:"total_count"`
	CompletedCount        int        `gorm:"column:completed_count;type:integer;not null;default:0" json:"completed_count"`
	FailedCount           int        `gorm:"column:failed_count;type:integer;not null;default:0" json:"failed_count"`
	Status                string     `gorm:"column:status;type:text;not null;default:'running'" json:"status"`
	CreatedBy             string     `gorm:"column:created_by;type:text" json:"created_by,omitempty"`          // 062
	FinishedAt            *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"` // 063
	NotificationSentAt    *time.Time `gorm:"column:notification_sent_at;type:timestamptz" json:"notification_sent_at,omitempty"` // 064
	CreatedAt             time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (BackfillJob) TableName() string { return "backfill_jobs" }

// BackfillItem — `backfill_items` (043; altered by 053 to add run_id,
// 061 to add attempts / last_error).
type BackfillItem struct {
	ID            string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	JobID         string     `gorm:"column:job_id;type:text;not null" json:"job_id"`
	AssetID       string     `gorm:"column:asset_id;type:text;not null" json:"asset_id"`
	Status        string     `gorm:"column:status;type:text;not null;default:'pending'" json:"status"`
	WorkflowName  string     `gorm:"column:workflow_name;type:text" json:"workflow_name,omitempty"`
	RunID         *string    `gorm:"column:run_id;type:text" json:"run_id,omitempty"`                  // 053
	Attempts      int        `gorm:"column:attempts;type:integer;not null;default:0" json:"attempts"` // 061
	LastError     string     `gorm:"column:last_error;type:text" json:"last_error,omitempty"`         // 061
	ErrorMessage  string     `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	StartedAt     *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt    *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (BackfillItem) TableName() string { return "backfill_items" }
