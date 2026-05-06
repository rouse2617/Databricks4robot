package cdc

import "testing"

func TestChangeEvent_StringField_KeyPayloadWrapper(t *testing.T) {
	ev := ChangeEvent{
		Table: "assets",
		Op:    OperationCreate,
		After: nil,
		Before: nil,
		Key: map[string]any{
			"payload": map[string]any{
				"asset_id": "a1",
			},
		},
	}

	if got := ev.StringField("asset_id"); got != "a1" {
		t.Fatalf("unexpected asset_id: %q", got)
	}
}

