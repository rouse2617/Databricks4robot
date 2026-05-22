package asset

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// lineageResponse is the JSON shape for GET /assets/{id}/lineage and provenance.lineage.
type lineageResponse struct {
	AssetID    string         `json:"asset_id"`
	Upstream   gin.H          `json:"upstream"`
	Downstream gin.H          `json:"downstream"`
}

func (h *Handler) buildLineageResponse(ctx context.Context, assetID string) (lineageResponse, error) {
	out := lineageResponse{AssetID: assetID, Upstream: gin.H{}, Downstream: gin.H{
		"algo_results": []any{},
		"deliveries":   []any{},
		"eval_results": []any{},
	}}

	if h.pg == nil {
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

	var mcapFileID, mcapURI, ingestState string
	err := h.pg.QueryRow(ctx, `
		SELECT mcap_file_id, COALESCE(storage_uri,''), COALESCE(ingest_state,'')
		FROM mcap_files
		WHERE mcap_file_id = (SELECT mcap_file_id FROM assets WHERE asset_id = $1)
	`, assetID).Scan(&mcapFileID, &mcapURI, &ingestState)
	if err == nil && mcapFileID != "" {
		out.Upstream = gin.H{
			"mcap_file_id": mcapFileID,
			"mcap_uri":     mcapURI,
			"ingest_state": ingestState,
		}
	}

	algoRows, _ := h.pg.Query(ctx, `
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
			if err := algoRows.Scan(&a.AlgoName, &a.AlgoVersion, &a.Status, &a.RunID, &a.OutputURI); err == nil {
				algos = append(algos, a)
			}
		}
	}

	delRows, _ := h.pg.Query(ctx, `
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
			if err := delRows.Scan(&d.DeliveryID, &d.CustomerID, &deliveredAt); err == nil {
				if deliveredAt != nil {
					d.DeliveredAt = deliveredAt.Format(time.RFC3339)
				}
				deliveries = append(deliveries, d)
			}
		}
	}

	evalRows, _ := h.pg.Query(ctx, `
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
			if err := evalRows.Scan(&e.EvalName, &e.MetricKey, &e.MetricValue); err == nil {
				evals = append(evals, e)
			}
		}
	}

	out.Downstream = gin.H{
		"algo_results": algos,
		"deliveries":   deliveries,
		"eval_results": evals,
	}
	return out, nil
}
