package id

import "testing"

func TestValidateAssetID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"short", false},
		{"1234567", false},
		{"123456789", false},
		{"abcd1234", true},
		{"ABCD1234", true},
		{"abcd-123", false},
		{"abcd123!", false},
	}
	for _, tc := range cases {
		if got := ValidateAssetID(tc.in); got != tc.want {
			t.Fatalf("ValidateAssetID(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestGenerateAssetID(t *testing.T) {
	seen := map[string]struct{}{}
	for range 200 {
		idv, err := GenerateAssetID()
		if err != nil {
			t.Fatal(err)
		}
		if len(idv) != AssetIDLen {
			t.Fatalf("len=%d %q", len(idv), idv)
		}
		if !ValidateAssetID(idv) {
			t.Fatalf("invalid id %q", idv)
		}
		seen[idv] = struct{}{}
	}
	if len(seen) != 200 {
		t.Fatalf("expected unique ids in sample, got %d distinct", len(seen))
	}
}

func TestValidateActionID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"short", false},
		{"1234567", false},
		{"123456789", false},
		{"abcd1234", true},
		{"ABCD1234", true},
		{"abcd-123", false},
		{"abcd123!", false},
	}
	for _, tc := range cases {
		if got := ValidateActionID(tc.in); got != tc.want {
			t.Fatalf("ValidateActionID(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestGenerateActionID(t *testing.T) {
	seen := map[string]struct{}{}
	for range 200 {
		idv, err := GenerateActionID()
		if err != nil {
			t.Fatal(err)
		}
		if len(idv) != AssetIDLen {
			t.Fatalf("len=%d %q", len(idv), idv)
		}
		if !ValidateActionID(idv) {
			t.Fatalf("invalid action id %q", idv)
		}
		seen[idv] = struct{}{}
	}
	if len(seen) != 200 {
		t.Fatalf("expected unique action ids in sample, got %d distinct", len(seen))
	}
}
