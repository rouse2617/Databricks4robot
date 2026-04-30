// Package elasticsearch provides a lightweight Elasticsearch client for search and
// bulk-index operations against the "assets" index.
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"data-platform/internal/metrics"
)

// Client wraps HTTP calls to an Elasticsearch cluster.
type Client struct {
	baseURL    string
	index      string
	httpClient *http.Client
}

// New creates a Client. Pass "" for index to default to "assets".
func New(baseURL, index string) *Client {
	if index == "" {
		index = "assets"
	}
	return &Client{
		baseURL: baseURL,
		index:   index,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ---------- Search ----------

// FilterOp represents a single filter with field, operator, and value.
type FilterOp struct {
	Field string
	Op    string // eq, ne, gt, gte, lt, lte, between
	Value string
}

// SearchRequest describes the parameters accepted by the search handler.
type SearchRequest struct {
	Query    string     // free-text query (multi_match)
	Filters  []FilterOp // structured filters with operators
	Page     int
	PageSize int
}

// SearchHit is a single document returned by Elasticsearch.
type SearchHit struct {
	ID        string              `json:"_id"`
	Score     float64             `json:"_score"`
	Source    map[string]any      `json:"_source"`
	Highlight map[string][]string `json:"highlight,omitempty"`
}

// AggBucket is one bucket from a terms aggregation.
type AggBucket struct {
	Key      string `json:"key"`
	DocCount int64  `json:"doc_count"`
}

// SearchResponse is the structured result of a search call.
type SearchResponse struct {
	Total        int64                  `json:"total"`
	Hits         []SearchHit            `json:"hits"`
	Aggregations map[string][]AggBucket `json:"aggregations,omitempty"`
}

// Search executes a bool query against the assets index.
//
// Filter field routing:
//
//   - tags.<key>            → nested query on path "tags",
//     match {tags.key=<key>, tags.value=<value>} (also supports range/ne).
//   - algos.<name>          → nested query on path "algos",
//     match {algos.name=<name>, algos.status=<value>}.
//   - algos.<name>.<attr>   → nested query on path "algos",
//     match {algos.name=<name>, algos.<attr> op <value>}; attr "score"
//     is rewritten to nested field result_score.
//   - everything else       → flat term/range on the literal field name.
//     This includes flattened paths (`tags_flat.<key>`,
//     `metadata.<key>`) and reflective namespaces (`mcap.<col>`).
//
// Operator "between" expects Value="lower,upper" and translates to
// {"range": {field: {gte: lower, lte: upper}}}.
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 200 {
		req.PageSize = 20
	}

	body := buildSearchBody(req)
	return c.doSearch(ctx, body)
}

// buildSearchBody is exported via Search; kept package-private and pure so it
// can be unit-tested without an HTTP roundtrip.
func buildSearchBody(req SearchRequest) map[string]any {
	from := (req.Page - 1) * req.PageSize

	must := make([]map[string]any, 0)
	filter := make([]map[string]any, 0)
	mustNot := make([]map[string]any, 0)

	if req.Query != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  req.Query,
				"fields": []string{"notes", "owner.text", "reviewer.text", "asset_id"},
				"type":   "best_fields",
			},
		})
	}

	// Group nested filters by path+key so that multiple conditions on the
	// same tag (e.g. tags.quality:eq:good AND tags.source_type:eq:algo) are
	// combined into a single nested query with all must clauses. This ensures
	// the conditions match within the same nested document.
	type nestedGroup struct {
		path    string
		must    []map[string]any
		mustNot []map[string]any
	}
	tagGroups := make(map[string]*nestedGroup) // key: "tags:<tagKey>" or "algos:<algoName>"

	for _, f := range req.Filters {
		if path, ok := nestedPath(f.Field); ok {
			// Determine the grouping key (tag key or algo name).
			groupKey := nestedGroupKey(path, f.Field)
			clauses := buildGroupedNestedClauses(path, f)
			if g, exists := tagGroups[groupKey]; exists {
				g.must = append(g.must, clauses.must...)
				g.mustNot = append(g.mustNot, clauses.mustNot...)
			} else {
				tagGroups[groupKey] = &nestedGroup{
					path:    path,
					must:    clauses.must,
					mustNot: clauses.mustNot,
				}
			}
			continue
		}
		positive, negative := buildFilterClause(f)
		if positive != nil {
			filter = append(filter, positive)
		}
		if negative != nil {
			mustNot = append(mustNot, negative)
		}
	}

	// When a request targets exactly one concrete tag key plus one or more
	// tag inner fields (e.g. tags.quality + tags.source_type), bind the inner
	// field clauses into that key group so they match on the same nested doc.
	var soleTagGroup *nestedGroup
	var soleTagGroupKey string
	tagKeyGroups := 0
	for key, g := range tagGroups {
		if g.path != "tags" || strings.HasPrefix(key, "tags:_inner:") {
			continue
		}
		tagKeyGroups++
		soleTagGroup = g
		soleTagGroupKey = key
	}
	if tagKeyGroups == 1 && soleTagGroup != nil {
		for key, g := range tagGroups {
			if g.path != "tags" || !strings.HasPrefix(key, "tags:_inner:") {
				continue
			}
			soleTagGroup.must = append(soleTagGroup.must, g.must...)
			soleTagGroup.mustNot = append(soleTagGroup.mustNot, g.mustNot...)
			delete(tagGroups, key)
		}
		tagGroups[soleTagGroupKey] = soleTagGroup
	}

	// Emit grouped nested queries.
	for _, g := range tagGroups {
		if len(g.must) == 0 && len(g.mustNot) == 0 {
			continue
		}
		boolQuery := map[string]any{}
		if len(g.must) == 0 && len(g.mustNot) > 0 {
			boolQuery["must"] = g.mustNot
		} else if len(g.must) > 0 {
			boolQuery["must"] = g.must
		}
		if len(g.must) > 0 && len(g.mustNot) > 0 {
			boolQuery["must_not"] = g.mustNot
		}

		nested := map[string]any{
			"nested": map[string]any{
				"path":  g.path,
				"query": map[string]any{"bool": boolQuery},
			},
		}

		// Pure negative inner-field groups (e.g. tags.source_type != algo) must
		// stay as outer must_not; otherwise ES would match any sibling nested doc
		// that does not satisfy the excluded clause.
		if len(g.must) == 0 && len(g.mustNot) > 0 {
			mustNot = append(mustNot, nested)
		} else {
			filter = append(filter, nested)
		}
	}

	boolQuery := map[string]any{}
	if len(must) > 0 {
		boolQuery["must"] = must
	}
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}
	if len(mustNot) > 0 {
		boolQuery["must_not"] = mustNot
	}

	query := map[string]any{"match_all": map[string]any{}}
	if len(must) > 0 || len(filter) > 0 || len(mustNot) > 0 {
		query = map[string]any{"bool": boolQuery}
	}

	// Facets aligned with the v2 mapping. terms aggs only — the heavy
	// histogram/percentile aggs are exposed via dedicated endpoints later.
	aggs := map[string]any{
		"lifecycle_state_agg": map[string]any{"terms": map[string]any{"field": keywordAggField("lifecycle_state"), "size": 20}},
		"asset_type_agg":      map[string]any{"terms": map[string]any{"field": keywordAggField("asset_type"), "size": 20}},
		"owner_agg":           map[string]any{"terms": map[string]any{"field": keywordAggField("owner"), "size": 20}},
		"vendor_agg":          map[string]any{"terms": map[string]any{"field": keywordAggField("mcap.vendor_id"), "size": 20}},
		"scene_agg":           map[string]any{"terms": map[string]any{"field": keywordAggField("mcap.scene_id"), "size": 20}},
	}

	highlight := map[string]any{
		"fields": map[string]any{
			"notes":         map[string]any{},
			"owner.text":    map[string]any{},
			"reviewer.text": map[string]any{},
		},
	}

	return map[string]any{
		"query":     query,
		"from":      from,
		"size":      req.PageSize,
		"aggs":      aggs,
		"sort":      []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
		"highlight": highlight,
	}
}

func keywordAggField(field string) string {
	switch field {
	case "lifecycle_state", "asset_type", "owner", "mcap.vendor_id", "mcap.scene_id":
		return field + ".keyword"
	default:
		return field
	}
}

// buildFilterClause returns (positive, negative) clauses for a single filter.
// Exactly one of the two is non-nil for typical filters; "ne" returns
// (nil, term clause) so the caller adds it to `must_not`.
func buildFilterClause(f FilterOp) (positive, negative map[string]any) {
	if path, ok := nestedPath(f.Field); ok {
		clause := buildNestedClause(path, f)
		if f.Op == "ne" {
			return nil, clause
		}
		return clause, nil
	}

	clause := buildScalarClause(f.Field, f.Op, f.Value)
	if clause == nil {
		return nil, nil
	}
	if f.Op == "ne" {
		return nil, clause
	}
	return clause, nil
}

// nestedPath reports whether the filter field targets a nested document path.
// Only `tags.<...>` and `algos.<...>` are nested in the v2 mapping; flattened
// paths like `tags_flat.<key>` or object paths like `mcap.<col>` stay flat.
func nestedPath(field string) (string, bool) {
	switch {
	case strings.HasPrefix(field, "tags.") && !strings.HasPrefix(field, "tags_flat."):
		return "tags", true
	case strings.HasPrefix(field, "algos."):
		return "algos", true
	}
	return "", false
}

// tagInnerMatchFields are tag document sub-fields that can be filtered without
// pinning tags.key (e.g. tags.source_type:eq:algo).
var tagInnerMatchFields = map[string]bool{
	"source_type": true,
	"source_name": true,
	"confidence":  true,
	"value_num":   true,
	"value_bool":  true,
	"value":       true,
	"key":         true,
}

// nestedGroupKey returns a grouping key for nested filter consolidation.
// Filters with the same group key are combined into a single nested query.
func nestedGroupKey(path, field string) string {
	switch path {
	case "tags":
		rest := strings.TrimPrefix(field, "tags.")
		if tagInnerMatchFields[rest] {
			// Inner-only fields (e.g. tags.source_type) don't pin a key,
			// so each gets its own group to avoid merging unrelated conditions.
			return "tags:_inner:" + rest
		}
		return "tags:" + rest
	case "algos":
		rest := strings.TrimPrefix(field, "algos.")
		parts := strings.SplitN(rest, ".", 2)
		return "algos:" + parts[0]
	}
	return path + ":" + field
}

// buildGroupedNestedClauses returns the grouped must / must_not clauses for a
// nested filter without wrapping them in a nested query. The caller merges
// groups that target the same nested document and wraps them once.
type groupedNestedClauses struct {
	must    []map[string]any
	mustNot []map[string]any
}

func buildGroupedNestedClauses(path string, f FilterOp) groupedNestedClauses {
	clauses := groupedNestedClauses{}
	switch path {
	case "tags":
		rest := strings.TrimPrefix(f.Field, "tags.")
		if tagInnerMatchFields[rest] {
			clause := buildScalarClause("tags."+rest, coerceOp(f.Op), f.Value)
			if clause != nil {
				if f.Op == "ne" {
					clauses.mustNot = append(clauses.mustNot, clause)
				} else {
					clauses.must = append(clauses.must, clause)
				}
			}
			return clauses
		}
		key := rest
		clauses.must = append(clauses.must, map[string]any{"term": map[string]any{"tags.key": key}})
		valueClauses := tagKeyValueClauses(coerceOp(f.Op), f.Value)
		if f.Op == "ne" {
			clauses.mustNot = append(clauses.mustNot, valueClauses...)
		} else {
			clauses.must = append(clauses.must, valueClauses...)
		}
	case "algos":
		rest := strings.TrimPrefix(f.Field, "algos.")
		parts := strings.SplitN(rest, ".", 2)
		algoName := parts[0]
		attr := "status"
		if len(parts) == 2 {
			attr = parts[1]
		}
		if attr == "score" {
			attr = "result_score"
		}
		clauses.must = append(clauses.must, map[string]any{"term": map[string]any{"algos.name": algoName}})
		valueClause := buildScalarClause("algos."+attr, coerceOp(f.Op), f.Value)
		if valueClause != nil {
			if f.Op == "ne" {
				clauses.mustNot = append(clauses.mustNot, valueClause)
			} else {
				clauses.must = append(clauses.must, valueClause)
			}
		}
	}
	return clauses
}

// buildNestedClause produces a `nested` query that pins both the "selector"
// (e.g. tags.key=scene, algos.name=hand_tracking) and the "value/range" inside
// the same nested document, which is what users mean when they write
// `tags.scene:eq:highway` or `algos.hand_tracking.result_score:gt:0.8`.
func buildNestedClause(path string, f FilterOp) map[string]any {
	must := []map[string]any{}

	switch path {
	case "tags":
		rest := strings.TrimPrefix(f.Field, "tags.")
		if tagInnerMatchFields[rest] {
			clause := buildScalarClause("tags."+rest, coerceOp(f.Op), f.Value)
			if clause != nil {
				must = append(must, clause)
			}
			break
		}
		// tags.<key> — pin key + match value (string or numeric).
		key := rest
		must = append(must, map[string]any{"term": map[string]any{"tags.key": key}})
		for _, c := range tagKeyValueClauses(coerceOp(f.Op), f.Value) {
			must = append(must, c)
		}

	case "algos":
		// f.Field is "algos.<name>" or "algos.<name>.<attr>".
		rest := strings.TrimPrefix(f.Field, "algos.")
		parts := strings.SplitN(rest, ".", 2)
		algoName := parts[0]
		attr := "status"
		if len(parts) == 2 {
			attr = parts[1]
		}
		// Map API-friendly "score" to ES nested field result_score.
		if attr == "score" {
			attr = "result_score"
		}
		must = append(must, map[string]any{"term": map[string]any{"algos.name": algoName}})
		valueClause := buildScalarClause("algos."+attr, coerceOp(f.Op), f.Value)
		if valueClause != nil {
			must = append(must, valueClause)
		}
	}

	return map[string]any{
		"nested": map[string]any{
			"path":  path,
			"query": map[string]any{"bool": map[string]any{"must": must}},
		},
	}
}

// tagKeyValueClauses returns clauses matching tags.value or tags.value_num for
// a pinned tags.key (caller adds the key term separately).
func tagKeyValueClauses(op, value string) []map[string]any {
	if op == "between" {
		lo, hi, ok := splitBetween(value)
		if !ok {
			return nil
		}
		if isNumericString(lo) && isNumericString(hi) {
			return []map[string]any{{
				"range": map[string]any{"tags.value_num": map[string]any{"gte": lo, "lte": hi}},
			}}
		}
		return []map[string]any{{
			"range": map[string]any{"tags.value": map[string]any{"gte": lo, "lte": hi}},
		}}
	}
	if op == "gt" || op == "gte" || op == "lt" || op == "lte" {
		field := "tags.value"
		if isNumericString(value) {
			field = "tags.value_num"
		}
		return []map[string]any{buildScalarClause(field, op, value)}
	}
	if op == "eq" || op == "ne" {
		if isNumericString(value) {
			return []map[string]any{buildScalarClause("tags.value_num", op, value)}
		}
		return []map[string]any{buildScalarClause("tags.value", op, value)}
	}
	return nil
}

func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// coerceOp downgrades "ne" inside a nested clause to "eq", because negation
// at the nested-document level is awkward (requires must_not at the outer
// level). The caller of buildNestedClause hoists "ne" to the outer must_not.
func coerceOp(op string) string {
	if op == "ne" {
		return "eq"
	}
	return op
}

// buildScalarClause builds a single term/range clause for a flat field.
// Returns nil for unknown ops.
func buildScalarClause(field, op, value string) map[string]any {
	switch op {
	case "eq", "ne":
		return map[string]any{"term": map[string]any{field: value}}
	case "gt", "gte", "lt", "lte":
		return map[string]any{"range": map[string]any{field: map[string]any{op: value}}}
	case "between":
		lo, hi, ok := splitBetween(value)
		if !ok {
			return nil
		}
		return map[string]any{"range": map[string]any{field: map[string]any{"gte": lo, "lte": hi}}}
	case "ilike":
		pattern := "*" + elasticsearchWildcardQuote(value) + "*"
		return map[string]any{
			"wildcard": map[string]any{
				field: map[string]any{"value": pattern, "case_insensitive": true},
			},
		}
	}
	return nil
}

// elasticsearchWildcardQuote escapes * and \ for ES wildcard syntax, then lowercases
// so case_insensitive matching is predictable on keyword fields.
func elasticsearchWildcardQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `*`, `\*`)
	s = strings.ReplaceAll(s, `?`, `\?`)
	return strings.ToLower(s)
}

// splitBetween parses "lower,upper". Both sides must be non-empty.
func splitBetween(value string) (string, string, bool) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	lo := strings.TrimSpace(parts[0])
	hi := strings.TrimSpace(parts[1])
	if lo == "" || hi == "" {
		return "", "", false
	}
	return lo, hi, true
}

func (c *Client) doSearch(ctx context.Context, body map[string]any) (*SearchResponse, error) {
	start := time.Now()
	outcome := "error"
	defer func() {
		metrics.ElasticsearchRequestsTotal.WithLabelValues("search", outcome).Inc()
		metrics.ElasticsearchRequestDurationSeconds.WithLabelValues("search", outcome).Observe(time.Since(start).Seconds())
	}()

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", c.baseURL, c.index)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("elasticsearch: status %d: %s", resp.StatusCode, string(respBody))
	}

	var osResp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID        string              `json:"_id"`
				Score     float64             `json:"_score"`
				Source    map[string]any      `json:"_source"`
				Highlight map[string][]string `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]struct {
			Buckets []AggBucket `json:"buckets"`
		} `json:"aggregations"`
	}

	if err := json.Unmarshal(respBody, &osResp); err != nil {
		return nil, fmt.Errorf("elasticsearch: unmarshal response: %w", err)
	}

	result := &SearchResponse{
		Total:        osResp.Hits.Total.Value,
		Hits:         make([]SearchHit, 0, len(osResp.Hits.Hits)),
		Aggregations: make(map[string][]AggBucket),
	}

	for _, h := range osResp.Hits.Hits {
		result.Hits = append(result.Hits, SearchHit{
			ID:        h.ID,
			Score:     h.Score,
			Source:    h.Source,
			Highlight: h.Highlight,
		})
	}

	for name, agg := range osResp.Aggregations {
		result.Aggregations[name] = agg.Buckets
	}

	outcome = "ok"
	return result, nil
}

// ---------- BulkIndex ----------

// BulkIndexDoc is a document to be indexed.
type BulkIndexDoc struct {
	ID  string
	Doc map[string]any
}

// BulkItemResult holds the per-document outcome of a bulk index operation.
type BulkItemResult struct {
	ID     string // document _id
	Status int    // HTTP status code for this item
	Error  string // non-empty when the item failed
}

// BulkIndexResult holds the aggregate outcome of a BulkIndex call.
type BulkIndexResult struct {
	Succeeded []BulkItemResult // items with status < 300
	Failed    []BulkItemResult // items with status >= 300
}

// BulkIndex sends documents to Elasticsearch via the _bulk API.
// On transport-level or HTTP-level failure it returns a non-nil error.
// On success (HTTP 2xx) it returns per-doc results so the caller can
// handle partial failures (some docs succeed, some fail).
func (c *Client) BulkIndex(ctx context.Context, docs []BulkIndexDoc) (*BulkIndexResult, error) {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.ElasticsearchRequestsTotal.WithLabelValues("bulk_index", outcome).Inc()
		metrics.ElasticsearchRequestDurationSeconds.WithLabelValues("bulk_index", outcome).Observe(time.Since(start).Seconds())
	}()

	if len(docs) == 0 {
		return &BulkIndexResult{}, nil
	}

	var buf bytes.Buffer
	for _, d := range docs {
		action := map[string]any{
			"index": map[string]any{
				"_index": c.index,
				"_id":    d.ID,
			},
		}
		actionLine, _ := json.Marshal(action)
		buf.Write(actionLine)
		buf.WriteByte('\n')
		docLine, _ := json.Marshal(d.Doc)
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	url := fmt.Sprintf("%s/_bulk", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: bulk request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		outcome = "error"
		return nil, fmt.Errorf("elasticsearch: bulk failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: read bulk response: %w", err)
	}

	if resp.StatusCode >= 300 {
		outcome = "http_error"
		return nil, fmt.Errorf("elasticsearch: bulk status %d: %s", resp.StatusCode, string(respBody))
	}

	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []struct {
			Index struct {
				ID     string `json:"_id"`
				Status int    `json:"status"`
				Error  *struct {
					Type   string `json:"type"`
					Reason string `json:"reason"`
				} `json:"error,omitempty"`
			} `json:"index"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &bulkResp); err != nil {
		outcome = "error"
		return nil, fmt.Errorf("elasticsearch: unmarshal bulk response: %w", err)
	}

	result := &BulkIndexResult{}
	for i, item := range bulkResp.Items {
		docID := item.Index.ID
		if docID == "" && i < len(docs) {
			docID = docs[i].ID
		}
		bir := BulkItemResult{
			ID:     docID,
			Status: item.Index.Status,
		}
		if item.Index.Error != nil {
			bir.Error = item.Index.Error.Type + ": " + item.Index.Error.Reason
		}
		if item.Index.Status < 300 {
			result.Succeeded = append(result.Succeeded, bir)
		} else {
			result.Failed = append(result.Failed, bir)
		}
	}
	if len(result.Failed) > 0 {
		outcome = "partial_error"
	}
	return result, nil
}

// DeleteDocument removes a document from the index by id (asset_id). Idempotent: 404 is treated as success.
func (c *Client) DeleteDocument(ctx context.Context, id string) error {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.ElasticsearchRequestsTotal.WithLabelValues("delete_document", outcome).Inc()
		metrics.ElasticsearchRequestDurationSeconds.WithLabelValues("delete_document", outcome).Observe(time.Since(start).Seconds())
	}()

	if id == "" {
		return nil
	}
	url := fmt.Sprintf("%s/%s/_doc/%s", strings.TrimRight(c.baseURL, "/"), c.index, id)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("elasticsearch: delete request: %w", err)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		outcome = "error"
		return fmt.Errorf("elasticsearch: delete failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 300 {
		outcome = "http_error"
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch: delete status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteAllDocuments removes every document in the configured index while
// preserving the index and its mapping. Returns the number of deleted docs.
func (c *Client) DeleteAllDocuments(ctx context.Context) (int64, error) {
	url := fmt.Sprintf(
		"%s/%s/_delete_by_query?conflicts=proceed&refresh=true",
		strings.TrimRight(c.baseURL, "/"),
		c.index,
	)
	body := strings.NewReader(`{"query":{"match_all":{}}}`)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return 0, fmt.Errorf("elasticsearch: delete_by_query request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("elasticsearch: delete_by_query failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("elasticsearch: read delete_by_query response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return 0, fmt.Errorf("elasticsearch: delete_by_query status %d: %s", resp.StatusCode, string(respBody))
	}

	var deleteResp struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.Unmarshal(respBody, &deleteResp); err != nil {
		return 0, fmt.Errorf("elasticsearch: unmarshal delete_by_query response: %w", err)
	}
	return deleteResp.Deleted, nil
}

// Count returns the total document count in the index (approximate for large indices).
func (c *Client) Count(ctx context.Context) (int64, error) {
	url := fmt.Sprintf("%s/%s/_count", strings.TrimRight(c.baseURL, "/"), c.index)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("elasticsearch: count request: %w", err)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("elasticsearch: count failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("elasticsearch: count read: %w", err)
	}
	if resp.StatusCode >= 300 {
		return 0, fmt.Errorf("elasticsearch: count status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Count int64 `json:"count"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, fmt.Errorf("elasticsearch: count unmarshal: %w", err)
	}
	return parsed.Count, nil
}

// Ping checks if Elasticsearch is reachable.
func (c *Client) Ping(ctx context.Context) error {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.ElasticsearchRequestsTotal.WithLabelValues("ping", outcome).Inc()
		metrics.ElasticsearchRequestDurationSeconds.WithLabelValues("ping", outcome).Observe(time.Since(start).Seconds())
	}()

	url := fmt.Sprintf("%s/_cluster/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		outcome = "error"
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		outcome = "error"
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		outcome = "http_error"
		return fmt.Errorf("elasticsearch: ping status %d", resp.StatusCode)
	}
	return nil
}
