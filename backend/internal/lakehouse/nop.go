package lakehouse

import (
	"context"
	"errors"
)

// ErrLakehouseDisabled is returned by Nop().Query when the analytical layer
// is not configured. Handlers should map this to HTTP 503.
var ErrLakehouseDisabled = errors.New("lakehouse: analytical backend disabled")

// nopQuerier is returned when the lakehouse query layer is disabled (e.g. via
// LAKEHOUSE_BACKEND=none or missing BQ config). All endpoints depending on it
// degrade gracefully.
type nopQuerier struct{}

// Nop returns a no-op Querier reporting Enabled=false.
func Nop() Querier { return nopQuerier{} }

func (nopQuerier) Status(_ context.Context) Status {
	return Status{Enabled: false, Healthy: false, Backend: "none"}
}

func (nopQuerier) Query(_ context.Context, _ string) ([]Row, error) {
	return nil, ErrLakehouseDisabled
}

func (nopQuerier) Close() {}
