package auth

import "testing"

func TestGenerateAndVerifyAPIKey(t *testing.T) {
	full, prefix, hash, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if !IsAPIKey(full) {
		t.Fatalf("generated key %q not recognized as api key", full)
	}
	p, secret, ok := ParseAPIKey(full)
	if !ok {
		t.Fatalf("ParseAPIKey(%q) failed", full)
	}
	if p != prefix {
		t.Fatalf("parsed prefix %q != %q", p, prefix)
	}
	if !VerifyAPIKeySecret(secret, hash) {
		t.Fatal("VerifyAPIKeySecret failed for correct secret")
	}
	if VerifyAPIKeySecret(secret+"x", hash) {
		t.Fatal("VerifyAPIKeySecret accepted a wrong secret")
	}
	if VerifyAPIKeySecret(secret, HashAPIKeySecret("other")) {
		t.Fatal("VerifyAPIKeySecret accepted mismatched hash")
	}
}

func TestGenerateAPIKeyUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		full, prefix, _, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey: %v", err)
		}
		if seen[prefix] {
			t.Fatalf("duplicate prefix generated: %q", prefix)
		}
		seen[prefix] = true
		if !IsAPIKey(full) {
			t.Fatalf("key %q missing prefix", full)
		}
	}
}

func TestParseAPIKeyMalformed(t *testing.T) {
	cases := []string{"", "nope", "dbk_", "dbk_abc", "dbk_abc_", "dbk__secret", "jwt.token.here"}
	for _, c := range cases {
		if _, _, ok := ParseAPIKey(c); ok {
			t.Errorf("ParseAPIKey(%q) = ok, want false", c)
		}
	}
}
