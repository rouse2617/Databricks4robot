package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// BatchOpsRepo implements admin-only hard-delete batch operations.
//
// The deletion order follows the foreign-key topology defined in
// schemas/pg-phase0.sql. Each chunk runs in its own transaction so a single
// large purge does not hold cluster-wide locks for an extended period; search
// projection consumers may delete rows in chunks so the index converges incrementally.
type BatchOpsRepo struct {
	c *Client
}

var _ repository.BatchOpsRepository = (*BatchOpsRepo)(nil)

func NewBatchOpsRepo(c *Client) *BatchOpsRepo { return &BatchOpsRepo{c: c} }

func (r *BatchOpsRepo) ListAssetIDsByImportBatch(ctx context.Context, importBatch string) ([]string, error) {
	const q = `
SELECT asset_id
  FROM assets
 WHERE metadata->>'import_batch' = $1`
	rows, err := r.c.db.Query(ctx, q, importBatch)
	if err != nil {
		return nil, fmt.Errorf("postgres BatchOpsRepo.ListAssetIDsByImportBatch query: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0, 1024)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres BatchOpsRepo.ListAssetIDsByImportBatch scan: %w", err)
		}
		out = append(out, id)
	}
	return out, nil
}

func (r *BatchOpsRepo) ListMcapFileIDsByImportBatch(ctx context.Context, importBatch string) ([]string, error) {
	const q = `
SELECT mcap_file_id
  FROM mcap_files
 WHERE metadata->>'import_batch' = $1`
	rows, err := r.c.db.Query(ctx, q, importBatch)
	if err != nil {
		return nil, fmt.Errorf("postgres BatchOpsRepo.ListMcapFileIDsByImportBatch query: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0, 1024)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres BatchOpsRepo.ListMcapFileIDsByImportBatch scan: %w", err)
		}
		out = append(out, id)
	}
	return out, nil
}

func (r *BatchOpsRepo) CountAssetReferencesToMcapFiles(ctx context.Context, mcapFileIDs []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, id := range mcapFileIDs {
		out[id] = 0
	}
	if len(mcapFileIDs) == 0 {
		return out, nil
	}
	const q = `
SELECT mcap_file_id, COUNT(*)
  FROM assets
 WHERE mcap_file_id = ANY($1::text[])
 GROUP BY mcap_file_id`
	rows, err := r.c.db.Query(ctx, q, mcapFileIDs)
	if err != nil {
		return nil, fmt.Errorf("postgres BatchOpsRepo.CountAssetReferencesToMcapFiles query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int64
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("postgres BatchOpsRepo.CountAssetReferencesToMcapFiles scan: %w", err)
		}
		out[id] = n
	}
	return out, nil
}

// countTable runs SELECT COUNT(*) FROM table WHERE col = ANY($1::text[]).
func (r *BatchOpsRepo) countTable(ctx context.Context, table, col string, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = ANY($1::text[])`, table, col)
	var n int64
	if err := r.c.db.QueryRow(ctx, q, ids).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres BatchOpsRepo.countTable %s.%s: %w", table, col, err)
	}
	return n, nil
}

// countAssetRelations counts rows in asset_relations referencing any of the
// given asset_ids on either side of the edge (parent or child).
func (r *BatchOpsRepo) countAssetRelations(ctx context.Context, assetIDs []string) (int64, error) {
	if len(assetIDs) == 0 {
		return 0, nil
	}
	const q = `
SELECT COUNT(*)
  FROM asset_relations
 WHERE parent_asset_id = ANY($1::text[])
    OR child_asset_id  = ANY($1::text[])`
	var n int64
	if err := r.c.db.QueryRow(ctx, q, assetIDs).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres BatchOpsRepo.countAssetRelations: %w", err)
	}
	return n, nil
}

func (r *BatchOpsRepo) PurgeAssetsDryRun(ctx context.Context, assetIDs []string) (repository.PurgeCounts, error) {
	var counts repository.PurgeCounts
	if len(assetIDs) == 0 {
		return counts, nil
	}
	type step struct {
		field *int64
		table string
		col   string
	}
	steps := []step{
		{&counts.AssetMetrics, "asset_metrics", "asset_id"},
		{&counts.AssetEvalResults, "asset_eval_results", "asset_id"},
		{&counts.AssetAlgoLatest, "asset_algo_latest", "asset_id"},
		{&counts.AssetTags, "asset_tags", "asset_id"},
		{&counts.Actions, "actions", "asset_id"},
		{&counts.DeliveryItems, "delivery_items", "asset_id"},
		{&counts.AssetEventsByAsset, "asset_events", "asset_id"},
		{&counts.Assets, "assets", "asset_id"},
	}
	for _, s := range steps {
		n, err := r.countTable(ctx, s.table, s.col, assetIDs)
		if err != nil {
			return counts, err
		}
		*s.field = n
	}
	rel, err := r.countAssetRelations(ctx, assetIDs)
	if err != nil {
		return counts, err
	}
	counts.AssetRelations = rel
	return counts, nil
}

func (r *BatchOpsRepo) PurgeMcapFilesDryRun(ctx context.Context, mcapFileIDs []string) (repository.PurgeCounts, error) {
	var counts repository.PurgeCounts
	if len(mcapFileIDs) == 0 {
		return counts, nil
	}
	type step struct {
		field *int64
		table string
		col   string
	}
	steps := []step{
		{&counts.EvalResultsByMcap, "asset_eval_results", "mcap_file_id"},
		{&counts.AssetEventsByMcap, "asset_events", "mcap_file_id"},
		{&counts.McapFiles, "mcap_files", "mcap_file_id"},
	}
	for _, s := range steps {
		n, err := r.countTable(ctx, s.table, s.col, mcapFileIDs)
		if err != nil {
			return counts, err
		}
		*s.field = n
	}
	return counts, nil
}

// chunkStrings splits ids into batches no larger than size. size <= 0 is
// normalized to a single batch (entire slice).
func chunkStrings(ids []string, size int) [][]string {
	if size <= 0 || len(ids) <= size {
		return [][]string{ids}
	}
	out := make([][]string, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}
		out = append(out, ids[start:end])
	}
	return out
}

// execAffected runs the given SQL against the chunk and returns rows affected.
// It uses dbFromCtx so it participates in any caller-bound transaction.
func (r *BatchOpsRepo) execAffected(ctx context.Context, q string, chunk []string) (int64, error) {
	db := dbFromCtx(ctx, r.c.db)
	return db.ExecResult(ctx, q, chunk)
}

// purgeAssetsChunk deletes one chunk of asset_ids and accumulates per-table
// counts. Each chunk runs inside its own transaction so progress survives a
// later failure.
func (r *BatchOpsRepo) purgeAssetsChunk(ctx context.Context, chunk []string, counts *repository.PurgeCounts) error {
	if len(chunk) == 0 {
		return nil
	}
	// Each step is (sql, &counter). Order matches FK topology in
	// schemas/pg-phase0.sql so child rows leave before parent rows.
	steps := []struct {
		sql     string
		counter *int64
	}{
		{`DELETE FROM asset_metrics      WHERE asset_id = ANY($1::text[])`, &counts.AssetMetrics},
		{`DELETE FROM asset_eval_results WHERE asset_id = ANY($1::text[])`, &counts.AssetEvalResults},
		{`DELETE FROM asset_algo_latest  WHERE asset_id = ANY($1::text[])`, &counts.AssetAlgoLatest},
		{`DELETE FROM asset_tags         WHERE asset_id = ANY($1::text[])`, &counts.AssetTags},
		{`DELETE FROM actions            WHERE asset_id = ANY($1::text[])`, &counts.Actions},
		{`DELETE FROM delivery_items     WHERE asset_id = ANY($1::text[])`, &counts.DeliveryItems},
		{`DELETE FROM asset_relations    WHERE parent_asset_id = ANY($1::text[]) OR child_asset_id = ANY($1::text[])`, &counts.AssetRelations},
		{`DELETE FROM asset_events       WHERE asset_id = ANY($1::text[])`, &counts.AssetEventsByAsset},
		{`DELETE FROM assets             WHERE asset_id = ANY($1::text[])`, &counts.Assets},
	}
	return r.c.WithTx(ctx, func(txCtx context.Context) error {
		for _, s := range steps {
			n, err := r.execAffected(txCtx, s.sql, chunk)
			if err != nil {
				return fmt.Errorf("postgres BatchOpsRepo.purgeAssetsChunk: %w", err)
			}
			*s.counter += n
		}
		return nil
	})
}

func (r *BatchOpsRepo) PurgeAssets(ctx context.Context, assetIDs []string, chunkSize int) (repository.PurgeCounts, error) {
	var counts repository.PurgeCounts
	if len(assetIDs) == 0 {
		return counts, nil
	}
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	for _, chunk := range chunkStrings(assetIDs, chunkSize) {
		if err := r.purgeAssetsChunk(ctx, chunk, &counts); err != nil {
			return counts, err
		}
	}
	return counts, nil
}

func (r *BatchOpsRepo) purgeMcapFilesChunk(ctx context.Context, chunk []string, counts *repository.PurgeCounts) error {
	if len(chunk) == 0 {
		return nil
	}
	steps := []struct {
		sql     string
		counter *int64
	}{
		{`DELETE FROM asset_eval_results WHERE mcap_file_id = ANY($1::text[])`, &counts.EvalResultsByMcap},
		{`DELETE FROM asset_events       WHERE mcap_file_id = ANY($1::text[])`, &counts.AssetEventsByMcap},
		{`DELETE FROM mcap_files         WHERE mcap_file_id = ANY($1::text[])`, &counts.McapFiles},
	}
	return r.c.WithTx(ctx, func(txCtx context.Context) error {
		for _, s := range steps {
			n, err := r.execAffected(txCtx, s.sql, chunk)
			if err != nil {
				return fmt.Errorf("postgres BatchOpsRepo.purgeMcapFilesChunk: %w", err)
			}
			*s.counter += n
		}
		return nil
	})
}

func (r *BatchOpsRepo) PurgeMcapFiles(ctx context.Context, mcapFileIDs []string, chunkSize int) (repository.PurgeCounts, error) {
	var counts repository.PurgeCounts
	if len(mcapFileIDs) == 0 {
		return counts, nil
	}
	refs, err := r.CountAssetReferencesToMcapFiles(ctx, mcapFileIDs)
	if err != nil {
		return counts, err
	}
	var stillReferenced []string
	for id, n := range refs {
		if n > 0 {
			stillReferenced = append(stillReferenced, id)
		}
	}
	if len(stillReferenced) > 0 {
		return counts, errors.New("postgres BatchOpsRepo.PurgeMcapFiles: assets still reference mcap_files; purge assets first")
	}

	if chunkSize <= 0 {
		chunkSize = 1000
	}
	for _, chunk := range chunkStrings(mcapFileIDs, chunkSize) {
		if err := r.purgeMcapFilesChunk(ctx, chunk, &counts); err != nil {
			return counts, err
		}
	}
	return counts, nil
}
