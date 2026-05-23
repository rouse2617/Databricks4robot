package audit

import (
	"fmt"
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
	pg *postgres.Client
}

// New creates a new audit Handler.
func New(pg *postgres.Client) *Handler {
	return &Handler{pg: pg}
}

// auditEventRow mirrors a subset of asset_events columns for cross-asset search
// responses. It includes the actor fields that the asset_events table stores at
// the row level (actor_type, actor_id).
type auditEventRow struct {
	EventID              string    `json:"event_id"`
	EventSeq             int64     `json:"event_seq"`
	EventType            string    `json:"event_type"`
	AggregateType        string    `json:"aggregate_type"`
	AssetID              string    `json:"asset_id,omitempty"`
	McapFileID           string    `json:"mcap_file_id,omitempty"`
	TenantID             string    `json:"tenant_id,omitempty"`
	ProjectID            string    `json:"project_id,omitempty"`
	EventSource          string    `json:"event_source"`
	ActorType            string    `json:"actor_type,omitempty"`
	ActorID              string    `json:"actor_id,omitempty"`
	RunID                string    `json:"run_id,omitempty"`
	OccurredAt           time.Time `json:"occurred_at"`
	CreatedAt            time.Time `json:"created_at"`
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

	if timeFromStr != "" {
		t, err := time.Parse(time.RFC3339, timeFromStr)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid time_from: must be RFC3339", map[string]any{"error": err.Error()})
			return
		}
		paramIdx++
		args = append(args, t)
		where = append(where, fmt.Sprintf("occurred_at >= $%d", paramIdx))
	}

	if timeToStr != "" {
		t, err := time.Parse(time.RFC3339, timeToStr)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid time_to: must be RFC3339", map[string]any{"error": err.Error()})
			return
		}
		paramIdx++
		args = append(args, t)
		where = append(where, fmt.Sprintf("occurred_at <= $%d", paramIdx))
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

	rows, err := h.pg.Query(c.Request.Context(), q, args...)
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

	// Build the recursive CTE query using one or two branches.
	// upstream: child_asset_id = asset_id  → follow parent_asset_id
	// downstream: parent_asset_id = asset_id → follow child_asset_id
	var branches []string
	paramIdx := 0
	args := make([]interface{}, 0)

	if direction == "upstream" || direction == "both" {
		paramIdx++
		args = append(args, assetID)
		branches = append(branches, fmt.Sprintf(`
    SELECT parent_asset_id AS related_asset_id, 'upstream'::text AS direction, 1 AS depth
    FROM asset_relations
    WHERE child_asset_id = $%d
      AND relation_type = 'revision_of'
    UNION
    SELECT ar.parent_asset_id, 'upstream'::text, r.depth + 1
    FROM asset_relations ar
    JOIN rec r ON ar.child_asset_id = r.related_asset_id AND r.direction = 'upstream'
    WHERE ar.relation_type = 'revision_of'
      AND r.depth < $%d`, paramIdx, paramIdx+1))
	}

	if direction == "downstream" || direction == "both" {
		paramIdx++
		args = append(args, assetID)
		branches = append(branches, fmt.Sprintf(`
    SELECT child_asset_id AS related_asset_id, 'downstream'::text AS direction, 1 AS depth
    FROM asset_relations
    WHERE parent_asset_id = $%d
      AND relation_type = 'revision_of'
    UNION
    SELECT ar.child_asset_id, 'downstream'::text, r.depth + 1
    FROM asset_relations ar
    JOIN rec r ON ar.parent_asset_id = r.related_asset_id AND r.direction = 'downstream'
    WHERE ar.relation_type = 'revision_of'
      AND r.depth < $%d`, paramIdx, paramIdx+1))
	}

	paramIdx++
	args = append(args, depth)

	branchUnion := strings.Join(branches, "\n    UNION\n")

	q := fmt.Sprintf(`WITH RECURSIVE rec AS (
%s
)
SELECT DISTINCT related_asset_id, direction, depth
FROM rec
ORDER BY depth ASC, related_asset_id ASC`, branchUnion)

	rows, err := h.pg.Query(c.Request.Context(), q, args...)
	if err != nil {
		httpresp.Internal(c, "lineage query failed: "+err.Error())
		return
	}
	defer rows.Close()

	var nodes []lineageNode
	for rows.Next() {
		var n lineageNode
		if err := rows.Scan(&n.AssetID, &n.Direction, &n.Depth); err != nil {
			httpresp.Internal(c, "scan failed: "+err.Error())
			return
		}
		nodes = append(nodes, n)
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

