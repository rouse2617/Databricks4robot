// Package lakehouse defines the analytical query port used by the lakehouse
// handler. Implementations adapt different query engines (BigQuery today,
// historically Trino) behind a stable interface so the handler / usecase
// layers do not depend on the engine choice.
//
// See docs/adr/001-iceberg-catalog-selection.md for the rationale of the
// current BigQuery + BigLake-managed Iceberg pairing.
package lakehouse

import "context"

// Status is the engine-level health snapshot surfaced by GET /lakehouse/status.
type Status struct {
	Enabled bool   `json:"enabled"`
	Healthy bool   `json:"healthy"`
	Backend string `json:"backend,omitempty"`
	Project string `json:"project,omitempty"`
	Dataset string `json:"dataset,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Row is a single result row keyed by column name. Values use the engine's
// native Go types (e.g. int64, float64, string, time.Time, civil.Date for BQ).
type Row map[string]any

// Querier is the minimum surface a lakehouse adapter must implement. Query
// runs an arbitrary read-only SQL statement against the analytical engine
// (BigQuery today). Adapters are responsible for engine-specific dialect.
//
// Query is intended for server-built SQL only — handlers must NOT interpolate
// untrusted user input. Parameterized inputs should be validated on the
// handler side and embedded as typed literals (e.g. DATE '2026-05-15').
type Querier interface {
	Status(ctx context.Context) Status
	Query(ctx context.Context, sql string) ([]Row, error)
	Close()
}
