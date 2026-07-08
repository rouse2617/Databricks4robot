package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// APIKeyPrefix marks opaque API keys so the auth middleware can distinguish
// them from JWTs and the legacy static token.
const APIKeyPrefix = "dbk_" // pragma: allowlist secret

// IsAPIKey reports whether a credential looks like an API key.
func IsAPIKey(token string) bool {
	return strings.HasPrefix(token, APIKeyPrefix)
}

// GenerateAPIKey mints a new key. It returns:
//   - full:       the plaintext `dbk_<prefix>_<secret>` (show once, never stored)
//   - prefix:     the lookup id stored in api_keys.key_prefix (unique)
//   - secretHash: sha256(secret) hex, stored in api_keys.secret_hash
//
// The secret is 256 bits of CSPRNG entropy, so a fast sha256 + constant-time
// compare is appropriate (bcrypt is for low-entropy passwords, not needed here).
func GenerateAPIKey() (full, prefix, secretHash string, err error) {
	pb := make([]byte, 6)
	if _, err = rand.Read(pb); err != nil {
		return "", "", "", err
	}
	prefix = hex.EncodeToString(pb) // 12 hex chars

	sb := make([]byte, 32)
	if _, err = rand.Read(sb); err != nil {
		return "", "", "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(sb)

	full = APIKeyPrefix + prefix + "_" + secret
	secretHash = HashAPIKeySecret(secret)
	return full, prefix, secretHash, nil
}

// HashAPIKeySecret hashes the secret portion for storage/comparison.
func HashAPIKeySecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// ParseAPIKey splits a full key into (prefix, secret). ok=false on malformed input.
func ParseAPIKey(full string) (prefix, secret string, ok bool) {
	if !strings.HasPrefix(full, APIKeyPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(full, APIKeyPrefix)
	i := strings.IndexByte(rest, '_')
	if i <= 0 || i >= len(rest)-1 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}

// VerifyAPIKeySecret constant-time compares a presented secret against a stored hash.
func VerifyAPIKeySecret(secret, storedHash string) bool {
	return subtle.ConstantTimeCompare([]byte(HashAPIKeySecret(secret)), []byte(storedHash)) == 1
}
