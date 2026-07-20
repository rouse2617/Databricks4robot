package models

import "time"

// DispatcherConfig is one cluster's online-tunable dispatch settings
// (CYB-3679). Absent row = compiled defaults; the submitter re-reads config
// every cycle, so edits take effect within one tick without a deploy.
type DispatcherConfig struct {
	ClusterID      string    `json:"cluster_id"`
	MaxConcurrency int       `json:"max_concurrency"`
	SubmitBatch    int       `json:"submit_batch"`
	RatePerSec     float64   `json:"rate_per_sec"`
	Paused         bool      `json:"paused"`
	UpdatedBy      string    `json:"updated_by"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DispatcherClusterStatus merges a cluster's configured settings with the
// governor's live state — what the UI shows as "configured vs effective".
type DispatcherClusterStatus struct {
	ClusterID string `json:"cluster_id"`
	// Configured knobs (from DB row or compiled defaults).
	Config DispatcherConfig `json:"config"`
	// HasRow reports whether an explicit DB row exists (false = defaults).
	HasRow bool `json:"has_row"`
	// EffectiveConcurrency is the governor's current AIMD value — lower than
	// configured means backpressure is active.
	EffectiveConcurrency int `json:"effective_concurrency"`
	// SuppressionReason is a short human-readable reason why the channel is
	// not dispatching at full configured capacity ("" = none):
	// "paused" | "aimd_backoff".
	SuppressionReason string `json:"suppression_reason,omitempty"`
}
