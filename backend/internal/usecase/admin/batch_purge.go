// Package admin holds usecases that back the privileged /api/v1/internal/*
// endpoints. These are intentionally narrow surfaces for operational chores
// that cannot or should not be expressed through the public asset API
// (for example, hard-deleting rows after a bad bulk import).
package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ESDocDeleter is the minimal interface admin purge needs from the ES client
// to garbage-collect orphan search docs after a hard delete. Implemented by
// *elasticsearch.Client.DeleteDocument. Optional — when nil, ES cleanup is
// skipped (and the orphan must be cleaned by a separate reindex/reconcile).
type ESDocDeleter interface {
	DeleteDocument(ctx context.Context, id string) error
}

// DefaultChunkSize is the per-transaction batch size used by PurgeAssets /
// PurgeMcapFiles when callers don't specify one. 1000 is small enough to
// keep Cloud SQL lock holders short while still amortizing round trips.
const DefaultChunkSize = 1000

// ErrNotFound is returned by PurgeOne when the targeted asset_id does not
// exist (no asset row, no child rows).
var ErrNotFound = errors.New("admin purge: asset not found")

// ErrInvalidInput is returned when the BatchInput is empty or contradictory
// (no asset_ids and no import_batch, or both).
var ErrInvalidInput = errors.New("admin purge: provide exactly one of asset_ids or import_batch")

// BatchInput is the application-level request for the batch hard-delete
// endpoint. Exactly one of AssetIDs or ImportBatch must be set.
type BatchInput struct {
	AssetIDs         []string
	ImportBatch      string
	IncludeMcapFiles bool
	DryRun           bool
	ChunkSize        int
}

// BatchOutput captures the resolved input shape and the per-table counts
// that were (or would be, for dry-run) deleted.
type BatchOutput struct {
	DryRun           bool                   `json:"dry_run"`
	IncludeMcapFiles bool                   `json:"include_mcap_files"`
	Resolved         BatchResolved          `json:"resolved"`
	Deleted          repository.PurgeCounts `json:"deleted"`
}

// BatchResolved reports the cardinality of the resolved targets so callers
// can sanity-check before re-issuing without --dry-run.
type BatchResolved struct {
	AssetIDs    int    `json:"asset_ids_count"`
	McapFileIDs int    `json:"mcap_file_ids_count"`
	ImportBatch string `json:"import_batch,omitempty"`
}

// Usecase orchestrates dry-run inspection and the actual purge.
type Usecase struct {
	Ops repository.BatchOpsRepository
	// ES is optional. When set, successful (non-dry-run) hard deletes also
	// remove the corresponding asset docs from the search index so the
	// post-condition "deleted asset is invisible everywhere" holds without
	// waiting for an outbox event (hard delete bypasses the outbox).
	ES ESDocDeleter
}

// New constructs a usecase. ops must be non-nil. ES defaults to nil; wire it
// via WithES on the returned value if hard-delete should also clean ES.
func New(ops repository.BatchOpsRepository) *Usecase {
	return &Usecase{Ops: ops}
}

// WithES wires an optional ES doc deleter. Returns u for chaining.
func (u *Usecase) WithES(es ESDocDeleter) *Usecase {
	u.ES = es
	return u
}

// cleanES best-effort deletes asset docs from the search index. Errors are
// logged but do not fail the caller — PG is the source of truth and a stale
// ES doc is recoverable via reindex/reconcile.
func (u *Usecase) cleanES(ctx context.Context, assetIDs []string) {
	if u.ES == nil || len(assetIDs) == 0 {
		return
	}
	for _, id := range assetIDs {
		if id == "" {
			continue
		}
		if err := u.ES.DeleteDocument(ctx, id); err != nil {
			slog.Warn("admin purge: ES doc cleanup failed (stale doc may linger until reindex)",
				"asset_id", id, "err", err)
		}
	}
}

// PurgeOne hard-deletes a single asset and its child rows. The mcap_file is
// never touched here: point deletion of a single asset row is expected to
// leave shared metadata intact.
func (u *Usecase) PurgeOne(ctx context.Context, assetID string) (repository.PurgeCounts, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return repository.PurgeCounts{}, ErrInvalidInput
	}
	counts, err := u.Ops.PurgeAssetsDryRun(ctx, []string{assetID})
	if err != nil {
		return counts, err
	}
	if counts.Assets == 0 {
		return counts, ErrNotFound
	}
	out, err := u.Ops.PurgeAssets(ctx, []string{assetID}, DefaultChunkSize)
	if err == nil {
		u.cleanES(ctx, []string{assetID})
	}
	return out, err
}

// PurgeBatch resolves the input, runs a dry-run or actual purge, and
// optionally extends the operation to include mcap_files when
// IncludeMcapFiles is true.
//
// When ImportBatch is set, AssetIDs and McapFileIDs are derived from
// metadata->>'import_batch'. When AssetIDs is set, IncludeMcapFiles is
// ignored unless the caller separately provides mcap_file_ids; the API
// surface keeps this simple by binding mcap_files cleanup to the
// import_batch flow only.
func (u *Usecase) PurgeBatch(ctx context.Context, in BatchInput) (BatchOutput, error) {
	out := BatchOutput{DryRun: in.DryRun, IncludeMcapFiles: in.IncludeMcapFiles}
	hasIDs := len(in.AssetIDs) > 0
	hasBatch := strings.TrimSpace(in.ImportBatch) != ""
	if hasIDs == hasBatch {
		return out, ErrInvalidInput
	}

	assetIDs := in.AssetIDs
	var mcapIDs []string
	if hasBatch {
		batch := strings.TrimSpace(in.ImportBatch)
		ids, err := u.Ops.ListAssetIDsByImportBatch(ctx, batch)
		if err != nil {
			return out, fmt.Errorf("resolve asset_ids: %w", err)
		}
		assetIDs = ids
		out.Resolved.ImportBatch = batch
		if in.IncludeMcapFiles {
			mcapIDs, err = u.Ops.ListMcapFileIDsByImportBatch(ctx, batch)
			if err != nil {
				return out, fmt.Errorf("resolve mcap_file_ids: %w", err)
			}
		}
	}
	out.Resolved.AssetIDs = len(assetIDs)
	out.Resolved.McapFileIDs = len(mcapIDs)

	if in.DryRun {
		ac, err := u.Ops.PurgeAssetsDryRun(ctx, assetIDs)
		if err != nil {
			return out, err
		}
		out.Deleted = ac
		if len(mcapIDs) > 0 {
			mc, err := u.Ops.PurgeMcapFilesDryRun(ctx, mcapIDs)
			if err != nil {
				return out, err
			}
			out.Deleted.AssetEventsByMcap = mc.AssetEventsByMcap
			out.Deleted.EvalResultsByMcap = mc.EvalResultsByMcap
			out.Deleted.McapFiles = mc.McapFiles
		}
		return out, nil
	}

	chunk := in.ChunkSize
	if chunk <= 0 {
		chunk = DefaultChunkSize
	}
	ac, err := u.Ops.PurgeAssets(ctx, assetIDs, chunk)
	if err != nil {
		return out, err
	}
	out.Deleted = ac
	u.cleanES(ctx, assetIDs)
	if len(mcapIDs) > 0 {
		mc, err := u.Ops.PurgeMcapFiles(ctx, mcapIDs, chunk)
		if err != nil {
			return out, err
		}
		out.Deleted.AssetEventsByMcap = mc.AssetEventsByMcap
		out.Deleted.EvalResultsByMcap = mc.EvalResultsByMcap
		out.Deleted.McapFiles = mc.McapFiles
	}
	return out, nil
}
