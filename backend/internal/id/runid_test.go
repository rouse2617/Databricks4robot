package id

import "testing"

func TestValidateRunID(t *testing.T) {
	if !ValidateRunID("R001abc123def456") {
		t.Fatal("expected valid 16-char run id")
	}
	if ValidateRunID("short") {
		t.Fatal("expected invalid short id")
	}
	if ValidateRunID("abcdefghijklmnop!") {
		t.Fatal("expected invalid punctuation")
	}
}

func TestGenerateRunID(t *testing.T) {
	for i := 0; i < 20; i++ {
		rid, err := GenerateRunID()
		if err != nil {
			t.Fatalf("GenerateRunID: %v", err)
		}
		if !ValidateRunID(rid) {
			t.Fatalf("generated id %q failed validation", rid)
		}
	}
}
