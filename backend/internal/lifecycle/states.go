// Package lifecycle defines asset lifecycle_state values enforced by PostgreSQL
// (see migrations/009_lifecycle_state_check.sql). API registry and clients should
// use AllowedAssetLifecycleStates; do not add values here without a migration.
package lifecycle

// AllowedAssetLifecycleStates is the exact set allowed in assets.lifecycle_state
// (CHECK constraint chk_lifecycle_state), in a stable display order.
var AllowedAssetLifecycleStates = []string{
	"created",
	"processing",
	"ready",
	"delivered",
	"archived",
	"superseded",
	"failed",
	"rejected",
}
