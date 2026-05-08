package postgres

import (
	"testing"

	"pgregory.net/rapid"
)

// ---------------------------------------------------------------------------
// Unit tests
// ---------------------------------------------------------------------------

func TestLifecycleStateToStatus_KnownValues(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"created", "approved"},
		{"processing", "approved"},
		{"ready", "approved"},
		{"rejected", "rejected"},
		{"delivered", "approved"},
		{"archived", "archived"},
		{"superseded", "superseded"},
		{"failed", "rejected"},
	}
	for _, tc := range cases {
		got := LifecycleStateToStatus(tc.input)
		if got != tc.want {
			t.Errorf("LifecycleStateToStatus(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestStatusToLifecycleState_KnownValues(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"approved", "ready"},
		{"rejected", "rejected"},
		{"archived", "archived"},
		{"superseded", "superseded"},
	}
	for _, tc := range cases {
		got := StatusToLifecycleState(tc.input)
		if got != tc.want {
			t.Errorf("StatusToLifecycleState(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestLifecycleStateToStatus_UnknownReturnsApproved(t *testing.T) {
	if got := LifecycleStateToStatus("bogus"); got != "approved" {
		t.Errorf("LifecycleStateToStatus(\"bogus\") = %q, want \"approved\"", got)
	}
}

func TestStatusToLifecycleState_UnknownReturnsCreated(t *testing.T) {
	if got := StatusToLifecycleState("bogus"); got != "created" {
		t.Errorf("StatusToLifecycleState(\"bogus\") = %q, want \"created\"", got)
	}
}

// ---------------------------------------------------------------------------
// Property 7: Lifecycle state ↔ status bidirectional mapping
// **Validates: Requirements 16**
//
// 1. For any valid lifecycle_state, LifecycleStateToStatus returns the correct
//    mapped status per the design mapping table.
// 2. For any valid status, StatusToLifecycleState returns the correct mapped
//    lifecycle_state per the design mapping table.
// 3. The mapping is deterministic and consistent (calling twice yields the
//    same result).
// ---------------------------------------------------------------------------

func TestProperty7_LifecycleStatusBidirectionalMapping(t *testing.T) {
	// Expected mapping tables (source of truth from design doc §6).
	expectedLStoStatus := map[string]string{
		"created":    "approved",
		"processing": "approved",
		"ready":      "approved",
		"rejected":   "rejected",
		"delivered":  "approved",
		"archived":   "archived",
		"superseded": "superseded",
		"failed":     "rejected",
	}
	expectedStatusToLS := map[string]string{
		"approved":   "ready",
		"rejected":   "rejected",
		"archived":   "archived",
		"superseded": "superseded",
	}

	// Generator: pick a random valid lifecycle_state.
	genLifecycleState := rapid.SampledFrom([]string{
		"created", "processing", "ready", "rejected",
		"delivered", "archived", "superseded", "failed",
	})

	// Generator: pick a random valid legacy status.
	genLegacyStatus := rapid.SampledFrom([]string{
		"approved", "rejected", "archived", "superseded",
	})

	// Sub-property 7a: lifecycle_state → status is correct and deterministic.
	t.Run("lifecycle_to_status", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			ls := genLifecycleState.Draw(t, "lifecycle_state")
			want := expectedLStoStatus[ls]

			got1 := LifecycleStateToStatus(ls)
			got2 := LifecycleStateToStatus(ls)

			if got1 != want {
				t.Fatalf("LifecycleStateToStatus(%q) = %q, want %q", ls, got1, want)
			}
			if got1 != got2 {
				t.Fatalf("LifecycleStateToStatus(%q) not deterministic: %q vs %q", ls, got1, got2)
			}
		})
	})

	// Sub-property 7b: status → lifecycle_state is correct and deterministic.
	t.Run("status_to_lifecycle", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			status := genLegacyStatus.Draw(t, "status")
			want := expectedStatusToLS[status]

			got1 := StatusToLifecycleState(status)
			got2 := StatusToLifecycleState(status)

			if got1 != want {
				t.Fatalf("StatusToLifecycleState(%q) = %q, want %q", status, got1, want)
			}
			if got1 != got2 {
				t.Fatalf("StatusToLifecycleState(%q) not deterministic: %q vs %q", status, got1, got2)
			}
		})
	})

	// Sub-property 7c: For any valid status, converting to lifecycle_state and
	// back to status yields the original status (round-trip for the status→ls
	// direction).
	t.Run("status_roundtrip", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			status := genLegacyStatus.Draw(t, "status")
			ls := StatusToLifecycleState(status)
			backToStatus := LifecycleStateToStatus(ls)

			if backToStatus != status {
				t.Fatalf("round-trip failed: status %q → ls %q → status %q (want %q)",
					status, ls, backToStatus, status)
			}
		})
	})
}
