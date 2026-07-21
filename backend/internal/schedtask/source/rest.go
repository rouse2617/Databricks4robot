package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// RestConfig is the config-driven REST-JSON source. One rule = one RestConfig.
// The shape stays flat + declarative so the UI can render it as a form and so
// "different endpoint / different API" is purely different values here (no
// code / deploy required).
//
// Grace's /grace/video_steps is expressible as a RestConfig (see restsource
// tests + design.md). If a real second REST API needs a shape we haven't
// exposed yet, extend RestConfig rather than forking a new source type.
type RestConfig struct {
	BaseURL        string            `json:"base_url"`        // e.g. https://grace.example.com/api
	Path           string            `json:"path"`            // e.g. /grace/video_steps
	Method         string            `json:"method"`          // GET (default); v1 doesn't need POST
	Auth           AuthConfig        `json:"auth"`            // credential lookup via secret_ref
	Query          QueryConfig       `json:"query"`           // static filters + time-window templating
	Paging         PagingConfig      `json:"paging"`          // v1: page/size
	IDPath         string            `json:"id_path"`         // e.g. "data[].video_id"
	Headers        map[string]string `json:"headers"`         // additional fixed headers
	UserAgent      string            `json:"user_agent"`      // override; defaults to a browser-like UA
	TimeoutSeconds int               `json:"timeout_seconds"` // per-request; default 30
}

// AuthConfig references (never inlines) credentials. Type governs the header;
// SecretRef is resolved via SecretResolver at Fetch time.
type AuthConfig struct {
	Type       string `json:"type"`        // "basic" | "bearer" | "header_key" | "none"
	Username   string `json:"username"`    // basic only; NOT a secret
	SecretRef  string `json:"secret_ref"`  // secret-manager key for the actual credential
	HeaderName string `json:"header_name"` // header_key only, e.g. "X-Api-Key"
}

// QueryConfig describes filters + optional time window on a time field.
//
//   - Static: static key/value pairs (multi-value allowed) appended to the URL.
//   - TimeField: when set together with Window, adds
//     <TimeField>:gte:<start> and <TimeField>:lt:<end>
//     rendered via TimeWindowTemplate (see below). This mirrors Grace's
//     filter=last_status_at:gte:… syntax while remaining generic — sources
//     using ?start=/?end= can express that too via TimeQueryStart/End.
//   - TimeQueryStart / TimeQueryEnd: if set, emit start/end as ordinary query
//     params with these names (e.g. "since"/"until"). Ignored when
//     TimeField is set (they compose the same idea via one canonical
//     "filter=…" style).
//   - TimeFormat: "rfc3339" (default) | "unix"
type QueryConfig struct {
	Static         map[string][]string `json:"static"`
	FilterParam    string              `json:"filter_param"`     // e.g. "filter" for Grace's repeated filter= keys; empty = don't emit as filter= repeats
	TimeField      string              `json:"time_field"`       // e.g. "last_status_at"
	TimeFormat     string              `json:"time_format"`      // "rfc3339" (default) | "unix"
	TimeQueryStart string              `json:"time_query_start"` // e.g. "since" — used when FilterParam is empty
	TimeQueryEnd   string              `json:"time_query_end"`   // e.g. "until"
}

// PagingConfig — v1 supports page/size paging (Grace's shape). More styles
// (cursor, offset/limit) are additive without breaking existing configs.
type PagingConfig struct {
	Mode      string `json:"mode"`       // "page_size" (default) | "none"
	PageParam string `json:"page_param"` // e.g. "page"
	SizeParam string `json:"size_param"` // e.g. "size"
	PageSize  int    `json:"page_size"`  // default 200
	TotalPath string `json:"total_path"` // e.g. "total"; empty = paginate until an empty page
	DataPath  string `json:"data_path"`  // e.g. "data"; matches the array prefix in IDPath
}

// restSource is an AssetSource driven by a RestConfig.
type restSource struct {
	cfg      RestConfig
	resolver SecretResolver
	// http is injectable for tests; nil = build one from cfg.TimeoutSeconds.
	http *http.Client
}

// NewREST constructs a REST source. Validation is intentionally light: we
// surface errors at Fetch time so bad config is caught with real context.
func NewREST(cfg RestConfig, resolver SecretResolver) AssetSource {
	if strings.TrimSpace(cfg.Method) == "" {
		cfg.Method = http.MethodGet
	}
	if cfg.Paging.Mode == "" {
		cfg.Paging.Mode = "page_size"
	}
	if cfg.Paging.PageSize <= 0 {
		cfg.Paging.PageSize = 200
	}
	if cfg.Paging.PageParam == "" {
		cfg.Paging.PageParam = "page"
	}
	if cfg.Paging.SizeParam == "" {
		cfg.Paging.SizeParam = "size"
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 30
	}
	if cfg.Query.TimeFormat == "" {
		cfg.Query.TimeFormat = "rfc3339"
	}
	if cfg.UserAgent == "" {
		// Grace sits behind Cloudflare Pages, which blocks Go's default UA
		// with 403; use a browser-like UA. See backend/internal/grace/client.go.
		cfg.UserAgent = "curl/8"
	}
	return &restSource{cfg: cfg, resolver: resolver}
}

// Fetch is the AssetSource contract. For the ids-mode window, we return the
// window's ids verbatim — no HTTP call — so ad-hoc "run these ids now" doesn't
// pay a network round-trip.
func (r *restSource) Fetch(ctx context.Context, cursor string, window Window) ([]string, string, error) {
	if len(window.IDs) > 0 {
		out := append([]string(nil), window.IDs...)
		return out, cursor, nil
	}

	if strings.TrimSpace(r.cfg.BaseURL) == "" {
		return nil, "", fmt.Errorf("restsource: base_url is empty")
	}

	authHeader, err := r.buildAuthHeader(ctx)
	if err != nil {
		return nil, "", err
	}

	httpClient := r.http
	if httpClient == nil {
		httpClient = &http.Client{Timeout: time.Duration(r.cfg.TimeoutSeconds) * time.Second}
	}

	var all []string
	nextCursor := cursor
	page := 1
	for {
		q, err := r.buildQuery(window, page)
		if err != nil {
			return nil, "", err
		}
		body, err := r.doRequest(ctx, httpClient, q, authHeader)
		if err != nil {
			return nil, "", err
		}

		ids, err := extractIDs(body, r.cfg.IDPath)
		if err != nil {
			return nil, "", fmt.Errorf("restsource: extract ids page %d: %w", page, err)
		}
		all = append(all, ids...)

		// Advance the incremental cursor to the max observed time_field value
		// in this batch (only when time_field is being used).
		if r.cfg.Query.TimeField != "" {
			if maxObs := extractMaxTimeField(body, r.cfg.Paging.DataPath, r.cfg.Query.TimeField); maxObs != "" {
				if isLater(maxObs, nextCursor) {
					nextCursor = maxObs
				}
			}
		}

		// Termination: honor TotalPath if given, else stop when a page returns
		// no ids (Grace returns Data=[] on the page past the last one).
		if r.cfg.Paging.Mode == "none" {
			break
		}
		if len(ids) == 0 {
			break
		}
		if r.cfg.Paging.TotalPath != "" {
			total := extractInt(body, r.cfg.Paging.TotalPath)
			if total > 0 && len(all) >= total {
				break
			}
		}
		page++
	}
	return all, nextCursor, nil
}

func (r *restSource) buildQuery(window Window, page int) (url.Values, error) {
	q := url.Values{}
	// Static filters (support repeated values, e.g. Grace's multi filter=…).
	for k, vs := range r.cfg.Query.Static {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	// Time window.
	tf := r.cfg.Query.TimeField
	if (window.Start != nil || window.End != nil) && tf != "" {
		start := formatTime(window.Start, r.cfg.Query.TimeFormat)
		end := formatTime(window.End, r.cfg.Query.TimeFormat)
		switch {
		case r.cfg.Query.FilterParam != "":
			// Grace-style repeated filter= keys, e.g. filter=last_status_at:gte:<t>
			if start != "" {
				q.Add(r.cfg.Query.FilterParam, tf+":gte:"+start)
			}
			if end != "" {
				q.Add(r.cfg.Query.FilterParam, tf+":lt:"+end)
			}
		case r.cfg.Query.TimeQueryStart != "" || r.cfg.Query.TimeQueryEnd != "":
			if start != "" && r.cfg.Query.TimeQueryStart != "" {
				q.Set(r.cfg.Query.TimeQueryStart, start)
			}
			if end != "" && r.cfg.Query.TimeQueryEnd != "" {
				q.Set(r.cfg.Query.TimeQueryEnd, end)
			}
		default:
			return nil, fmt.Errorf("restsource: time_field=%q needs filter_param or time_query_start/end", tf)
		}
	}
	// Paging.
	if r.cfg.Paging.Mode == "page_size" {
		q.Set(r.cfg.Paging.PageParam, strconv.Itoa(page))
		q.Set(r.cfg.Paging.SizeParam, strconv.Itoa(r.cfg.Paging.PageSize))
	}
	return q, nil
}

func (r *restSource) doRequest(ctx context.Context, client *http.Client, q url.Values, authHeader string) ([]byte, error) {
	u := strings.TrimRight(r.cfg.BaseURL, "/") + r.cfg.Path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, r.cfg.Method, u, nil)
	if err != nil {
		return nil, fmt.Errorf("restsource: build request: %w", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if r.cfg.Auth.Type == "header_key" && r.cfg.Auth.HeaderName != "" {
		// header_key auth sets a named header instead of Authorization.
		req.Header.Del("Authorization")
		req.Header.Set(r.cfg.Auth.HeaderName, authHeader)
	}
	req.Header.Set("User-Agent", r.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")
	for k, v := range r.cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("restsource: GET %s: %w", u, err)
	}
	defer resp.Body.Close()
	// Cap the response body so a misconfigured / hostile upstream can't OOM the
	// backend by returning a huge payload (id_path extraction still runs in
	// memory). 10MB is well above any realistic Grace-shaped page.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("restsource: %s → %d: %s", u, resp.StatusCode, truncate(body, 200))
	}
	return body, nil
}

// buildAuthHeader returns the value to put in the auth header (Authorization
// for basic/bearer; the named header for header_key). Empty when Auth.Type is
// "none" or empty.
func (r *restSource) buildAuthHeader(ctx context.Context) (string, error) {
	switch strings.ToLower(strings.TrimSpace(r.cfg.Auth.Type)) {
	case "", "none":
		return "", nil
	case "basic":
		if r.cfg.Auth.SecretRef == "" {
			return "", fmt.Errorf("restsource: basic auth requires secret_ref")
		}
		pw, err := r.resolveSecret(ctx, r.cfg.Auth.SecretRef)
		if err != nil {
			return "", err
		}
		return "Basic " + basicAuthValue(r.cfg.Auth.Username, pw), nil
	case "bearer":
		if r.cfg.Auth.SecretRef == "" {
			return "", fmt.Errorf("restsource: bearer auth requires secret_ref")
		}
		tok, err := r.resolveSecret(ctx, r.cfg.Auth.SecretRef)
		if err != nil {
			return "", err
		}
		return "Bearer " + tok, nil
	case "header_key":
		if r.cfg.Auth.SecretRef == "" || r.cfg.Auth.HeaderName == "" {
			return "", fmt.Errorf("restsource: header_key auth requires secret_ref and header_name")
		}
		v, err := r.resolveSecret(ctx, r.cfg.Auth.SecretRef)
		if err != nil {
			return "", err
		}
		return v, nil
	default:
		return "", fmt.Errorf("restsource: unsupported auth type %q", r.cfg.Auth.Type)
	}
}

func (r *restSource) resolveSecret(ctx context.Context, ref string) (string, error) {
	if r.resolver == nil {
		// Fail closed. Never fall back to env/plaintext — that's exactly the
		// leak the "secret_ref only" rule exists to prevent.
		return "", fmt.Errorf("restsource: secret ref %q needs a resolver but none was wired", ref)
	}
	v, err := r.resolver.Resolve(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("restsource: resolve secret %q: %w", ref, err)
	}
	if v == "" {
		return "", fmt.Errorf("restsource: secret %q resolved empty", ref)
	}
	return v, nil
}

// basicAuthValue base64-encodes "user:pass" the same way net/http does.
// Kept local so tests don't need to construct an http.Request just to check
// the header shape.
func basicAuthValue(user, pass string) string {
	req, _ := http.NewRequest(http.MethodGet, "http://x", nil)
	req.SetBasicAuth(user, pass)
	return strings.TrimPrefix(req.Header.Get("Authorization"), "Basic ")
}

// extractIDs walks a "prefix[].field" idPath through the JSON body.
// Supports at most one "[].". This is the shape Grace and equivalents use;
// extend later if a real API needs deeper nesting.
func extractIDs(body []byte, idPath string) ([]string, error) {
	if idPath == "" {
		return nil, fmt.Errorf("id_path is empty")
	}
	dot := strings.Index(idPath, "[].")
	if dot < 0 {
		return nil, fmt.Errorf("id_path %q must contain \"[].\"", idPath)
	}
	arrKey := idPath[:dot]
	field := idPath[dot+3:]
	if arrKey == "" || field == "" {
		return nil, fmt.Errorf("id_path %q: empty array key or field", idPath)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	arr, ok := raw[arrKey]
	if !ok {
		return nil, nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(arr, &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		v, ok := it[field]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			// Non-string ids: coerce via number → string as a courtesy.
			var n json.Number
			if err2 := json.Unmarshal(v, &n); err2 == nil {
				s = n.String()
			} else {
				continue
			}
		}
		if s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

// extractMaxTimeField finds the max value of timeField across items under
// dataPath, returning it as-is (opaque string cursor).
func extractMaxTimeField(body []byte, dataPath, timeField string) string {
	if dataPath == "" || timeField == "" {
		return ""
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}
	arr, ok := raw[dataPath]
	if !ok {
		return ""
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(arr, &items); err != nil {
		return ""
	}
	var maxV string
	for _, it := range items {
		v, ok := it[timeField]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			var n json.Number
			if err2 := json.Unmarshal(v, &n); err2 == nil {
				s = n.String()
			} else {
				continue
			}
		}
		if s == "" {
			continue
		}
		if isLater(s, maxV) {
			maxV = s
		}
	}
	return maxV
}

// extractInt reads a top-level integer field like "total" from the body.
func extractInt(body []byte, path string) int {
	if path == "" {
		return 0
	}
	var raw map[string]json.Number
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0
	}
	v, ok := raw[path]
	if !ok {
		return 0
	}
	n, err := v.Int64()
	if err != nil {
		return 0
	}
	return int(n)
}

// isLater compares two opaque cursor strings. When both parse as RFC3339
// timestamps we compare them via time.After so different timezone offsets
// (e.g. "…Z" vs "…+08:00" for the same instant) compare correctly. Otherwise
// we fall back to lexicographical order, which is what unix-second or
// equal-width numeric cursors want.
func isLater(a, b string) bool {
	if b == "" {
		return a != ""
	}
	if ta, err := time.Parse(time.RFC3339, a); err == nil {
		if tb, err := time.Parse(time.RFC3339, b); err == nil {
			return ta.After(tb)
		}
	}
	return a > b
}

func formatTime(t *time.Time, format string) string {
	if t == nil || t.IsZero() {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "unix":
		return strconv.FormatInt(t.Unix(), 10)
	default:
		return t.UTC().Format(time.RFC3339)
	}
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "…"
	}
	return string(b)
}
