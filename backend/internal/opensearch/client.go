// Package opensearch provides a lightweight OpenSearch client for search and
// bulk-index operations against the "assets" index.
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps HTTP calls to an OpenSearch cluster.
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

// SearchRequest describes the parameters accepted by the search handler.
type SearchRequest struct {
	Query    string            // free-text query (multi_match)
	Filters  map[string]string // field → value term filters
	Page     int
	PageSize int
}

// SearchHit is a single document returned by OpenSearch.
type SearchHit struct {
	ID     string         `json:"_id"`
	Score  float64        `json:"_score"`
	Source map[string]any `json:"_source"`
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
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 200 {
		req.PageSize = 20
	}

	from := (req.Page - 1) * req.PageSize

	// Build bool query
	must := make([]map[string]any, 0)
	filter := make([]map[string]any, 0)

	if req.Query != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  req.Query,
				"fields": []string{"notes", "owner", "reviewer", "task", "env", "asset_id"},
				"type":   "best_fields",
			},
		})
	}

	for field, value := range req.Filters {
		filter = append(filter, map[string]any{
			"term": map[string]any{field: value},
		})
	}

	boolQuery := map[string]any{}
	if len(must) > 0 {
		boolQuery["must"] = must
	}
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}

	// If no must/filter, match_all
	query := map[string]any{"match_all": map[string]any{}}
	if len(must) > 0 || len(filter) > 0 {
		query = map[string]any{"bool": boolQuery}
	}

	// Aggregations for facets
	aggs := map[string]any{
		"status_agg": map[string]any{"terms": map[string]any{"field": "status", "size": 20}},
		"env_agg":    map[string]any{"terms": map[string]any{"field": "env", "size": 20}},
		"owner_agg":  map[string]any{"terms": map[string]any{"field": "owner", "size": 20}},
		"task_agg":   map[string]any{"terms": map[string]any{"field": "task", "size": 20}},
	}

	body := map[string]any{
		"query": query,
		"from":  from,
		"size":  req.PageSize,
		"aggs":  aggs,
		"sort":  []map[string]any{{"updated_at": map[string]any{"order": "desc"}}},
	}

	return c.doSearch(ctx, body)
}

func (c *Client) doSearch(ctx context.Context, body map[string]any) (*SearchResponse, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("opensearch: marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", c.baseURL, c.index)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("opensearch: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("opensearch: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("opensearch: read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("opensearch: status %d: %s", resp.StatusCode, string(respBody))
	}

	var osResp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string         `json:"_id"`
				Score  float64        `json:"_score"`
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]struct {
			Buckets []AggBucket `json:"buckets"`
		} `json:"aggregations"`
	}

	if err := json.Unmarshal(respBody, &osResp); err != nil {
		return nil, fmt.Errorf("opensearch: unmarshal response: %w", err)
	}

	result := &SearchResponse{
		Total:        osResp.Hits.Total.Value,
		Hits:         make([]SearchHit, 0, len(osResp.Hits.Hits)),
		Aggregations: make(map[string][]AggBucket),
	}

	for _, h := range osResp.Hits.Hits {
		result.Hits = append(result.Hits, SearchHit{
			ID:     h.ID,
			Score:  h.Score,
			Source: h.Source,
		})
	}

	for name, agg := range osResp.Aggregations {
		result.Aggregations[name] = agg.Buckets
	}

	return result, nil
}

// ---------- BulkIndex ----------

// BulkIndexDoc is a document to be indexed.
type BulkIndexDoc struct {
	ID  string
	Doc map[string]any
}

// BulkIndex sends documents to OpenSearch via the _bulk API.
// Returns the number of successfully indexed documents.
func (c *Client) BulkIndex(ctx context.Context, docs []BulkIndexDoc) (int, error) {
	if len(docs) == 0 {
		return 0, nil
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
		return 0, fmt.Errorf("opensearch: bulk request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("opensearch: bulk failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("opensearch: read bulk response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return 0, fmt.Errorf("opensearch: bulk status %d: %s", resp.StatusCode, string(respBody))
	}

	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []struct {
			Index struct {
				Status int `json:"status"`
			} `json:"index"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &bulkResp); err != nil {
		return 0, fmt.Errorf("opensearch: unmarshal bulk response: %w", err)
	}

	ok := 0
	for _, item := range bulkResp.Items {
		if item.Index.Status < 300 {
			ok++
		}
	}
	return ok, nil
}

// Ping checks if OpenSearch is reachable.
func (c *Client) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/_cluster/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("opensearch: ping status %d", resp.StatusCode)
	}
	return nil
}
