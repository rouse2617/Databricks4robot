package models

import "time"

// APIKey is an opaque credential issued to SDK / API callers. Only the hash
// of the secret is persisted; the plaintext (`dbk_<prefix>_<secret>`) is
// returned once at creation. Scopes drive least-privilege authorization; a
// key can be revoked independently (Status) without affecting other callers.
type APIKey struct {
	ID         string     `json:"id"`
	KeyPrefix  string     `json:"keyPrefix"`
	SecretHash string     `json:"-"`
	Name       string     `json:"name"`
	Owner      string     `json:"owner"`
	Scopes     []string   `json:"scopes"`
	Status     string     `json:"status"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedBy  string     `json:"createdBy"`
	CreatedAt  time.Time  `json:"createdAt"`
}
