package audit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
)

// Handler serves audit / discovery-layer endpoints backed by asset_events and
// asset_relations tables.
type Handler struct {
	db auditQuerier
}

type auditRows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
}

type auditQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (auditRows, error)
}

type postgresQuerier struct {
	pg *postgres.Client
}

func (q postgresQuerier) Query(ctx context.Context, sql string, args ...any) (auditRows, error) {
	if q.pg == nil {
		return nil, fmt.Errorf("postgres client is not configured")
	}
	return q.pg.Query(ctx, sql, args...)
}

// New creates a new audit Handler.
func New(pg *postgres.Client) *Handler {
	if pg == nil {
		return &Handler{}
	}
	return &Handler{db: postgresQuerier{pg: pg}}
}

// auditEventRow mirrors a subset of asset_events columns for cross-asset search
// responses. It includes the actor fields that the asset_events table stores at
// the row level (actor_type, actor_id).
type auditEventRow struct {
	EventID       string    `json:"event_id"`
	EventSeq      int64     `json:"event_seq"`
	EventType     string    `json:"event_type"`
	AggregateType string    `json:"aggregate_type"`
	AssetID       string    `json:"asset_id,omitempty"`
	McapFileID    string    `json:"mcap_file_id,omitempty"`
	TenantID      string    `json:"tenant_id,omitempty"`
	ProjectID     string    `json:"project_id,omitempty"`
	EventSource   string    `json:"event_source"`
	ActorType     string    `json:"actor_type,omitempty"`
	ActorID       string    `json:"actor_id,omitempty"`
	RunID         string    `json:"run_id,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// HandleAuditSearch returns paginated asset_events rows across all assets,
// filtered by optional query parameters (CYB-1097).
//
// Query params: actor, time_from, time_to, event_type, run_id, limit, cursor
//   - actor: filters on actor_id (case-insensitive LIKE)
//   - time_from / time_to: RFC3339 bounds on occurred_at
//   - event_type: exact match on event_type
//   - run_id: exact match on run_id
//   - limit: max rows (default 50, max 200)
//   - cursor: event_seq for keyset pagination (returns rows with event_seq < cursor)
//
// @Summary      Search audit events
// @Description  Cross-asset search on asset_events with filters, keyset pagination
// @Tags         audit
// @Produce      json
// @Param        actor     query string false "Filter by actor_id (LIKE)"
// @Param        time_from query string false "Occurred at lower bound (RFC3339)"
// @Param        time_to   query string false "Occurred at upper bound (RFC3339)"
// @Param        event_type query string false "Filter by event_type (exact)"
// @Param        run_id    query string false "Filter by run_id (exact)"
// @Param        limit     query int    false "Page size" default(50)
// @Param        cursor    query int64  false "Keyset cursor (event_seq < cursor)"
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /audit/search [get]
func (h *Handler) HandleAuditSearch(c *gin.Context) {
	if h == nil || h.db == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable, "audit search is not configured", nil)
		return
	}
	actor := strings.TrimSpace(c.Query("actor"))
	timeFromStr := strings.TrimSpace(c.Query("time_from"))
	timeToStr := strings.TrimSpace(c.Query("time_to"))
	eventType := strings.TrimSpace(c.Query("event_type"))
	runID := strings.TrimSpace(c.Query("run_id"))
	cursorStr := strings.TrimSpace(c.Query("cursor"))
	limitStr := strings.TrimSpace(c.Query("limit"))

	limit := 50
	if limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v < 1 {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid limit: must be a positive integer", nil)
			return
		}
		if v > 200 {
			v = 200
		}
		limit = v
	}

	args := make([]interface{}, 0)
	where := make([]string, 0)
	paramIdx := 0

	if actor != "" {
		paramIdx++
		args = append(args, "%"+actor+"%")
		where = append(where, fmt.Sprintf("actor_id ILIKE $%d", paramIdx))
	}

	if eventType != "" {
		paramIdx++
		args = append(args, eventType)
		where = append(where, fmt.Sprintf("event_type = $%d", paramIdx))
	}

	if runID != "" {
		paramIdx++
		args = append(args, runID)
		where = append(where, fmt.Sprintf("run_id = $%d", paramIdx))
	}

	var timeFrom *time.Time
	if timeFromStr != "" {
		t, err := time.Parse(time.RFC3339, timeFromStr)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid time_from: must be RFC3339", map[string]any{"error": err.Error()})
			return
		}
		timeFrom = &t
		paramIdx++
		args = append(args, t)
		where = append(where, fmt.Sprintf("occurred_at >= $%d", paramIdx))
	}

	var timeTo *time.Time
	if timeToStr != "" {
		t, err := time.Parse(time.RFC3339, timeToStr)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid time_to: must be RFC3339", map[string]any{"error": err.Error()})
			return
		}
		timeTo = &t
		paramIdx++
		args = append(args, t)
		where = append(where, fmt.Sprintf("occurred_at <= $%d", paramIdx))
	}
	if timeFrom != nil && timeTo != nil && timeFrom.After(*timeTo) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "time_from must be before or equal to time_to", nil)
		return
	}

	if cursorStr != "" {
		cursor, err := strconv.ParseInt(cursorStr, 10, 64)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid cursor: must be an int64 event_seq", nil)
			return
		}
		paramIdx++
		args = append(args, cursor)
		where = append(where, fmt.Sprintf("event_seq < $%d", paramIdx))
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ") + " "
	}

	paramIdx++
	args = append(args, limit+1) // fetch one extra to detect next page
	limitParam := fmt.Sprintf("$%d", paramIdx)

	q := `SELECT event_id, event_seq, event_type, aggregate_type,
  COALESCE(asset_id::text, ''), COALESCE(mcap_file_id::text, ''),
  COALESCE(tenant_id, ''), COALESCE(project_id, ''),
  event_source,
  COALESCE(actor_type, ''), COALESCE(actor_id, ''), COALESCE(run_id, ''),
  occurred_at, created_at
FROM asset_events
` + whereClause + `
ORDER BY event_seq DESC
LIMIT ` + limitParam

	rows, err := h.db.Query(c.Request.Context(), q, args...)
	if err != nil {
		httpresp.Internal(c, "database query failed: "+err.Error())
		return
	}
	defer rows.Close()

	var items []auditEventRow
	for rows.Next() {
		var row auditEventRow
		if err := rows.Scan(
			&row.EventID, &row.EventSeq, &row.EventType, &row.AggregateType,
			&row.AssetID, &row.McapFileID,
			&row.TenantID, &row.ProjectID,
			&row.EventSource,
			&row.ActorType, &row.ActorID, &row.RunID,
			&row.OccurredAt, &row.CreatedAt,
		); err != nil {
			httpresp.Internal(c, "scan failed: "+err.Error())
			return
		}
		items = append(items, row)
	}

	var nextCursor *int64
	if len(items) > limit {
		nextCursor = &items[limit-1].EventSeq
		items = items[:limit]
	}
	if items == nil {
		items = []auditEventRow{}
	}

	resp := gin.H{
		"items": items,
		"limit": limit,
	}
	if nextCursor != nil {
		resp["next_cursor"] = *nextCursor
	}
	c.JSON(200, resp)
}

// lineageNode represents one node in a lineage graph returned by
// HandleLineageSearch.
type lineageNode struct {
	AssetID      string `json:"asset_id"`
	RelationType string `json:"relation_type,omitempty"`
	Direction    string `json:"direction"` // "upstream" or "downstream"
	Depth        int    `json:"depth"`
}

// HandleLineageSearch uses a recursive CTE on asset_relations to trace lineage
// upstream, downstream, or both (CYB-1098).
//
// Query params:
//   - asset_id (required): starting asset ID
//   - direction: "upstream", "downstream", or "both" (default "both")
//   - depth: max recursion depth (default 10, max 50)
//
// @Summary      Lineage search
// @Description  Recursive CTE-based lineage trace upstream/downstream/both
// @Tags         audit
// @Produce      json
// @Param        asset_id  query string true  "Starting asset ID"
// @Param        direction query string false "upstream | downstream | both" default(both)
// @Param        depth     query int    false "Max recursion depth" default(10)
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /audit/lineage-search [get]
func (h *Handler) HandleLineageSearch(c *gin.Context) {
	if h == nil || h.db == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable, "audit lineage search is not configured", nil)
		return
	}
	assetID := strings.TrimSpace(c.Query("asset_id"))
	if assetID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset_id is required", nil)
		return
	}

	direction := strings.TrimSpace(c.Query("direction"))
	if direction == "" {
		direction = "both"
	}
	direction = strings.ToLower(direction)
	if direction != "upstream" && direction != "downstream" && direction != "both" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "direction must be upstream, downstream, or both", nil)
		return
	}

	depthStr := strings.TrimSpace(c.Query("depth"))
	depth := 10
	if depthStr != "" {
		v, err := strconv.Atoi(depthStr)
		if err != nil || v < 1 {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid depth: must be a positive integer", nil)
			return
		}
		if v > 50 {
			v = 50
		}
		depth = v
	}

	// Run separate recursive CTEs for each direction; merge in Go.
	var nodes []lineageNode
	ctx := c.Request.Context()

	runDirection := func(dir string) {
		var startCol, joinCol string
		if dir == "upstream" {
			startCol, joinCol = "child_asset_id", "parent_asset_id"
		} else {
			startCol, joinCol = "parent_asset_id", "child_asset_id"
		}
		q := fmt.Sprintf(`WITH RECURSIVE rec AS (
	SELECT %s AS related_asset_id, 1 AS depth
	FROM asset_relations
	WHERE %s = $1 AND relation_type = 'revision_of'
	UNION ALL
	SELECT ar.%s, r.depth + 1
	FROM asset_relations ar
	JOIN rec r ON ar.%s = r.related_asset_id
	WHERE ar.relation_type = 'revision_of' AND r.depth < $2
)
SELECT DISTINCT related_asset_id, depth
FROM rec
ORDER BY depth ASC, related_asset_id ASC`, joinCol, startCol, joinCol, startCol)

		rows, err := h.db.Query(ctx, q, assetID, depth)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var n lineageNode
			n.Direction = dir
			if err := rows.Scan(&n.AssetID, &n.Depth); err != nil {
				continue
			}
			nodes = append(nodes, n)
		}
	}

	if direction == "upstream" || direction == "both" {
		runDirection("upstream")
	}
	if direction == "downstream" || direction == "both" {
		runDirection("downstream")
	}
	if nodes == nil {
		nodes = []lineageNode{}
	}

	c.JSON(200, gin.H{
		"asset_id":  assetID,
		"direction": direction,
		"depth":     depth,
		"nodes":     nodes,
		"count":     len(nodes),
	})
}
