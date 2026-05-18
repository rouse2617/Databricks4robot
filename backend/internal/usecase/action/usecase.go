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

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// Sentinel errors surfaced to handlers.
var (
	ErrSegNotFound       = errors.New("seg not found")
	ErrParentNotSeg      = errors.New("parent asset is not a segment")
	ErrInvalidRange      = errors.New("end_ns must be >= start_ns")
	ErrRangeOutsideSeg   = errors.New("action time range falls outside the parent seg")
	ErrInvalidSourceType = errors.New("invalid source_type")
	ErrInvalidLabel      = errors.New("invalid action label")
	ErrActionNotFound    = errors.New("action not found")
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

// List validates the parent exists then returns matching actions ordered
// by (start_ns ASC, action_id ASC).
//
// Actions are defined only on segment assets; for other asset types the API
// returns an empty list (200) so asset-detail UIs can load without surfacing
// an error. Create still rejects non-segment parents with ErrParentNotSeg.
func (u *Usecase) List(ctx context.Context, in ListInput) ([]*models.Action, error) {
	parent, err := u.assets.Get(ctx, in.AssetID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrSegNotFound
	}
	if parent.AssetType != "" && parent.AssetType != "segment" {
		return []*models.Action{}, nil
	}
	return u.actions.ListByAsset(ctx, in.AssetID, repository.ActionListOptions{
		PointAtNs: in.PointAtNs,
		FromNs:    in.FromNs,
		ToNs:      in.ToNs,
		Label:     in.Label,
		Limit:     in.Limit,
	})
}

// UpdateInput captures the request shape for PATCH /assets/:id/actions/:action_id.
// All fields are optional (nil pointer = unchanged); ExpectedVersion enables
// optimistic concurrency.
type UpdateInput struct {
	AssetID         string
	ActionID        string
	ExpectedVersion int64
	Patch           repository.ActionPatch
}

// Update validates the parent seg, applies the patch with optimistic CAS, and
// emits an `action_upserted` event in the same transaction.
func (u *Usecase) Update(ctx context.Context, in UpdateInput) (*models.Action, error) {
	if in.Patch.SourceType != nil {
		st := *in.Patch.SourceType
		if st == "" || !models.IsValidActionSourceType(st) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidSourceType, st)
		}
	}
	// Validate label patches against the optional registry.
	if u.labelRegistry != nil && (in.Patch.PrimaryLabel != nil || in.Patch.Labels != nil) {
		current, err := u.actions.Get(ctx, in.ActionID)
		if err != nil {
			return nil, err
		}
		if current == nil || current.AssetID != in.AssetID {
			return nil, ErrActionNotFound
		}
		primary := current.PrimaryLabel
		if in.Patch.PrimaryLabel != nil {
			primary = *in.Patch.PrimaryLabel
		}
		labels := current.Labels
		if in.Patch.Labels != nil {
			labels = *in.Patch.Labels
		}
		if err := u.labelRegistry.Validate(primary, labels); err != nil {
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

	var updated *models.Action
	err = u.withTx(ctx, func(txCtx context.Context) error {
		// Range guardrail using effective start/end after patch.
		current, gerr := u.actions.Get(txCtx, in.ActionID)
		if gerr != nil {
			return gerr
		}
		if current == nil || current.AssetID != in.AssetID {
			return ErrActionNotFound
		}
		start := current.StartNs
		end := current.EndNs
		if in.Patch.StartNs != nil {
			start = *in.Patch.StartNs
		}
		if in.Patch.EndNs != nil {
			end = *in.Patch.EndNs
		}
		if end < start {
			return ErrInvalidRange
		}
		if parent.StartTimestampNs > 0 && parent.EndTimestampNs > 0 {
			if start < parent.StartTimestampNs || end > parent.EndTimestampNs {
				return ErrRangeOutsideSeg
			}
		}
		expected := in.ExpectedVersion
		if expected <= 0 {
			expected = current.Version
		}
		row, uerr := u.actions.Update(txCtx, in.ActionID, expected, in.Patch)
		if uerr != nil {
			return uerr
		}
		if row == nil {
			return ErrActionNotFound
		}
		updated = row
		return u.appendUpsertEvent(txCtx, row)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteInput captures the request shape for DELETE /assets/:id/actions/:action_id.
type DeleteInput struct {
	AssetID         string
	ActionID        string
	ExpectedVersion int64
}

// Delete soft-deletes the action and emits an `action_deleted` event in the
// same transaction.
func (u *Usecase) Delete(ctx context.Context, in DeleteInput) error {
	parent, err := u.assets.Get(ctx, in.AssetID)
	if err != nil {
		return err
	}
	if parent == nil {
		return ErrSegNotFound
	}
	if parent.AssetType != "" && parent.AssetType != "segment" {
		return ErrParentNotSeg
	}
	return u.withTx(ctx, func(txCtx context.Context) error {
		current, gerr := u.actions.Get(txCtx, in.ActionID)
		if gerr != nil {
			return gerr
		}
		if current == nil || current.AssetID != in.AssetID {
			return ErrActionNotFound
		}
		expected := in.ExpectedVersion
		if expected <= 0 {
			expected = current.Version
		}
		row, derr := u.actions.SoftDelete(txCtx, in.ActionID, expected)
		if derr != nil {
			return derr
		}
		if row == nil {
			return ErrActionNotFound
		}
		return u.appendDeleteEvent(txCtx, row)
	})
}

func (u *Usecase) appendDeleteEvent(ctx context.Context, a *models.Action) error {
	if u.eventRepo == nil {
		return nil
	}
	payload := map[string]any{
		"action_id": a.ActionID,
		"asset_id":  a.AssetID,
		"version":   a.Version,
	}
	body, _ := json.Marshal(payload)
	return u.eventRepo.Append(ctx, repository.AssetEventAppendInput{
		EventType:            "action_deleted",
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
