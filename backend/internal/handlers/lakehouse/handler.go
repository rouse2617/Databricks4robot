package lakehouse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sort"
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
	if h.trino == nil {
		c.JSON(200, gin.H{"items": []trinopkg.TableCount{}})
		return
	}
	items, err := h.trino.Tables(c.Request.Context())
	if err != nil {
		// Tables() now returns empty list on failure, but handle defensively.
		c.JSON(200, gin.H{"items": []trinopkg.TableCount{}})
		return
	}
	if items == nil {
		items = []trinopkg.TableCount{}
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

// SyncStatusData represents the latest reconciliation result.
type SyncStatusData struct {
	DagsterRunID      string         `json:"dagster_run_id"`
	CheckedAt         time.Time      `json:"checked_at"`
	PgTotalCount      int64          `json:"pg_total_count"`
	IcebergTotalCount int64          `json:"iceberg_total_count"`
	CountDiffPct      float64        `json:"count_diff_pct"`
	PgStatusDist      map[string]any `json:"pg_status_dist"`
	IcebergStatusDist map[string]any `json:"iceberg_status_dist"`
	StatusDiff        map[string]any `json:"status_diff"`
	IsAlert           bool           `json:"is_alert"`
	IcebergMaxSeq     int64          `json:"iceberg_max_seq,omitempty"`
}

type SyncStatusEnvelope struct {
	Available bool            `json:"available"`
	Source    string          `json:"source,omitempty"`
	Message   string          `json:"message,omitempty"`
	Data      *SyncStatusData `json:"data,omitempty"`
}

func newRealtimeSyncStatusData(checkedAt time.Time, pgTotal, iceTotal, iceMaxSeq int64) SyncStatusData {
	var diffRatio float64
	switch {
	case pgTotal == 0 && iceTotal == 0:
		diffRatio = 0
	case pgTotal == 0:
		diffRatio = 1.0
	default:
		diff := pgTotal - iceTotal
		if diff < 0 {
			diff = -diff
		}
		diffRatio = float64(diff) / float64(pgTotal)
	}

	return SyncStatusData{
		DagsterRunID:      "",
		CheckedAt:         checkedAt.UTC(),
		PgTotalCount:      pgTotal,
		IcebergTotalCount: iceTotal,
		CountDiffPct:      diffRatio,
		PgStatusDist:      map[string]any{},
		IcebergStatusDist: map[string]any{},
		StatusDiff:        map[string]any{},
		IsAlert:           diffRatio > 0.001,
		IcebergMaxSeq:     iceMaxSeq,
	}
}

func buildStatusDiff(pgDist, iceDist map[string]int64) map[string]any {
	keysMap := map[string]struct{}{}
	for key := range pgDist {
		keysMap[key] = struct{}{}
	}
	for key := range iceDist {
		keysMap[key] = struct{}{}
	}

	keys := make([]string, 0, len(keysMap))
	for key := range keysMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	diff := make(map[string]any, len(keys))
	for _, key := range keys {
		pg := pgDist[key]
		ice := iceDist[key]
		diff[key] = map[string]int64{
			"pg":      pg,
			"iceberg": ice,
			"diff":    pg - ice,
		}
	}
	return diff
}

func (h *Handler) pgAssetCurrentStatusSummary(ctx context.Context) (int64, map[string]int64, error) {
	const q = `
SELECT status, count(*)::bigint
FROM assets
WHERE is_deleted = false
GROUP BY status`

	var total int64
	if err := h.pg.QueryRow(ctx, "SELECT count(*) FROM assets WHERE is_deleted = false").Scan(&total); err != nil {
		return 0, nil, err
	}

	rows, err := h.pg.Query(ctx, q)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	dist := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return 0, nil, err
		}
		dist[status] = count
	}
	return total, dist, nil
}

func (h *Handler) icebergAssetCurrentStatusSummary(ctx context.Context) (int64, map[string]int64, error) {
	rows, err := h.trino.QueryRows(ctx, `
		SELECT status, count(*) AS row_count
		FROM `+h.trino.Table("silver_assets_current")+`
		WHERE is_deleted = false
		GROUP BY status
	`)
	if err != nil {
		return 0, nil, err
	}

	var total int64
	dist := map[string]int64{}
	for _, row := range rows {
		status, _ := row["status"].(string)
		switch v := row["row_count"].(type) {
		case int64:
			dist[status] = v
			total += v
		case int32:
			dist[status] = int64(v)
			total += int64(v)
		case int:
			dist[status] = int64(v)
			total += int64(v)
		case float64:
			dist[status] = int64(v)
			total += int64(v)
		}
	}
	return total, dist, nil
}

func (h *Handler) buildMvpAssetSyncStatus(ctx context.Context) (*SyncStatusData, error) {
	pgTotal, pgDist, err := h.pgAssetCurrentStatusSummary(ctx)
	if err != nil {
		return nil, err
	}
	iceTotal, iceDist, err := h.icebergAssetCurrentStatusSummary(ctx)
	if err != nil {
		return nil, err
	}
	data := newRealtimeSyncStatusData(time.Now(), pgTotal, iceTotal, 0)
	data.PgStatusDist = map[string]any{}
	for key, value := range pgDist {
		data.PgStatusDist[key] = value
	}
	data.IcebergStatusDist = map[string]any{}
	for key, value := range iceDist {
		data.IcebergStatusDist[key] = value
	}
	data.StatusDiff = buildStatusDiff(pgDist, iceDist)
	data.IcebergMaxSeq = 0
	return &data, nil
}

// SyncStatus returns the latest reconciliation result.
// It compares PG event counts with Iceberg Bronze counts via Trino
// for a real-time sync health check. Falls back to the sync_reconciliation
// table if available.
func (h *Handler) SyncStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Try real-time comparison: PG published count vs Iceberg bronze count.
	if h.pg != nil && h.trino != nil {
		pgTotal, pgErr := h.pgPublishedEventCount(ctx)
		iceTotal, iceErr := h.trino.BronzeEventCount(ctx)
		iceMaxSeq, seqErr := h.trino.BronzeMaxEventSeq(ctx)

		if pgErr == nil && iceErr == nil && seqErr == nil {
			data := newRealtimeSyncStatusData(time.Now(), pgTotal, iceTotal, iceMaxSeq)
			c.JSON(200, SyncStatusEnvelope{
				Available: true,
				Source:    "realtime",
				Data:      &data,
			})
			return
		}
		if data, err := h.buildMvpAssetSyncStatus(ctx); err == nil {
			c.JSON(200, SyncStatusEnvelope{
				Available: true,
				Source:    "realtime",
				Data:      data,
			})
			return
		}
		// Fall through to sync_reconciliation table if Trino query fails.
	}

	// Fallback: read from sync_reconciliation table (Dagster-populated).
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
		resp              SyncStatusData
		pgStatusDistJSON  []byte
		iceStatusDistJSON []byte
		statusDiffJSON    []byte
	)

	err := h.pg.QueryRow(ctx, q).Scan(
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
		c.JSON(200, SyncStatusEnvelope{
			Available: false,
			Message:   "对账数据暂不可用，请先运行 Dagster pipeline 或等待 Bronze MERGE",
		})
		return
	}

	_ = json.Unmarshal(pgStatusDistJSON, &resp.PgStatusDist)
	_ = json.Unmarshal(iceStatusDistJSON, &resp.IcebergStatusDist)
	_ = json.Unmarshal(statusDiffJSON, &resp.StatusDiff)

	c.JSON(200, SyncStatusEnvelope{
		Available: true,
		Source:    "sync_reconciliation",
		Data:      &resp,
	})
}

// pgPublishedEventCount returns the count of published events in PG.
func (h *Handler) pgPublishedEventCount(ctx context.Context) (int64, error) {
	var count int64
	err := h.pg.QueryRow(ctx,
		"SELECT count(*) FROM asset_events WHERE publish_state = 'published'",
	).Scan(&count)
	return count, err
}
