package postgres

// LifecycleState represents the authoritative status of an asset in the 2.0
// schema, replacing the legacy "status" column during the dual-write period.
type LifecycleState string

const (
	LifecycleCreated    LifecycleState = "created"
	LifecycleProcessing LifecycleState = "processing"
	LifecycleReady      LifecycleState = "ready"
	LifecycleRejected   LifecycleState = "rejected"
	LifecycleDelivered  LifecycleState = "delivered"
	LifecycleArchived   LifecycleState = "archived"
	LifecycleSuperseded LifecycleState = "superseded"
	LifecycleFailed     LifecycleState = "failed"
)

// AllLifecycleStates enumerates every valid lifecycle_state value.
var AllLifecycleStates = []LifecycleState{
	LifecycleCreated,
	LifecycleProcessing,
	LifecycleReady,
	LifecycleRejected,
	LifecycleDelivered,
	LifecycleArchived,
	LifecycleSuperseded,
	LifecycleFailed,
}

// AllLegacyStatuses enumerates every valid legacy status value.
var AllLegacyStatuses = []string{"approved", "rejected", "archived", "superseded"}

// lifecycleToStatus maps lifecycle_state → legacy status.
var lifecycleToStatus = map[LifecycleState]string{
	LifecycleCreated:    "approved",
	LifecycleProcessing: "approved",
	LifecycleReady:      "approved",
	LifecycleRejected:   "rejected",
	LifecycleDelivered:  "approved",
	LifecycleArchived:   "archived",
	LifecycleSuperseded: "superseded",
	LifecycleFailed:     "rejected",
}

// statusToLifecycle maps legacy status → lifecycle_state.
var statusToLifecycle = map[string]LifecycleState{
	"approved":   LifecycleReady,
	"rejected":   LifecycleRejected,
	"archived":   LifecycleArchived,
	"superseded": LifecycleSuperseded,
}

// LifecycleStateToStatus converts a lifecycle_state value to the corresponding
// legacy status value. Unknown inputs return "approved" as a safe default.
func LifecycleStateToStatus(ls string) string {
	if s, ok := lifecycleToStatus[LifecycleState(ls)]; ok {
		return s
	}
	return "approved"
}

// StatusToLifecycleState converts a legacy status value to the corresponding
// lifecycle_state value. Unknown inputs return "created" as a safe default.
func StatusToLifecycleState(status string) string {
	if ls, ok := statusToLifecycle[status]; ok {
		return string(ls)
	}
	return "created"
}
