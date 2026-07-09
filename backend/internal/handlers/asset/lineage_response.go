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

	if h.pg != nil {
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
				out.Upstream = gin.H{
					"mcap_file_id": mcapFileID,
					"mcap_uri":     mcapURI,
					"ingest_state": ingestState,
				}
			}
		case errors.Is(err, pgx.ErrNoRows):
			// Asset has no upstream mcap (e.g. derived / grace assets) — leave empty.
		default:
			slog.Warn("lineage: upstream mcap query", "asset_id", assetID, "error", err)
		}
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

	out.Downstream = gin.H{
		"algo_results": algos,
		"deliveries":   deliveries,
		"eval_results": evals,
	}
	return out, nil
}
