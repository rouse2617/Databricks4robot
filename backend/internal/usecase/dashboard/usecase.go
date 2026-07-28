// Package dashboard hosts read-only aggregations that feed the DataBrew
// dashboard (frontend `DashboardPage`). Each endpoint is a thin SQL wrapper
// — no writes, no cross-service orchestration.
package dashboard

import (
	"context"
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// AssetDurationRepo is the minimum surface the dashboard needs from the
// asset repository. Kept small so tests can stub it without depending on
// the full repository.AssetRepository interface.
type AssetDurationRepo interface {
	DurationDistribution(ctx context.Context, assetType string) (*models.DurationDistribution, error)
}

// Usecase is the dashboard aggregation façade.
type Usecase struct {
	repo AssetDurationRepo
}

// New constructs a dashboard Usecase.
func New(repo AssetDurationRepo) *Usecase { return &Usecase{repo: repo} }

// DurationDistribution returns the fleet-wide duration histogram + overall
// stats used by 数据时长分布 (CYB-4303). assetType is optional — whitespace
// is trimmed; the empty string means "aggregate across all asset types".
// The `AssetType` field on the response echoes the trimmed value (nil when
// empty) so the client can render a consistent label.
func (u *Usecase) DurationDistribution(ctx context.Context, assetType string) (*models.DurationDistribution, error) {
	trimmed := strings.TrimSpace(assetType)
	dist, err := u.repo.DurationDistribution(ctx, trimmed)
	if err != nil {
		return nil, fmt.Errorf("dashboard.DurationDistribution: %w", err)
	}
	if trimmed != "" {
		v := trimmed
		dist.AssetType = &v
	}
	return dist, nil
}
