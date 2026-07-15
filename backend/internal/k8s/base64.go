package k8s

import (
	"encoding/base64"
	"strings"
)

// base64DecodeIfBase64 returns the decoded bytes when s parses cleanly as
// base64 std encoding AND doesn't already look like PEM. Returns (nil, nil)
// for strings that are almost certainly raw PEM (starts with "-----BEGIN").
// This is a soft heuristic: GKE describe returns pure base64 (no header),
// while pasted PEM starts with "-----BEGIN CERTIFICATE-----".
func base64DecodeIfBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "-----BEGIN") {
		return nil, nil
	}
	// Reject strings with any character that std base64 doesn't accept so
	// we don't accidentally partial-decode a mostly-PEM string.
	return base64.StdEncoding.DecodeString(s)
}
