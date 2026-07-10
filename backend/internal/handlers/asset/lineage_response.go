package asset

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// lineageResponse is the JSON shape for GET /assets/{id}/lineage and provenance.lineage.
type lineageResponse struct {
	AssetID    string `json:"asset_id"`
	Upstream   gin.H  `json:"upstream"`
	Downstream gin.H  `json:"downstream"`
}

func (h *Handler) buildLineageResponse(ctx context.Context, assetID string) (lineageResponse, error) {
	out := lineageResponse{AssetID: assetID, Upstream: gin.H{}, Downstream: gin.H{
		"algo_results": []any{},
		"deliveries":   []any{},
		"eval_results": []any{},
	}}

	if h.pg == nil && h.pgq == nil {
		return out, nil
	}

	type algoEntry struct {
		AlgoName    string `json:"algo_name"`
		AlgoVersion string `json:"algo_version"`
		Status      string `json:"status"`
		RunID       string `json:"run_id,omitempty"`
		OutputURI   string `json:"output_uri,omitempty"`
	}
	type deliveryEntry struct {
		DeliveryID  string `json:"delivery_id"`
		CustomerID  string `json:"customer_id"`
		DeliveredAt string `json:"delivered_at,omitempty"`
	}
	type evalEntry struct {
		EvalName    string  `json:"eval_name"`
		MetricKey   string  `json:"metric_key"`
		MetricValue float64 `json:"metric_value,omitempty"`
	}
	// CYB-3281: immediate child assets (assets.parent_asset_id = self).
	type childEntry struct {
		AssetID       string `json:"asset_id"`
		AssetType     string `json:"asset_type"`
		ParentAssetID string `json:"parent_asset_id"`
		RootAssetID   string `json:"root_asset_id,omitempty"`
		ImportBatch   string `json:"import_batch,omitempty"`
	}

	if h.pg != nil {
		// CYB-3281: upstream is only-ADD. Keep the raw-mcap fields at the TOP LEVEL
		// (mcap_file_id / mcap_uri / ingest_state) — the frontend LineageTab + the
		// CYB-3279 parent node read them there — and merge the immediate parent
		// asset (asset_id / asset_type / …) into the same object as sibling keys.
		upstream := gin.H{}

		var mcapFileID, mcapURI, ingestState string
		// mcap_files stores the object URI in `mcap_uri` (NOT `storage_uri`, which
		// is a column on `assets`). Selecting the wrong column errored 42703 every
		// call, so upstream was silently always empty (CYB-3227).
		err := h.pg.QueryRow(ctx, `
			SELECT mcap_file_id, COALESCE(mcap_uri,''), COALESCE(ingest_state,'')
			FROM mcap_files
			WHERE mcap_file_id = (SELECT mcap_file_id FROM assets WHERE asset_id = $1)
		`, assetID).Scan(&mcapFileID, &mcapURI, &ingestState)
		switch {
		case err == nil:
			if mcapFileID != "" {
				upstream["mcap_file_id"] = mcapFileID
				upstream["mcap_uri"] = mcapURI
				upstream["ingest_state"] = ingestState
			}
		case errors.Is(err, pgx.ErrNoRows):
			// Asset has no upstream mcap (e.g. derived / grace assets) — leave empty.
		default:
			slog.Warn("lineage: upstream mcap query", "asset_id", assetID, "error", err)
		}

		// CYB-3281: immediate parent asset via assets.parent_asset_id. For a
		// segment the parent is the raw_mcap asset (asset_id == mcap_file_id).
		var pID, pType, pRoot string
		var pStart, pEnd int64
		perr := h.pg.QueryRow(ctx, `
			SELECT parent.asset_id, COALESCE(parent.asset_type,''), COALESCE(parent.root_asset_id,''),
			       COALESCE(parent.start_timestamp_ns,0), COALESCE(parent.end_timestamp_ns,0)
			FROM assets self
			JOIN assets parent ON parent.asset_id = self.parent_asset_id
			WHERE self.asset_id = $1
		`, assetID).Scan(&pID, &pType, &pRoot, &pStart, &pEnd)
		switch {
		case perr == nil:
			if pID != "" {
				upstream["asset_id"] = pID
				upstream["asset_type"] = pType
				upstream["root_asset_id"] = pRoot
				upstream["start_timestamp_ns"] = pStart
				upstream["end_timestamp_ns"] = pEnd
			}
		case errors.Is(perr, pgx.ErrNoRows):
			// No parent asset (top-level asset) — leave parent keys absent.
		default:
			slog.Warn("lineage: upstream parent query", "asset_id", assetID, "error", perr)
		}

		out.Upstream = upstream
	}

	algoRows, _ := h.pgq.Query(ctx, `
		SELECT algo_name, algo_version, status, COALESCE(run_id,''), COALESCE(output_uri,'')
		FROM asset_algo_latest
		WHERE asset_id = $1
		ORDER BY algo_name
	`, assetID)
	algos := []algoEntry{}
	if algoRows != nil {
		defer algoRows.Close()
		for algoRows.Next() {
			var a algoEntry
			if err := algoRows.Scan(&a.AlgoName, &a.AlgoVersion, &a.Status, &a.RunID, &a.OutputURI); err != nil {
				slog.Warn("lineage: scan algo row", "asset_id", assetID, "error", err)
			} else {
				algos = append(algos, a)
			}
		}
		if err := algoRows.Err(); err != nil {
			slog.Warn("lineage: iterate algo results", "asset_id", assetID, "error", err)
		}
	}

	delRows, _ := h.pgq.Query(ctx, `
		SELECT d.delivery_id, d.customer_id, d.delivered_at
		FROM delivery_items di
		JOIN deliveries d ON d.delivery_id = di.delivery_id
		WHERE di.asset_id = $1
		ORDER BY d.delivered_at DESC
		LIMIT 20
	`, assetID)
	deliveries := []deliveryEntry{}
	if delRows != nil {
		defer delRows.Close()
		for delRows.Next() {
			var d deliveryEntry
			var deliveredAt *time.Time
			if err := delRows.Scan(&d.DeliveryID, &d.CustomerID, &deliveredAt); err != nil {
				slog.Warn("lineage: scan delivery row", "asset_id", assetID, "error", err)
			} else {
				if deliveredAt != nil {
					d.DeliveredAt = deliveredAt.Format(time.RFC3339)
				}
				deliveries = append(deliveries, d)
			}
		}
		if err := delRows.Err(); err != nil {
			slog.Warn("lineage: iterate delivery results", "asset_id", assetID, "error", err)
		}
	}

	evalRows, _ := h.pgq.Query(ctx, `
		SELECT eval_name, metric_key, COALESCE(metric_value,0)
		FROM asset_eval_results
		WHERE asset_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`, assetID)
	evals := []evalEntry{}
	if evalRows != nil {
		defer evalRows.Close()
		for evalRows.Next() {
			var e evalEntry
			if err := evalRows.Scan(&e.EvalName, &e.MetricKey, &e.MetricValue); err != nil {
				slog.Warn("lineage: scan eval row", "asset_id", assetID, "error", err)
			} else {
				evals = append(evals, e)
			}
		}
		if err := evalRows.Err(); err != nil {
			slog.Warn("lineage: iterate eval results", "asset_id", assetID, "error", err)
		}
	}

	// CYB-3281: immediate child assets (segments under an mcap; actions/frames
	// under a seg). Additive — sits alongside the existing downstream lists.
	childRows, _ := h.pgq.Query(ctx, `
		SELECT asset_id, COALESCE(asset_type,''), COALESCE(parent_asset_id,''),
		       COALESCE(root_asset_id,''), COALESCE(metadata->>'import_batch','')
		FROM assets
		WHERE parent_asset_id = $1 AND is_deleted = FALSE
		ORDER BY asset_type, asset_id
		LIMIT 50
	`, assetID)
	children := []childEntry{}
	if childRows != nil {
		defer childRows.Close()
		for childRows.Next() {
			var c childEntry
			if err := childRows.Scan(&c.AssetID, &c.AssetType, &c.ParentAssetID, &c.RootAssetID, &c.ImportBatch); err != nil {
				slog.Warn("lineage: scan child row", "asset_id", assetID, "error", err)
			} else {
				children = append(children, c)
			}
		}
		if err := childRows.Err(); err != nil {
			slog.Warn("lineage: iterate children", "asset_id", assetID, "error", err)
		}
	}

	out.Downstream = gin.H{
		"algo_results": algos,
		"deliveries":   deliveries,
		"eval_results": evals,
		"children":     children,
	}
	return out, nil
}
