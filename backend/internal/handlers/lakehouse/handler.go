package lakehouse

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"data-platform/internal/httpresp"
	"data-platform/internal/postgres"
	trinopkg "data-platform/internal/trino"
)

type Handler struct {
	reportPath string
	trino      *trinopkg.Client
	pg         *postgres.Client
}

// New constructs a lakehouse Handler. pgClient may be nil; SyncStatus
// degrades to "unavailable" in that case.
func New(reportPath string, trinoClient *trinopkg.Client, pgClient *postgres.Client) *Handler {
	return &Handler{reportPath: reportPath, trino: trinoClient, pg: pgClient}
}

func (h *Handler) Report(c *gin.Context) {
	data, err := os.ReadFile(h.reportPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			httpresp.NotFound(c, "LAKEHOUSE_REPORT_NOT_FOUND", "lakehouse report not found; run `make iceberg-mvp` first")
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		httpresp.Internal(c, "invalid lakehouse report json: "+err.Error())
		return
	}

	c.JSON(200, payload)
}

func (h *Handler) Status(c *gin.Context) {
	if h.trino == nil {
		c.JSON(200, trinopkg.Status{Enabled: false, Healthy: false})
		return
	}
	c.JSON(200, h.trino.Status(c.Request.Context()))
}

func (h *Handler) Tables(c *gin.Context) {
	if !h.requireTrino(c) {
		return
	}
	items, err := h.trino.Tables(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"items": items})
}

func (h *Handler) TrainingAssets(c *gin.Context) {
	if !h.requireTrino(c) {
		return
	}
	snapshotID := c.DefaultQuery("snapshot_id", "mvp_hand_tracking_quality_v1")
	rows, err := h.trino.QueryRows(c.Request.Context(), `
		SELECT dataset_snapshot_id, asset_id, mcap_file_id, segment_locator, env, task, algo_key, algo_status, output_uri, quality, created_at
		FROM `+h.trino.Table("gold_dataset_snapshot_items")+`
		WHERE dataset_snapshot_id = ?
		ORDER BY asset_id
		LIMIT 100
	`, snapshotID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"snapshot_id": snapshotID, "items": rows})
}

func (h *Handler) RecomputeCandidates(c *gin.Context) {
	if !h.requireTrino(c) {
		return
	}
	algoKey := c.DefaultQuery("algo_key", "hand_tracking@1.2.0")
	targetVersion := c.DefaultQuery("target_version", "next")
	rows, err := h.trino.QueryRows(c.Request.Context(), `
		SELECT a.asset_id, a.mcap_file_id, a.env, a.task, algo.algo_key, algo.status, algo.updated_at
		FROM `+h.trino.Table("silver_asset_algo_latest")+` algo
		JOIN `+h.trino.Table("silver_assets_current")+` a ON a.asset_id = algo.asset_id
		WHERE algo.algo_key = ?
		  AND algo.status IN ('ok', 'failed', 'running', 'pending')
		ORDER BY algo.updated_at DESC
		LIMIT 100
	`, algoKey)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"algo_key": algoKey, "target_version": targetVersion, "items": rows})
}

func (h *Handler) TagTimeline(c *gin.Context) {
	if !h.requireTrino(c) {
		return
	}
	tagKey := c.DefaultQuery("tag_key", "quality")
	rows, err := h.trino.QueryRows(c.Request.Context(), `
		SELECT tag_key, tag_value, updated_at, count(*) AS asset_count
		FROM `+h.trino.Table("silver_asset_tags")+`
		WHERE tag_key = ?
		GROUP BY tag_key, tag_value, updated_at
		ORDER BY updated_at DESC
		LIMIT 100
	`, tagKey)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{
		"tag_key": tagKey,
		"note":    "当前 schema 只有 tag current-state updated_at；精确追加时间需要 asset_mutation_outbox 入湖。",
		"items":   rows,
	})
}

func (h *Handler) QualityDistribution(c *gin.Context) {
	if !h.requireTrino(c) {
		return
	}
	window := c.DefaultQuery("window", "30d")
	if window != "30d" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "only window=30d is supported in the MVP", nil)
		return
	}
	rows, err := h.trino.QueryRows(c.Request.Context(), `
		SELECT tag.tag_value AS quality, count(*) AS asset_count
		FROM `+h.trino.Table("silver_assets_current")+` a
		JOIN `+h.trino.Table("silver_asset_tags")+` tag ON a.asset_id = tag.asset_id
		WHERE tag.tag_key = 'quality'
		  AND a.created_at >= current_date - INTERVAL '30' DAY
		  AND a.created_at < current_date + INTERVAL '1' DAY
		GROUP BY tag.tag_value
		ORDER BY asset_count DESC
	`)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"window": window, "items": rows})
}

func (h *Handler) CustomerReplay(c *gin.Context) {
	if !h.requireTrino(c) {
		return
	}
	customerID := c.DefaultQuery("customer_id", "urn:grace:customer:A")
	rows, err := h.trino.QueryRows(c.Request.Context(), `
		SELECT d.customer_id, d.delivery_id, d.status, d.delivered_at, di.asset_id, a.mcap_file_id, a.segment_locator
		FROM `+h.trino.Table("silver_deliveries_current")+` d
		JOIN `+h.trino.Table("bronze_delivery_items")+` di ON d.delivery_id = di.delivery_id
		JOIN `+h.trino.Table("silver_assets_current")+` a ON a.asset_id = di.asset_id
		WHERE d.customer_id = ?
		ORDER BY d.delivered_at DESC, di.asset_id
		LIMIT 100
	`, customerID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"customer_id": customerID, "items": rows})
}

func (h *Handler) requireTrino(c *gin.Context) bool {
	if h.trino == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "TRINO_DISABLED", "trino query layer is disabled", nil)
		return false
	}
	return true
}

// SyncStatusResponse represents the latest reconciliation result.
type SyncStatusResponse struct {
	DagsterRunID      string         `json:"dagster_run_id"`
	CheckedAt         time.Time      `json:"checked_at"`
	PgTotalCount      int64          `json:"pg_total_count"`
	IcebergTotalCount int64          `json:"iceberg_total_count"`
	CountDiffPct      float64        `json:"count_diff_pct"`
	PgStatusDist      map[string]any `json:"pg_status_dist"`
	IcebergStatusDist map[string]any `json:"iceberg_status_dist"`
	StatusDiff        map[string]any `json:"status_diff"`
	IsAlert           bool           `json:"is_alert"`
}

// SyncStatus returns the most recent reconciliation result from the sync_reconciliation table.
func (h *Handler) SyncStatus(c *gin.Context) {
	if h.pg == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "PG_DISABLED", "postgres not available for sync status", nil)
		return
	}

	const q = `
SELECT dagster_run_id, checked_at, pg_total_count, iceberg_total_count,
       count_diff_pct, pg_status_dist, iceberg_status_dist, status_diff, is_alert
FROM sync_reconciliation
ORDER BY checked_at DESC
LIMIT 1`

	var (
		resp              SyncStatusResponse
		pgStatusDistJSON  []byte
		iceStatusDistJSON []byte
		statusDiffJSON    []byte
	)

	err := h.pg.QueryRow(c.Request.Context(), q).Scan(
		&resp.DagsterRunID,
		&resp.CheckedAt,
		&resp.PgTotalCount,
		&resp.IcebergTotalCount,
		&resp.CountDiffPct,
		&pgStatusDistJSON,
		&iceStatusDistJSON,
		&statusDiffJSON,
		&resp.IsAlert,
	)
	if err != nil {
		// Table may not exist yet or no rows — return empty status
		c.JSON(200, gin.H{
			"available": false,
			"message":   "对账数据暂不可用，请先运行 Dagster pipeline",
		})
		return
	}

	_ = json.Unmarshal(pgStatusDistJSON, &resp.PgStatusDist)
	_ = json.Unmarshal(iceStatusDistJSON, &resp.IcebergStatusDist)
	_ = json.Unmarshal(statusDiffJSON, &resp.StatusDiff)

	c.JSON(200, gin.H{
		"available": true,
		"data":      resp,
	})
}
