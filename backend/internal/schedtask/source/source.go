// Package source is the pluggable data-source layer for scheduled tasks
// (CYB-3744). A scheduled-task rule carries a source_type + source_config; this
// package resolves that into an AssetSource that fetches asset-ids on each run.
//
// v1 ships a single implementation, restSource, driven by config so that
// different endpoints AND different REST APIs are expressible without code
// changes. Genuinely non-REST platforms (GraphQL/gRPC/exotic auth) add a coded
// adapter behind the same interface when a real one appears.
package source

import (
	"context"
	"time"
)

// Window describes what a Fetch call should return.
//
//   - Incremental / rolling / range modes populate Start/End (UTC).
//   - "ids" mode populates IDs; the source may then return them verbatim.
type Window struct {
	Start *time.Time
	End   *time.Time
	IDs   []string
}

// AssetSource pulls a batch of asset-ids from an upstream system. Fetch is
// stateless w.r.t. the rule; the scheduler owns cursor advancement.
//
// The returned nextCursor is opaque to the scheduler: incremental rules
// persist it and pass it back on the next Fetch. Empty nextCursor means "keep
// the previous cursor" — the scheduler will decide whether that's a genuine
// no-op or a signal to advance.
type AssetSource interface {
	Fetch(ctx context.Context, cursor string, window Window) (ids []string, nextCursor string, err error)
}

// SecretResolver resolves a secret-manager reference to its plaintext value.
// Injected so tests can stub without touching Secret Manager, and so the
// backend can wire whatever secret backend is already in use.
//
// A nil resolver on a source that needs one MUST fail closed (return an error
// from Fetch); it MUST NOT fall back to reading env vars or files, to avoid
// accidental plaintext-in-config regressions.
type SecretResolver interface {
	Resolve(ctx context.Context, secretRef string) (string, error)
}
