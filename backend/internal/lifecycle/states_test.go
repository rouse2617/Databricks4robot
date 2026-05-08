package lifecycle

import "testing"

func TestAllowedAssetLifecycleStates_MatchesPGCheck(t *testing.T) {
	// Keep in lockstep with migrations/009_lifecycle_state_check.sql
	want := []string{
		"created", "processing", "ready", "delivered",
		"archived", "superseded", "failed", "rejected",
	}
	if len(AllowedAssetLifecycleStates) != len(want) {
		t.Fatalf("len %d, want %d", len(AllowedAssetLifecycleStates), len(want))
	}
	for i, w := range want {
		if AllowedAssetLifecycleStates[i] != w {
			t.Fatalf("index %d: got %q, want %q", i, AllowedAssetLifecycleStates[i], w)
		}
	}
}
