// Package grace is a small reusable client for the Grace API. Video-duration
// sync (CYB-3072) is its first consumer; future Grace data needs should add
// methods here rather than open-coding HTTP calls elsewhere.
package grace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds Grace API connection settings.
type Config struct {
	BaseURL  string // e.g. https://dev.cyber-grace.pages.dev/api
	Username string
	Password string
	Timeout  time.Duration
}

// ConfigFromEnv reads Grace settings from the environment. Password comes from
// GRACE_PASSWORD, which (like grace-sync) may be a plain string or a JSON secret
// blob {"AUTH_PASSWORD": "..."} mounted from Secret Manager grace-api-dev.
func ConfigFromEnv() Config {
	return Config{
		BaseURL:  strings.TrimSpace(os.Getenv("GRACE_API_URL")),
		Username: strings.TrimSpace(os.Getenv("GRACE_USERNAME")),
		Password: parsePassword(os.Getenv("GRACE_PASSWORD")),
		Timeout:  30 * time.Second,
	}
}

// parsePassword accepts a plain password or a JSON object with AUTH_PASSWORD
// (GCP Secret Manager often stores the whole grace-api-dev JSON in one var).
func parsePassword(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "{") {
		var creds struct {
			AuthPassword string `json:"AUTH_PASSWORD"`
		}
		if err := json.Unmarshal([]byte(raw), &creds); err == nil && creds.AuthPassword != "" {
			return creds.AuthPassword
		}
	}
	return raw
}

// Enabled reports whether the client has enough config to make calls.
func (c Config) Enabled() bool {
	return c.BaseURL != "" && c.Username != "" && c.Password != ""
}

// Client talks to the Grace API.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient constructs a Grace client.
func NewClient(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// Enabled reports whether the underlying config is usable.
func (c *Client) Enabled() bool { return c.cfg.Enabled() }

// get performs an authenticated GET and returns the raw body. It sets a
// browser-like User-Agent because Grace sits behind Cloudflare Pages, which
// blocks default Go/urllib user agents with 403.
func (c *Client) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	u := strings.TrimRight(c.cfg.BaseURL, "/") + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	req.Header.Set("User-Agent", "curl/8")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("grace GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grace GET %s: status %d: %s", path, resp.StatusCode, truncate(body, 200))
	}
	return body, nil
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n])
	}
	return string(b)
}

// video is the subset of /grace/videos we consume.
type video struct {
	ID          string  `json:"id"`
	DurationSec float64 `json:"duration_sec"`
}

type videoListResp struct {
	Data  []video `json:"data"`
	Total int     `json:"total"`
}

const videosPageSize = 1000

// FetchVideoDurations paginates /grace/videos and returns video_id -> duration
// (seconds) for every video that has a positive recorded duration.
func (c *Client) FetchVideoDurations(ctx context.Context) (map[string]float64, error) {
	out := make(map[string]float64)
	page := 1
	fetched := 0
	for {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("size", strconv.Itoa(videosPageSize))
		body, err := c.get(ctx, "/grace/videos", q)
		if err != nil {
			return nil, err
		}
		var list videoListResp
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("grace videos page %d: decode: %w", page, err)
		}
		for _, v := range list.Data {
			if v.ID != "" && v.DurationSec > 0 {
				out[v.ID] = v.DurationSec
			}
		}
		fetched += len(list.Data)
		if len(list.Data) == 0 || fetched >= list.Total {
			break
		}
		page++
	}
	return out, nil
}
