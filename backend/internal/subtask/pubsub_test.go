package subtask

import (
	"reflect"
	"testing"
)

// TestExtractAssetIDs pins the subscription message contract (CYB-3778 /
// CYB-3801): the data payload is JSON carrying an `asset_ids` array plus a
// reserved `topic`; asset ids drive dispatch (one -> single run, more ->
// batch). This is the sole parse entry point for the pubsub consumer and had
// no test coverage while the package churned (CYB-4025 #583). See CYB-4260.
func TestExtractAssetIDs(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		wantIDs   []string
		wantTopic string
		wantErr   bool
	}{
		{
			name:    "multi asset -> batch input",
			data:    `{"asset_ids":["a","b"]}`,
			wantIDs: []string{"a", "b"},
		},
		{
			name:    "single asset -> run input",
			data:    `{"asset_ids":["a"]}`,
			wantIDs: []string{"a"},
		},
		{
			name:      "reserved topic parsed, does not affect ids",
			data:      `{"asset_ids":["a"],"topic":"verify-single"}`,
			wantIDs:   []string{"a"},
			wantTopic: "verify-single",
		},
		{
			name:    "unknown keys ignored (forward-extensible)",
			data:    `{"asset_ids":["a"],"future":"y","nested":{"k":1}}`,
			wantIDs: []string{"a"},
		},
		{
			name:    "whitespace trimmed and empty strings dropped",
			data:    `{"asset_ids":["  a ","","b","   "]}`,
			wantIDs: []string{"a", "b"},
		},
		{
			name:    "all-empty asset_ids is an error",
			data:    `{"asset_ids":["",  "   "]}`,
			wantErr: true,
		},
		{
			name:    "missing asset_ids is an error",
			data:    `{"topic":"x"}`,
			wantErr: true,
		},
		{
			name:    "malformed json is an error",
			data:    `{"asset_ids":[`,
			wantErr: true,
		},
		{
			// Legacy single-field {"asset_id":"a"} was intentionally dropped in
			// CYB-3801 (no back-compat). Guard against a silent re-introduction:
			// the singular key must NOT satisfy the parser.
			name:    "legacy singular asset_id is not accepted",
			data:    `{"asset_id":"a"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, topic, err := extractAssetIDs([]byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got ids=%v topic=%q", ids, topic)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(ids, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", ids, tt.wantIDs)
			}
			if topic != tt.wantTopic {
				t.Errorf("topic = %q, want %q", topic, tt.wantTopic)
			}
		})
	}
}
