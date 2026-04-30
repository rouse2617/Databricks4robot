package cdc

import "testing"

func TestDecodeDebeziumMessage_Insert(t *testing.T) {
	raw := []byte(`{
	  "payload": {
	    "op": "c",
	    "before": null,
	    "after": {
	      "asset_id": "a1",
	      "owner": "alice"
	    },
	    "source": {
	      "table": "assets"
	    }
	  }
	}`)

	event, err := DecodeDebeziumMessage(raw, map[string]any{"asset_id": "a1"})
	if err != nil {
		t.Fatalf("DecodeDebeziumMessage returned error: %v", err)
	}
	if event.Table != "assets" || event.Op != OperationCreate {
		t.Fatalf("unexpected event: %+v", event)
	}
	if event.StringField("asset_id") != "a1" {
		t.Fatalf("unexpected asset_id: %+v", event)
	}
}

func TestDecodeDebeziumMessage_UnsupportedOp(t *testing.T) {
	raw := []byte(`{
	  "payload": {
	    "op": "x",
	    "source": {
	      "table": "assets"
	    }
	  }
	}`)

	_, err := DecodeDebeziumMessage(raw, nil)
	if err == nil {
		t.Fatal("expected unsupported op error")
	}
}
