package bigtable

// File-scope helpers for the deprecated Bigtable end-to-end test. The
// runtime path is PostgreSQL; this file only exists so the legacy e2e test
// keeps compiling against the new AlgoUsecase signature without dragging
// the projection / outbox tables into Bigtable. Algorithm endpoints are
// not exercised by the e2e suite — these mocks are intentionally inert.

import (
	"context"

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

type bigtableNoopTxRunner struct{}

func (bigtableNoopTxRunner) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type bigtableNoopAlgoLatestRepo struct{}

func (bigtableNoopAlgoLatestRepo) Upsert(context.Context, *models.AssetAlgoLatest) error { return nil }
func (bigtableNoopAlgoLatestRepo) GetByAlgo(context.Context, string, string) (*models.AssetAlgoLatest, error) {
	return nil, nil
}
func (bigtableNoopAlgoLatestRepo) ListByAsset(context.Context, string) ([]*models.AssetAlgoLatest, error) {
	return nil, nil
}

type bigtableNoopAssetEventRepo struct{}

func (bigtableNoopAssetEventRepo) Append(context.Context, repository.AssetEventAppendInput) error {
	return nil
}
func (bigtableNoopAssetEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (bigtableNoopAssetEventRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (bigtableNoopAssetEventRepo) MarkPublished(context.Context, []int64) error { return nil }

func (bigtableNoopAssetEventRepo) MarkFailed(context.Context, int64, string) error { return nil }

func (bigtableNoopAssetEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }

func (bigtableNoopAssetEventRepo) ComputeSafeHorizon(context.Context) (int64, error) { return 0, nil }

func (bigtableNoopAssetEventRepo) MarkPublishedAndAdvanceCursor(context.Context, []int64, string) error {
	return nil
}

func (bigtableNoopAssetEventRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}

func (bigtableNoopAssetEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (bigtableNoopAssetEventRepo) CursorSeq(context.Context, string) (int64, error) {
	return 0, nil
}

func (bigtableNoopAssetEventRepo) AdvanceCursor(context.Context, string, int64) error {
	return nil
}
