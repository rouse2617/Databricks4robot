package pipeline

import (
	"strings"
	"testing"
)

// Covers spec scenario: "Batch run submitted with a known batch job, template, and owner".
func TestBuildCostTrackingLabels_AllKnown(t *testing.T) {
	labels := buildCostTrackingLabels("batch-123", "tpl-456", "ruipeng.huang@cyberorigin.ai")

	want := map[string]string{
		"cyber-databrew/batch-job-id": "batch-123",
		"cyber-databrew/template-id":  "tpl-456",
		"cyber-databrew/owner":        "ruipeng.huang-at-cyberorigin.ai",
	}
	if len(labels) != len(want) {
		t.Fatalf("labels = %v, want %v", labels, want)
	}
	for k, v := range want {
		if labels[k] != v {
			t.Errorf("labels[%q] = %q, want %q", k, labels[k], v)
		}
	}
}

// Covers spec scenario: "Ad-hoc run submitted with no batch association".
func TestBuildCostTrackingLabels_SkipsUnknownIdentifiers(t *testing.T) {
	labels := buildCostTrackingLabels("", "tpl-456", "")

	if _, ok := labels["cyber-databrew/batch-job-id"]; ok {
		t.Errorf("expected no batch-job-id label when BatchJobID is empty, got %v", labels)
	}
	if _, ok := labels["cyber-databrew/owner"]; ok {
		t.Errorf("expected no owner label when Owner is empty, got %v", labels)
	}
	if labels["cyber-databrew/template-id"] != "tpl-456" {
		t.Errorf("expected template-id label to still be present, got %v", labels)
	}
	if len(labels) != 1 {
		t.Errorf("expected exactly 1 label (template-id only), got %v", labels)
	}
}

// Covers spec scenario: "Owner identifier is an email address".
func TestSanitizeLabelValue_Email(t *testing.T) {
	got := sanitizeLabelValue("ruipeng.huang@cyberorigin.ai")
	if strings.Contains(got, "@") {
		t.Fatalf("sanitized value still contains '@': %q", got)
	}
	if got != "ruipeng.huang-at-cyberorigin.ai" {
		t.Fatalf("sanitizeLabelValue email = %q, want %q", got, "ruipeng.huang-at-cyberorigin.ai")
	}
}

// Covers spec scenario: "Identifier exceeds the Kubernetes label value length limit or
// contains unexpected characters".
func TestSanitizeLabelValue_LengthAndIllegalChars(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"too long", strings.Repeat("a", 100)},
		{"spaces and slashes", "my task / batch 2026"},
		{"leading and trailing illegal chars", "!!weird-value!!"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeLabelValue(tc.in)
			if len(got) > 63 {
				t.Errorf("sanitizeLabelValue(%q) length = %d, want <= 63", tc.in, len(got))
			}
			if got == "" {
				return // legal Kubernetes label values may be empty
			}
			first, last := got[0], got[len(got)-1]
			if !isAlphaNumeric(first) || !isAlphaNumeric(last) {
				t.Errorf("sanitizeLabelValue(%q) = %q, must start/end alphanumeric", tc.in, got)
			}
			for _, r := range got {
				switch {
				case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
				default:
					t.Errorf("sanitizeLabelValue(%q) = %q, contains illegal rune %q", tc.in, got, r)
				}
			}
		})
	}
}

func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
