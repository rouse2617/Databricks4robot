// Package lakehouse hosts the lakehouse HTTP handlers.
//
// Scope is a hybrid of three backends:
//
//   - PG-backed: /overview, /asset-growth, /failure-clusters,
//     /sync-progress (lifecycle stats and per-day aggregations come from
//     the OLTP source of truth — fast, indexed).
//   - BigQuery-backed: /event-daily, /event-type-share, /tables (event
//     stream aggregations and Iceberg row counts run against the
//     bronze_asset_events external table).
//   - Static file: /report (the local notebook-based MVP snapshot).
//
// Silver/Gold-dependent surfaces (training-assets / recompute-candidates /
// customer-replay / tag-timeline) remain retired until the matching
// materializations exist; quality-distribution is enabled via
// silver_asset_quality_current.
package lakehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/lakehouse"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
)

// BronzeCheckpointReader reads the single-row Bronze checkpoint (written by
// the bronze-incremental Cloud Run Job). Backed by
// postgres.LakehouseBronzeCheckpointRepo in production; in tests, can be
// faked.
type BronzeCheckpointReader interface {
	Get(ctx context.Context) (*postgres.LakehouseBronzeCheckpoint, error)
}

// Handler serves the lakehouse endpoints. lake is always non-nil — callers
// pass lakehouse.Nop() when the analytical layer is disabled.
type Handler struct {
	reportPath       string
	lake             lakehouse.Querier
	pg               *postgres.Client
	bronzeCheckpoint BronzeCheckpointReader
}

// New constructs a lakehouse Handler. pgClient may be nil; endpoints that
// require it degrade to 503.
func New(reportPath string, lake lakehouse.Querier, pgClient *postgres.Client) *Handler {
	if lake == nil {
		lake = lakehouse.Nop()
	}
	return &Handler{reportPath: reportPath, lake: lake, pg: pgClient}
}

// WithBronzeCheckpoint wires in the Bronze checkpoint reader so
// BronzeSyncProgress can return real numbers. Optional — the handler
// degrades to "Bronze unknown" when this is unset.
func (h *Handler) WithBronzeCheckpoint(r BronzeCheckpointReader) *Handler {
	h.bronzeCheckpoint = r
	return h
}

// Report serves the pre-generated MVP lakehouse JSON report. Kept for
// backwards compatibility with the local notebook-based MVP flow.
func (h *Handler) Report(c *gin.Context) {
	data, err := os.ReadFile(h.reportPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			httpresp.NotFound(c, "LAKEHOUSE_REPORT_NOT_FOUND", "lakehouse report not found; run `make iceberg-mvp` first")
			return
		}
		c.Error(err)
		httpresp.Internal(c, "failed to read lakehouse report")
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		c.Error(err)
		httpresp.Internal(c, "failed to parse lakehouse report")
		return
	}

	c.JSON(200, payload)
}

// Status reports the analytical engine health (BigQuery today).
func (h *Handler) Status(c *gin.Context) {
	c.JSON(200, h.lake.Status(c.Request.Context()))
}

// FailureClusterItem is a single row in the failure-clusters aggregation.
type FailureClusterItem struct {
	FailureMode    string  `json:"failure_mode"`
	AlgoName       string  `json:"algo_name"`
	AffectedAssets int64   `json:"affected_assets"`
	Ratio          float64 `json:"ratio"`
}

// FailureClusters returns algo_failed events aggregated by failure_mode over
// the last N days. Primary source is PostgreSQL asset_events (no lakehouse
// dependency), using event_payload->>'failure_mode' for JSON extraction.
//
// affected_assets counts DISTINCT asset_id so repeated retries on the same
// asset don't inflate impact.
func (h *Handler) FailureClusters(c *gin.Context) {
	if h.pg == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "PG_DISABLED", "postgres not available for failure clustering", nil)
		return
	}

	days, err := parseDaysQuery(c.DefaultQuery("days", "7"))
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if days > 90 {
		days = 90
	}

	rows, err := h.pg.Query(c.Request.Context(), `
		WITH failed AS (
			SELECT
				COALESCE(event_payload->>'failure_mode', 'unknown') AS failure_mode,
				COALESCE(event_payload->>'algo_name', 'unknown')    AS algo_name,
				COUNT(DISTINCT asset_id)::bigint                    AS cnt
			FROM asset_events
			WHERE event_type = 'algo_failed'
			  AND occurred_at > NOW() - ($1::int * INTERVAL '1 day')
			GROUP BY 1, 2
		),
		total AS (SELECT COALESCE(SUM(cnt), 0) AS t FROM failed)
		SELECT
			f.failure_mode,
			f.algo_name,
			f.cnt AS affected_assets,
			CASE WHEN t.t = 0 THEN 0.0
			     ELSE CAST(f.cnt AS double precision) / CAST(t.t AS double precision)
			END AS ratio
		FROM failed f
		CROSS JOIN total t
		ORDER BY f.cnt DESC
	`, days)
	if err != nil {
		c.Error(err)
		httpresp.Internal(c, "failed to query failure clusters")
		return
	}
	defer rows.Close()

	items := make([]FailureClusterItem, 0)
	for rows.Next() {
		var fm, algoName string
		var affected int64
		var ratio float64
		if err := rows.Scan(&fm, &algoName, &affected, &ratio); err != nil {
			c.Error(err)
			httpresp.Internal(c, "failed to scan failure cluster row")
			return
		}
		items = append(items, FailureClusterItem{
			FailureMode:    fm,
			AlgoName:       algoName,
			AffectedAssets: affected,
			Ratio:          ratio,
		})
	}
	c.JSON(200, gin.H{"days": days, "items": items})
}

func parseDaysQuery(raw string) (int, error) {
	days, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || days <= 0 {
		return 0, errors.New("days must be a positive integer")
	}
	return days, nil
}

// BronzeSyncProgress is the watermark-based health snapshot for the PG→Bronze
// pipeline (Cloud Run Job `bronze-incremental` running every 5 min). All
// fields are O(1) — `outbox_published_max_seq` is `MAX(event_seq)` over PG's
// `asset_events WHERE publish_state='published'`, and the Bronze numbers
// come from the lakehouse_bronze_checkpoint single-row table maintained by
// the ingest job (no Iceberg scan).
//
// See docs/review/lakehouse-incremental-ingestion.md §Observability.
type BronzeSyncProgress struct {
	OutboxPublishedMaxSeq int64      `json:"outbox_published_max_seq"`
	BronzeMaxEventSeq     int64      `json:"bronze_max_event_seq"`
	BronzeLagEvents       int64      `json:"bronze_lag_events"`
	BronzeLastIngestedAt  *time.Time `json:"bronze_last_ingested_at,omitempty"`
	BronzeStaleSeconds    float64    `json:"bronze_stale_seconds"`
	CheckedAt             time.Time  `json:"checked_at"`
}

type SyncStatusData struct {
	DagsterRunID      string         `json:"dagster_run_id"`
	CheckedAt         time.Time      `json:"checked_at"`
	PGTotalCount      int64          `json:"pg_total_count"`
	IcebergTotalCount int64          `json:"iceberg_total_count"`
	CountDiffPct      float64        `json:"count_diff_pct"`
	PGStatusDist      map[string]any `json:"pg_status_dist"`
	IcebergStatusDist map[string]any `json:"iceberg_status_dist"`
	StatusDiff        map[string]any `json:"status_diff"`
	IsAlert           bool           `json:"is_alert"`
	IcebergMaxSeq     int64          `json:"iceberg_max_seq"`
}

type SyncStatusResponse struct {
	Available bool            `json:"available"`
	Source    string          `json:"source"`
	Message   string          `json:"message,omitempty"`
	Data      *SyncStatusData `json:"data,omitempty"`
}

// SyncStatus returns a lightweight, realtime sync snapshot between PG outbox
// and Bronze checkpoint watermarks.
func (h *Handler) SyncStatus(c *gin.Context) {
	if h.pg == nil {
		c.JSON(http.StatusOK, SyncStatusResponse{
			Available: false,
			Source:    "realtime",
			Message:   "postgres not available",
		})
		return
	}
	progress, err := h.loadBronzeSyncProgress(c.Request.Context())
	if err != nil {
		c.Error(err)
		httpresp.Internal(c, "failed to read sync status")
		return
	}
	c.JSON(http.StatusOK, buildRealtimeSyncStatus(progress))
}

// BronzeSyncProgress reports PG→Bronze watermarks for dashboards and alerts.
func (h *Handler) BronzeSyncProgress(c *gin.Context) {
	if h.pg == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "PG_DISABLED", "postgres not available for bronze sync progress", nil)
		return
	}
	progress, err := h.loadBronzeSyncProgress(c.Request.Context())
	if err != nil {
		c.Error(err)
		httpresp.Internal(c, "failed to read bronze sync progress")
		return
	}

	metrics.LakehouseBronzeMaxEventSeq.Set(float64(progress.BronzeMaxEventSeq))
	metrics.LakehouseBronzeLagEvents.Set(float64(progress.BronzeLagEvents))
	if progress.BronzeLastIngestedAt != nil {
		metrics.LakehouseBronzeLastIngestedUnixSeconds.Set(float64(progress.BronzeLastIngestedAt.Unix()))
	}

	c.JSON(http.StatusOK, progress)
}

func (h *Handler) loadBronzeSyncProgress(ctx context.Context) (BronzeSyncProgress, error) {
	progress := BronzeSyncProgress{CheckedAt: time.Now().UTC(), BronzeStaleSeconds: -1}
	if err := h.pg.QueryRow(ctx,
		`SELECT COALESCE(MAX(event_seq), 0) FROM asset_events WHERE publish_state = 'published'`,
	).Scan(&progress.OutboxPublishedMaxSeq); err != nil {
		return BronzeSyncProgress{}, err
	}

	if h.bronzeCheckpoint == nil {
		return progress, nil
	}
	cp, err := h.bronzeCheckpoint.Get(ctx)
	if err != nil {
		return BronzeSyncProgress{}, err
	}
	if cp == nil {
		return progress, nil
	}
	progress.BronzeMaxEventSeq = cp.AppliedSeq
	ts := cp.IngestedAt.UTC()
	progress.BronzeLastIngestedAt = &ts
	progress.BronzeStaleSeconds = progress.CheckedAt.Sub(ts).Seconds()
	if progress.OutboxPublishedMaxSeq > cp.AppliedSeq {
		progress.BronzeLagEvents = progress.OutboxPublishedMaxSeq - cp.AppliedSeq
	}
	return progress, nil
}

func buildRealtimeSyncStatus(progress BronzeSyncProgress) SyncStatusResponse {
	if progress.BronzeLastIngestedAt == nil {
		return SyncStatusResponse{
			Available: false,
			Source:    "realtime",
			Message:   "bronze checkpoint not available",
		}
	}
	diff := progress.BronzeLagEvents
	if diff < 0 {
		diff = 0
	}
	diffPct := 0.0
	if progress.OutboxPublishedMaxSeq > 0 {
		diffPct = float64(diff) / float64(progress.OutboxPublishedMaxSeq)
	}
	return SyncStatusResponse{
		Available: true,
		Source:    "realtime",
		Data: &SyncStatusData{
			DagsterRunID:      "",
			CheckedAt:         progress.CheckedAt,
			PGTotalCount:      progress.OutboxPublishedMaxSeq,
			IcebergTotalCount: progress.BronzeMaxEventSeq,
			CountDiffPct:      diffPct,
			PGStatusDist:      map[string]any{},
			IcebergStatusDist: map[string]any{},
			StatusDiff:        map[string]any{},
			IsAlert:           diff > 0,
			IcebergMaxSeq:     progress.BronzeMaxEventSeq,
		},
	}
}

// Overview returns Dashboard top-card stats. PG-only — no lakehouse dep so
// it stays alive even when BigQuery is down. Bronze row count piggybacks on
// MAX(event_seq) from the outbox watermark, which is cheaper than COUNT(*)
// and identical for our append-only event stream.
func (h *Handler) Overview(c *gin.Context) {
	if h.pg == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "PG_DISABLED", "postgres not available for overview", nil)
		return
	}
	ctx := c.Request.Context()

	out := gin.H{}

	var assetTotal, activeAssets, todayNew, weekNew int64
	if err := h.pg.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE NOT is_deleted),
			COUNT(*) FILTER (WHERE NOT is_deleted AND lifecycle_state NOT IN ('archived','superseded','rejected')),
			COUNT(*) FILTER (WHERE NOT is_deleted AND created_at >= date_trunc('day', NOW())),
			COUNT(*) FILTER (WHERE NOT is_deleted AND created_at >= NOW() - INTERVAL '7 days')
		FROM assets
	`).Scan(&assetTotal, &activeAssets, &todayNew, &weekNew); err != nil {
		c.Error(err)
		httpresp.Internal(c, "failed to query overview data")
		return
	}
	out["asset_total"] = assetTotal
	out["active_assets"] = activeAssets
	out["today_new_assets"] = todayNew
	out["week_new_assets"] = weekNew
	out["day7_avg_new_assets"] = float64(weekNew) / 7.0

	var mcapTotal int64
	if err := h.pg.QueryRow(ctx, `SELECT COUNT(*) FROM mcap_files`).Scan(&mcapTotal); err == nil {
		out["mcap_total"] = mcapTotal
	}

	var eventMaxSeq int64
	if err := h.pg.QueryRow(ctx,
		`SELECT COALESCE(MAX(event_seq), 0) FROM asset_events`,
	).Scan(&eventMaxSeq); err == nil {
		out["bronze_event_rows"] = eventMaxSeq
	}

	if h.bronzeCheckpoint != nil {
		if cp, err := h.bronzeCheckpoint.Get(ctx); err == nil && cp != nil {
			lag := time.Since(cp.IngestedAt).Hours()
			if lag < 0 {
				lag = 0
			}
			out["data_lag_hours"] = lag
		}
	}

	c.JSON(http.StatusOK, out)
}

// AssetGrowth returns per-day new asset count over the last N days plus a
// running cumulative count. PG-only — uses the (created_at) index on
// assets.
func (h *Handler) AssetGrowth(c *gin.Context) {
	if h.pg == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "PG_DISABLED", "postgres not available", nil)
		return
	}
	days, err := parseDaysQuery(c.DefaultQuery("days", "30"))
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if days > 365 {
		days = 365
	}

	rows, err := h.pg.Query(c.Request.Context(), `
		WITH days AS (
			SELECT generate_series(
				date_trunc('day', NOW()) - ($1::int - 1) * INTERVAL '1 day',
				date_trunc('day', NOW()),
				INTERVAL '1 day'
			)::date AS d
		),
		daily AS (
			SELECT date_trunc('day', created_at)::date AS d, COUNT(*)::bigint AS n
			FROM assets
			WHERE NOT is_deleted
			  AND created_at >= date_trunc('day', NOW()) - ($1::int - 1) * INTERVAL '1 day'
			GROUP BY 1
		),
		baseline AS (
			SELECT COUNT(*)::bigint AS c FROM assets
			WHERE NOT is_deleted
			  AND created_at < date_trunc('day', NOW()) - ($1::int - 1) * INTERVAL '1 day'
		)
		SELECT
			d.d::text,
			COALESCE(daily.n, 0) AS new_assets,
			(SELECT c FROM baseline) + SUM(COALESCE(daily.n, 0)) OVER (ORDER BY d.d) AS cumulative_assets
		FROM days d
		LEFT JOIN daily ON daily.d = d.d
		ORDER BY d.d
	`, days)
	if err != nil {
		c.Error(err)
		httpresp.Internal(c, "failed to query asset growth data")
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0, days)
	for rows.Next() {
		var date string
		var newCount, cumulative int64
		if err := rows.Scan(&date, &newCount, &cumulative); err != nil {
			c.Error(err)
			httpresp.Internal(c, "failed to read lakehouse data")
			return
		}
		items = append(items, gin.H{
			"event_date":        date,
			"new_assets":        newCount,
			"cumulative_assets": cumulative,
		})
	}
	c.JSON(http.StatusOK, gin.H{"days": days, "items": items})
}

// Tables returns row counts for lakehouse Iceberg tables visible to BigQuery.
// Currently only bronze_asset_events is materialized; Silver/Gold join in
// later when their external tables exist.
func (h *Handler) Tables(c *gin.Context) {
	rows, err := h.lake.Query(c.Request.Context(),
		`SELECT 'bronze_asset_events' AS table_name, COUNT(*) AS row_count FROM bronze_asset_events`,
	)
	if err != nil {
		h.lakehouseFail(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		items = append(items, gin.H{
			"table_name": asString(r["table_name"]),
			"row_count":  asInt64(r["row_count"]),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// EventDaily aggregates Bronze events by day × event_type over the last N
// days. Driven by occurred_at so re-ingested rows don't skew the curve.
func (h *Handler) EventDaily(c *gin.Context) {
	days, err := parseDaysQuery(c.DefaultQuery("days", "14"))
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if days > 90 {
		days = 90
	}
	sql := fmt.Sprintf(`
		SELECT
			DATE(occurred_at)            AS event_date,
			COALESCE(event_type,'unknown') AS event_type,
			COUNT(*)                     AS event_count
		FROM bronze_asset_events
		WHERE occurred_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL %d DAY)
		GROUP BY 1, 2
		ORDER BY 1, 2`, days)
	rows, err := h.lake.Query(c.Request.Context(), sql)
	if err != nil {
		h.lakehouseFail(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		n := asInt64(r["event_count"])
		items = append(items, gin.H{
			"event_date":  asDateString(r["event_date"]),
			"event_type":  asString(r["event_type"]),
			"count":       n,
			"asset_count": n, // legacy alias; DashboardPage and older SDKs read this
		})
	}
	c.JSON(http.StatusOK, gin.H{"days": days, "items": items})
}

// EventTypeShare returns the event_type distribution for a specific date.
// date is a required YYYY-MM-DD query param, validated before being embedded
// in the BQ SQL (no other user input flows into the query).
func (h *Handler) EventTypeShare(c *gin.Context) {
	dateStr, err := parseEventTypeShareDate(c.Query("date"), time.Now().UTC())
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	// Falls back to the most recent date with events (within last 30 days)
	// when the requested date has none — keeps the pie chart populated when
	// today hasn't ingested yet.
	sql := fmt.Sprintf(`
		WITH effective AS (
			SELECT COALESCE(
				(SELECT DATE(occurred_at) FROM bronze_asset_events
				  WHERE DATE(occurred_at) = DATE '%s' LIMIT 1),
				(SELECT MAX(DATE(occurred_at)) FROM bronze_asset_events
				  WHERE occurred_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY))
			) AS d
		),
		t AS (
			SELECT COALESCE(event_type,'unknown') AS event_type, COUNT(*) AS event_count
			FROM bronze_asset_events, effective
			WHERE DATE(occurred_at) = effective.d
			GROUP BY 1
		),
		tot AS (SELECT SUM(event_count) AS total FROM t)
		SELECT (SELECT CAST(d AS STRING) FROM effective) AS effective_date,
		       t.event_type, t.event_count,
		       SAFE_DIVIDE(t.event_count, (SELECT total FROM tot)) AS ratio
		FROM t
		ORDER BY t.event_count DESC`, dateStr)
	rows, err := h.lake.Query(c.Request.Context(), sql)
	if err != nil {
		h.lakehouseFail(c, err)
		return
	}
	effectiveDate := dateStr
	if len(rows) > 0 {
		if ed := asString(rows[0]["effective_date"]); ed != "" {
			effectiveDate = ed
		}
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		n := asInt64(r["event_count"])
		items = append(items, gin.H{
			"event_type":  asString(r["event_type"]),
			"count":       n,
			"asset_count": n, // legacy alias for frontend DashboardPage
			"ratio":       asFloat64(r["ratio"]),
		})
	}
	c.JSON(http.StatusOK, gin.H{"date": effectiveDate, "items": items})
}

func parseEventTypeShareDate(raw string, now time.Time) (string, error) {
	dateStr := strings.TrimSpace(raw)
	if dateStr == "" || strings.EqualFold(dateStr, "latest") {
		return now.UTC().Format("2006-01-02"), nil
	}
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return "", errors.New("date must be YYYY-MM-DD or 'latest'")
	}
	return dateStr, nil
}

const silverAssetQualityCurrentTable = "silver_asset_quality_current"

// QualityDistribution returns the count of assets per quality value over a
// rolling window. Lakehouse-backed — reads a Silver current-state table built
// from PG assets + asset_tags, so semantics stay aligned with
// `NOT is_deleted` and current quality tag value.
func (h *Handler) QualityDistribution(c *gin.Context) {
	days, err := parseQualityWindow(c.DefaultQuery("window", "30d"))
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	sql := fmt.Sprintf(`
		WITH latest AS (
			SELECT
				asset_id,
				created_at,
				is_deleted,
				quality,
				ROW_NUMBER() OVER (
					PARTITION BY asset_id
					ORDER BY updated_at DESC, _ingested_at DESC
				) AS rn
			FROM %s
		)
		SELECT
			COALESCE(NULLIF(quality, ''), 'unknown') AS quality,
			COUNT(*)                                 AS asset_count
		FROM latest
		WHERE rn = 1
		  AND is_deleted = FALSE
		  AND created_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL %d DAY)
		GROUP BY 1
		ORDER BY 2 DESC
	`, silverAssetQualityCurrentTable, days)
	rows, err := h.lake.Query(c.Request.Context(), sql)
	if err != nil {
		h.lakehouseFail(c, err)
		return
	}

	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		q := asString(r["quality"])
		if q == "" {
			q = "unknown"
		}
		items = append(items, gin.H{
			"quality":     q,
			"asset_count": asInt64(r["asset_count"]),
		})
	}
	c.JSON(http.StatusOK, gin.H{"window": fmt.Sprintf("%dd", days), "items": items})
}

func parseQualityWindow(raw string) (int, error) {
	switch strings.TrimSpace(raw) {
	case "", "30d":
		return 30, nil
	case "7d":
		return 7, nil
	case "60d":
		return 60, nil
	case "90d":
		return 90, nil
	default:
		return 0, errors.New("window must be one of 7d/30d/60d/90d")
	}
}

// CustomerReplay returns recent deliveries for a given customer_id with
// status timeline fields. PG-only — direct read from the deliveries table
// (no Silver/Gold dependency). customer_id is required; the result is
// capped at 100 rows and ordered by created_at DESC.
func (h *Handler) CustomerReplay(c *gin.Context) {
	if h.pg == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "PG_DISABLED", "postgres not available", nil)
		return
	}
	ctx := c.Request.Context()
	customerID := strings.TrimSpace(c.Query("customer_id"))

	// If the caller didn't ask for a specific customer, OR asked for one
	// that has no deliveries, auto-pick the customer with the most recent
	// activity so the Dashboard tile is rarely empty in non-prod envs.
	if customerID == "" {
		_ = h.pg.QueryRow(ctx, `
			SELECT customer_id FROM deliveries
			WHERE NOT is_deleted
			ORDER BY created_at DESC
			LIMIT 1`).Scan(&customerID)
	} else {
		var has int
		_ = h.pg.QueryRow(ctx,
			`SELECT 1 FROM deliveries WHERE customer_id=$1 AND NOT is_deleted LIMIT 1`,
			customerID,
		).Scan(&has)
		if has == 0 {
			var fallback string
			if err := h.pg.QueryRow(ctx, `
				SELECT customer_id FROM deliveries
				WHERE NOT is_deleted
				ORDER BY created_at DESC
				LIMIT 1`).Scan(&fallback); err == nil && fallback != "" {
				customerID = fallback
			}
		}
	}

	if customerID == "" {
		c.JSON(http.StatusOK, gin.H{"customer_id": "", "items": []any{}})
		return
	}

	rows, err := h.pg.Query(ctx, `
		SELECT
			delivery_id::text,
			customer_id,
			delivery_type,
			status,
			item_count,
			COALESCE(total_size_bytes, 0)             AS total_size_bytes,
			created_at,
			delivered_at,
			completed_at
		FROM deliveries
		WHERE customer_id = $1 AND NOT is_deleted
		ORDER BY created_at DESC
		LIMIT 100`, customerID)
	if err != nil {
		c.Error(err)
		httpresp.Internal(c, "failed to query customer replay data")
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var deliveryID, custID, dtype, status string
		var itemCount, totalBytes int64
		var createdAt time.Time
		var deliveredAt, completedAt *time.Time
		if err := rows.Scan(&deliveryID, &custID, &dtype, &status, &itemCount, &totalBytes, &createdAt, &deliveredAt, &completedAt); err != nil {
			c.Error(err)
			httpresp.Internal(c, "failed to read lakehouse data")
			return
		}
		items = append(items, gin.H{
			"delivery_id":      deliveryID,
			"customer_id":      custID,
			"delivery_type":    dtype,
			"status":           status,
			"item_count":       itemCount,
			"total_size_bytes": totalBytes,
			"created_at":       createdAt,
			"delivered_at":     deliveredAt,
			"completed_at":     completedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"customer_id": customerID, "items": items})
}

// lakehouseFail maps a Querier error to HTTP. Disabled → 503, anything else
// → 500. Used by all BQ-backed endpoints.
func (h *Handler) lakehouseFail(c *gin.Context, err error) {
	if errors.Is(err, lakehouse.ErrLakehouseDisabled) {
		httpresp.Error(c, http.StatusServiceUnavailable, "LAKEHOUSE_DISABLED", "lakehouse backend not configured", nil)
		return
	}
	// Log the real error for diagnostics but return a stable message to the client.
	c.Error(err) // gin.Context.Error writes to the gin error log
	httpresp.Internal(c, "lakehouse query service temporarily unavailable")
}

// asString / asInt64 / asFloat64 / asDateString coerce BigQuery's native Go
// types (which arrive as any from the map) into JSON-safe shapes.
func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func asInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}

func asFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}

// asDateString accepts cloud.google.com/go/civil.Date (BQ's native DATE
// representation) without importing the package, by relying on String() or
// fmt.Stringer.
func asDateString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if t, ok := v.(time.Time); ok {
		return t.Format("2006-01-02")
	}
	if st, ok := v.(fmt.Stringer); ok {
		return st.String()
	}
	return fmt.Sprintf("%v", v)
}
