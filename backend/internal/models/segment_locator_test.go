package models

import (
	"crypto/sha1"
	"encoding/hex"
	"strconv"
	"testing"

	"pgregory.net/rapid"
)

// TestComputeSegmentLocator_KnownValue verifies a specific known input.
func TestComputeSegmentLocator_KnownValue(t *testing.T) {
	got := ComputeSegmentLocator("abc-123", 100, 200)
	// Independent computation
	h := sha1.New()
	h.Write([]byte("abc-123"))
	h.Write([]byte("100"))
	h.Write([]byte("200"))
	want := hex.EncodeToString(h.Sum(nil))
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

// TestComputeSegmentLocator_Length verifies the output is always 40 hex chars.
func TestComputeSegmentLocator_Length(t *testing.T) {
	got := ComputeSegmentLocator("x", 0, 0)
	if len(got) != 40 {
		t.Fatalf("expected length 40, got %d", len(got))
	}
}

// Property 1: Segment locator is deterministic and matches independent SHA-1 computation.
// For any valid (mcap_file_id, start_timestamp_ns, end_timestamp_ns) triple,
// ComputeSegmentLocator SHALL always produce the same 40-character hex string,
// and that string SHALL equal sha1hex(mcap_file_id + str(start_ns) + str(end_ns)).
//
// **Validates: Requirements 3.1**
func TestProperty1_SegmentLocatorDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mcapFileID := rapid.String().Draw(t, "mcap_file_id")
		startNs := rapid.Int64().Draw(t, "start_ns")
		endNs := rapid.Int64().Draw(t, "end_ns")

		result1 := ComputeSegmentLocator(mcapFileID, startNs, endNs)
		result2 := ComputeSegmentLocator(mcapFileID, startNs, endNs)

		// Deterministic: same inputs → same output
		if result1 != result2 {
			t.Fatalf("non-deterministic: %q != %q", result1, result2)
		}

		// Always 40 hex chars
		if len(result1) != 40 {
			t.Fatalf("expected length 40, got %d", len(result1))
		}

		// Matches independent SHA-1 computation
		h := sha1.New()
		h.Write([]byte(mcapFileID))
		h.Write([]byte(strconv.FormatInt(startNs, 10)))
		h.Write([]byte(strconv.FormatInt(endNs, 10)))
		expected := hex.EncodeToString(h.Sum(nil))
		if result1 != expected {
			t.Fatalf("mismatch: got %q, want %q", result1, expected)
		}
	})
}

// Property 2: Distinct inputs produce distinct locators.
// For any two distinct (mcap_file_id, start_timestamp_ns, end_timestamp_ns) triples
// where at least one component differs, ComputeSegmentLocator SHALL produce
// different locator values (with overwhelming probability given SHA-1's collision resistance).
//
// **Validates: Requirements 3.1**
func TestProperty2_DistinctInputsDistinctLocators(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id1 := rapid.String().Draw(t, "id1")
		start1 := rapid.Int64().Draw(t, "start1")
		end1 := rapid.Int64().Draw(t, "end1")

		// Generate a second triple that differs in at least one component
		component := rapid.IntRange(0, 2).Draw(t, "diff_component")
		id2 := id1
		start2 := start1
		end2 := end1
		switch component {
		case 0:
			id2 = rapid.String().Filter(func(s string) bool { return s != id1 }).Draw(t, "id2")
		case 1:
			start2 = rapid.Int64().Filter(func(n int64) bool { return n != start1 }).Draw(t, "start2")
		case 2:
			end2 = rapid.Int64().Filter(func(n int64) bool { return n != end1 }).Draw(t, "end2")
		}

		loc1 := ComputeSegmentLocator(id1, start1, end1)
		loc2 := ComputeSegmentLocator(id2, start2, end2)

		if loc1 == loc2 {
			t.Fatalf("collision: (%q,%d,%d) and (%q,%d,%d) both produce %q",
				id1, start1, end1, id2, start2, end2, loc1)
		}
	})
}
