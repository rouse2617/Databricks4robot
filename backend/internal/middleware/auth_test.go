package middleware

import "testing"

// tokenEquals is the constant-time comparator that replaced the previous
// plain != checks in StaticTokenAuth / JWTAuth / AdminTokenAuth. These
// tests pin the invariants that matter for the timing-side-channel fix and
// the fail-closed empty-want behavior.
func TestTokenEquals(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
		ok   bool
	}{
		{"empty want rejects empty got (fail closed on misconfig)", "", "", false},
		{"empty want rejects any got", "anything", "", false},
		{"empty got rejects", "", "secret-123", false},
		{"length mismatch (short)", "sec", "secret-123", false},
		{"length mismatch (long)", "secret-1234", "secret-123", false},
		{"same length wrong bytes", "secret-XXX", "secret-123", false},
		{"exact match", "secret-123", "secret-123", true},
		{"Bearer prefix stripped then match", "Bearer secret-123", "secret-123", true},
		{"Bearer prefix but wrong secret", "Bearer secret-XXX", "secret-123", false},
		{"got has only 'Bearer ' (empty secret) rejects", "Bearer ", "secret-123", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tokenEquals(tc.got, tc.want); got != tc.ok {
				t.Fatalf("tokenEquals(%q, %q) = %v, want %v", tc.got, tc.want, got, tc.ok)
			}
		})
	}
}
