// Package action implements business rules for seg-internal time-bounded
// annotations (the third layer of the `mcap → seg → action` business model).
//
// Strong invariants:
//   - Every Create writes both the `actions` row and an `action_upserted` event
//     in the same transaction (transactional outbox).
//   - The parent asset MUST exist with asset_type='segment' and not be soft-
//     deleted; usecase checks this before persisting.
//   - Time bounds use the same ns scale as `assets.start_timestamp_ns / end_timestamp_ns`.
package action

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"data-platform/internal/config"
	"data-platform/internal/middleware"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// Sentinel errors surfaced to handlers.
var (
	ErrSegNotFound       = errors.New("seg not found")
	ErrParentNotSeg      = errors.New("parent asset is not a segment")
	ErrInvalidRange      = errors.New("end_ns must be >= start_ns")
	ErrRangeOutsideSeg   = errors.New("action time range falls outside the parent seg")
	ErrInvalidSourceType = errors.New("invalid source_type")
	ErrInvalidLabel      = errors.New("invalid action label")
)

// Usecase coordinates the action repository, the parent-seg lookup, and the
// transactional event append.
type Usecase struct {
	tx            repository.TxRunner
	actions       repository.ActionRepository
	assets        repository.AssetRepository
	eventRepo     repository.AssetEventRepository
	labelRegistry *config.ActionLabelRegistry
}

// New wires a Usecase with the minimal set of dependencies.
func New(
	tx repository.TxRunner,
	actions repository.ActionRepository,
	assets repository.AssetRepository,
	eventRepo repository.AssetEventRepository,
) *Usecase {
	return &Usecase{tx: tx, actions: actions, assets: assets, eventRepo: eventRepo}
}

// NewWithLabelRegistry wires a Usecase with optional action-label registry
// validation. Pass nil to keep validation disabled.
func NewWithLabelRegistry(
	tx repository.TxRunner,
	actions repository.ActionRepository,
	assets repository.AssetRepository,
	eventRepo repository.AssetEventRepository,
	labelRegistry *config.ActionLabelRegistry,
) *Usecase {
	uc := New(tx, actions, assets, eventRepo)
	uc.labelRegistry = labelRegistry
	return uc
}

func (u *Usecase) withTx(ctx context.Context, fn func(context.Context) error) error {
	if u.tx == nil {
		return fn(ctx)
	}
	return u.tx.WithTx(ctx, fn)
}

// CreateInput captures the request shape for POST /assets/:id/actions.
type CreateInput struct {
	AssetID       string
	StartNs       int64
	EndNs         int64
	ActionIndex   *int
	PrimaryLabel  string
	Labels        []string
	Description   string
	Attrs         map[string]interface{}
	SourceType    string
	SourceName    string
	SourceVersion string
	RunID         string
	Confidence    *float64
	ExternalID    string
}

// Create validates the parent seg, persists the action, and appends an
// `action_upserted` event in the same transaction.
func (u *Usecase) Create(ctx context.Context, in CreateInput) (*models.Action, error) {
	if in.EndNs < in.StartNs {
		return nil, ErrInvalidRange
	}
	sourceType := in.SourceType
	if sourceType == "" {
		sourceType = models.ActionSourceHuman
	}
	if !models.IsValidActionSourceType(sourceType) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidSourceType, sourceType)
	}
	if u.labelRegistry != nil {
		if err := u.labelRegistry.Validate(in.PrimaryLabel, in.Labels); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidLabel, err)
		}
	}

	parent, err := u.assets.Get(ctx, in.AssetID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrSegNotFound
	}
	if parent.AssetType != "" && parent.AssetType != "segment" {
		return nil, ErrParentNotSeg
	}
	// Time-window guardrail: require the action interval to fall within the seg.
	// Accepts the seg-as-mcap-window convention (start_timestamp_ns / end_timestamp_ns).
	if parent.StartTimestampNs > 0 && parent.EndTimestampNs > 0 {
		if in.StartNs < parent.StartTimestampNs || in.EndNs > parent.EndTimestampNs {
			return nil, ErrRangeOutsideSeg
		}
	}

	row := &models.Action{
		AssetID:       in.AssetID,
		StartNs:       in.StartNs,
		EndNs:         in.EndNs,
		ActionIndex:   in.ActionIndex,
		PrimaryLabel:  in.PrimaryLabel,
		Labels:        in.Labels,
		Description:   in.Description,
		Attrs:         in.Attrs,
		SourceType:    sourceType,
		SourceName:    in.SourceName,
		SourceVersion: in.SourceVersion,
		RunID:         in.RunID,
		Confidence:    in.Confidence,
		ExternalID:    in.ExternalID,
		TenantID:      parent.TenantID,
		ProjectID:     parent.ProjectID,
	}
	if row.Labels == nil {
		row.Labels = []string{}
	}
	if row.Attrs == nil {
		row.Attrs = map[string]interface{}{}
	}

	err = u.withTx(ctx, func(txCtx context.Context) error {
		if err := u.actions.Insert(txCtx, row); err != nil {
			return err
		}
		return u.appendUpsertEvent(txCtx, row)
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (u *Usecase) appendUpsertEvent(ctx context.Context, a *models.Action) error {
	if u.eventRepo == nil {
		return nil
	}
	payload := map[string]any{
		"action_id":   a.ActionID,
		"asset_id":    a.AssetID,
		"start_ns":    a.StartNs,
		"end_ns":      a.EndNs,
		"labels":      a.Labels,
		"source_type": a.SourceType,
		"version":     a.Version,
	}
	if a.ActionIndex != nil {
		payload["action_index"] = *a.ActionIndex
	}
	if a.PrimaryLabel != "" {
		payload["primary_label"] = a.PrimaryLabel
	}
	if a.SourceName != "" {
		payload["source_name"] = a.SourceName
	}
	if a.SourceVersion != "" {
		payload["source_version"] = a.SourceVersion
	}
	if a.RunID != "" {
		payload["run_id"] = a.RunID
	}
	if a.Confidence != nil {
		payload["confidence"] = *a.Confidence
	}
	if a.ExternalID != "" {
		payload["external_id"] = a.ExternalID
	}
	body, _ := json.Marshal(payload)
	return u.eventRepo.Append(ctx, repository.AssetEventAppendInput{
		EventType:            "action_upserted",
		AggregateType:        "asset",
		PayloadSchemaVersion: "v1",
		AssetID:              a.AssetID,
		TenantID:             a.TenantID,
		ProjectID:            a.ProjectID,
		EventSource:          "backend",
		RequestID:            middleware.RequestIDFromContext(ctx),
		EventPayload:         body,
	})
}

// ListInput is the seg-scoped GET /assets/:id/actions query.
type ListInput struct {
	AssetID   string
	PointAtNs *int64
	FromNs    *int64
	ToNs      *int64
	Label     string
	Limit     int
}

// List validates the parent seg exists then returns matching actions ordered
// by (start_ns ASC, action_id ASC).
func (u *Usecase) List(ctx context.Context, in ListInput) ([]*models.Action, error) {
	parent, err := u.assets.Get(ctx, in.AssetID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrSegNotFound
	}
	if parent.AssetType != "" && parent.AssetType != "segment" {
		return nil, ErrParentNotSeg
	}
	return u.actions.ListByAsset(ctx, in.AssetID, repository.ActionListOptions{
		PointAtNs: in.PointAtNs,
		FromNs:    in.FromNs,
		ToNs:      in.ToNs,
		Label:     in.Label,
		Limit:     in.Limit,
	})
}

// nowUTC is exposed for test injection if we add createdAt overrides later.
var nowUTC = func() time.Time { return time.Now().UTC() }
