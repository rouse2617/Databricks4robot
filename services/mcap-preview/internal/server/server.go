// Package server wires the HTTP handlers for mcap-preview. It is split out of
// cmd/server so unit tests can exercise the routing logic without spawning a
// real GCS-backed reader.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/gcsrs"
	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/manifest"
)

// Config_OpenMCAP is the function shape used by Config.OpenMCAP. Exported as
// a named type so cmd/server can spell it without a long function literal.
type Config_OpenMCAP = func(ctx context.Context, mcapURI string) (rs io.ReadSeeker, stats *gcsrs.Stats, closer func(), err error)

// Config holds the runtime knobs the HTTP layer needs.
type Config struct {
	// UpstreamBaseURL points at cyber-databrew backend's /api/v1 root parent
	// (e.g. http://cyber-databrew-backend:8080). Must NOT include /api/v1.
	UpstreamBaseURL string

	// GraceTokenPassthrough controls whether incoming X-Grace-Token is
	// forwarded to the upstream call. In phase 0 this is always true; the
	// flag exists so we can wire OpenFGA / service-to-service auth later
	// without bypassing token validation.
	GraceTokenPassthrough bool

	// HTTPClient calls the upstream backend. tests override.
	HTTPClient *http.Client

	// OpenMCAP returns a ReadSeeker over the MCAP bytes named by mcapURI
	// plus an optional GCS IO stats snapshot. The closer is invoked when
	// the handler is done; tests inject a no-op closer over an in-memory
	// fixture.
	OpenMCAP Config_OpenMCAP

	// StrictWindowValidation enforces hard-fail on window/log_time timebase
	// mismatch instead of silently falling back to stats start.
	StrictWindowValidation bool

	// MCAPReaderCacheTTL controls how long an open MCAP reader (with its
	// warmed-up page cache) stays usable across requests. Defaults to 5
	// minutes when unset; set to a negative duration to disable caching
	// (legacy "open fresh every time" behaviour, useful for tests).
	MCAPReaderCacheTTL time.Duration

	// mcapCache is the in-process reader cache. Populated by New(). Tests
	// that don't go through New() leave it nil and Acquire transparently
	// falls back to cfg.OpenMCAP.
	mcapCache *mcapReaderCache
}

// acquireMCAP is the cache-aware front door every handler should use instead
// of calling cfg.OpenMCAP directly. It preserves the OpenMCAP signature so
// the call sites stay one-line.
func (cfg *Config) acquireMCAP(ctx context.Context, mcapURI string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
	if cfg.mcapCache != nil {
		return cfg.mcapCache.Acquire(ctx, mcapURI)
	}
	return cfg.OpenMCAP(ctx, mcapURI)
}

// New constructs a configured Gin engine. The caller owns lifecycle.
func New(cfg Config) *gin.Engine {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	ttl := cfg.MCAPReaderCacheTTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	if ttl > 0 && cfg.OpenMCAP != nil {
		cfg.mcapCache = newMCAPReaderCache(ttl, cfg.OpenMCAP)
	}
	r := gin.New()
	r.Use(requestID())
	r.Use(accessLogger())
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1/preview")
	api.GET("/assets/:id/manifest", manifestHandler(cfg))
	api.GET("/assets/:id/segment.mp4", segmentHandler(cfg))
	api.POST("/assets/:id/prewarm", prewarmHandler(cfg))
	return r
}

func prewarmHandler(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		assetID := c.Param("id")
		token := extractGraceToken(c)
		if token == "" {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "missing X-Grace-Token")
			return
		}
		loc, status, err := fetchLocator(c, cfg, assetID, token)
		if err != nil {
			if status == 0 {
				httpresp.Error(c, http.StatusBadGateway, httpresp.CodeUpstreamError, err.Error(), nil)
			}
			return
		}
		rs, _, closer, err := cfg.acquireMCAP(c.Request.Context(), loc.Mcap.McapURI)
		if err != nil {
			httpresp.Error(c, http.StatusBadGateway, httpresp.CodeUpstreamError, err.Error(), nil)
			return
		}
		if closer != nil {
			defer closer()
		}
		topic, err := pickTopic(rs, c.Query("topic"))
		if err != nil {
			httpresp.Error(c, http.StatusUnprocessableEntity, codeNoCandidateTopic, err.Error(), nil)
			return
		}
		startNs := uint64(0)
		endNs := uint64(0)
		if loc.Window.StartTimestampNs > 0 {
			startNs = uint64(loc.Window.StartTimestampNs)
		}
		if loc.Window.EndTimestampNs > 0 {
			endNs = uint64(loc.Window.EndTimestampNs)
		}
		if codec, ok := manifest.DetectTopicCodec(rs, topic); ok && codec == "h265" {
			key := previewMP4Cache.key(assetID, topic, fmt.Sprintf("%d", startNs), fmt.Sprintf("%d", endNs), "h265-to-h264-v2")
			mcapURI := loc.Mcap.McapURI
			go func() {
				bg, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
			_, _ = previewMP4Cache.getOrBuild(bg, key, func(dst string) error {
				f, err := os.Create(dst)
				if err != nil {
					return err
				}
				defer f.Close()
				return transcodeWindowToWriter(
					bg,
					func(ctx context.Context) (io.ReadSeeker, func(), error) {
						rs3, _, closer3, err := cfg.acquireMCAP(ctx, mcapURI)
						if err != nil {
							return nil, nil, err
						}
						return rs3, closer3, nil
					},
					topic,
					startNs,
					endNs,
					f,
				)
			})
			}()
		}
		c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
	}
}

func accessLogger() gin.HandlerFunc {
	// Structured log line per request: method path status latency bytes request_id client_ip ua.
	// Kept text-mode so it works with current slog/stdout pipeline.
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		ua := strings.ReplaceAll(p.Request.UserAgent(), "\"", "'")
		return fmt.Sprintf(
			"{\"time\":\"%s\",\"level\":\"INFO\",\"msg\":\"access\",\"method\":\"%s\",\"path\":\"%s\",\"status\":%d,\"latency\":\"%s\",\"bytes\":%d,\"request_id\":\"%s\",\"client_ip\":\"%s\",\"ua\":\"%s\"}\n",
			p.TimeStamp.Format(time.RFC3339Nano),
			p.Method,
			p.Path,
			p.StatusCode,
			p.Latency.String(),
			p.BodySize,
			p.Request.Header.Get("X-Request-ID"),
			p.ClientIP,
			ua,
		)
	})
}

// extractGraceToken pulls the phase-0 grace token from the request, accepting
// any of the three transports we know about:
//   - `X-Grace-Token` header (server-to-server callers, curl)
//   - `?grace_token=` query (HTML <video> tags can't add headers)
//   - `grace_session` cookie (browser session set by backend /auth/login)
func extractGraceToken(c *gin.Context) string {
	if t := c.GetHeader("X-Grace-Token"); t != "" {
		return t
	}
	if t := c.Query("grace_token"); t != "" {
		return t
	}
	if t, err := c.Cookie("grace_session"); err == nil && t != "" {
		return t
	}
	return ""
}

// requestID mirrors backend/internal/middleware.RequestID semantics.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)
		// Ensure downstream middlewares / logs see request id in headers too.
		c.Request.Header.Set("X-Request-ID", rid)
		c.Next()
	}
}

// assetIDPattern enforces the same shape upstream uses (alphanumeric +
// '-_'); rejects path traversal early.
var assetIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// upstreamLocator is just enough of asset.McapLocatorResponse for our needs.
// We deliberately don't import backend types — the two modules stay separate.
type upstreamLocator struct {
	AssetID string `json:"asset_id"`
	Mcap    struct {
		McapURI   string `json:"mcap_uri"`
		SizeBytes int64  `json:"size_bytes"`
	} `json:"mcap"`
	Window struct {
		StartTimestampNs int64 `json:"start_timestamp_ns"`
		EndTimestampNs   int64 `json:"end_timestamp_ns"`
		DurationMs       int64 `json:"duration_ms"`
	} `json:"window"`
}

func manifestHandler(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		assetID := c.Param("id")
		if !assetIDPattern.MatchString(assetID) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid asset id", nil)
			return
		}
		token := extractGraceToken(c)
		if token == "" {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "missing X-Grace-Token")
			return
		}
		if cfg.UpstreamBaseURL == "" {
			httpresp.Unavailable(c, httpresp.CodeServiceUnavailable, "upstream not configured")
			return
		}
		if cfg.OpenMCAP == nil {
			httpresp.Unavailable(c, httpresp.CodeServiceUnavailable, "mcap reader not configured")
			return
		}

		loc, status, err := fetchLocator(c, cfg, assetID, token)
		if err != nil {
			// fetchLocator already wrote the error envelope when it was
			// a propagated upstream status; only "transport error"
			// returns err with status 0.
			if status == 0 {
				httpresp.Error(c, http.StatusBadGateway, httpresp.CodeUpstreamError, err.Error(), nil)
			}
			return
		}

		rs, stats, closer, err := cfg.acquireMCAP(c.Request.Context(), loc.Mcap.McapURI)
		if err != nil {
			httpresp.Error(c, http.StatusBadGateway, httpresp.CodeUpstreamError, fmt.Sprintf("open mcap: %v", err), nil)
			return
		}
		if closer != nil {
			defer closer()
		}

		resp, err := manifest.Build(rs, manifest.Locator{
			AssetID:          loc.AssetID,
			McapURI:          loc.Mcap.McapURI,
			SizeBytes:        loc.Mcap.SizeBytes,
			StartTimestampNs: loc.Window.StartTimestampNs,
			EndTimestampNs:   loc.Window.EndTimestampNs,
			DurationMs:       loc.Window.DurationMs,
		}, stats)
		if err != nil {
			httpresp.Internal(c, fmt.Sprintf("build manifest: %v", err))
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// fetchLocator calls upstream GET /api/v1/assets/{id}/mcap-locator. On non-2xx
// it copies status+body through to the client so error semantics (404, 409,
// 503) match exactly. Returns (nil, status, err) with status>0 meaning the
// error was already written.
func fetchLocator(c *gin.Context, cfg Config, assetID, token string) (*upstreamLocator, int, error) {
	url := fmt.Sprintf("%s/api/v1/assets/%s/mcap-locator", cfg.UpstreamBaseURL, assetID)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	if cfg.GraceTokenPassthrough {
		req.Header.Set("X-Grace-Token", token)
	}
	if rid, _ := c.Get("request_id"); rid != nil {
		if s, ok := rid.(string); ok && s != "" {
			req.Header.Set("X-Request-ID", s)
		}
	}

	res, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("upstream call: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		// Pass-through: try to decode the upstream error envelope; if it
		// fails, synthesize one. Either way, this caller already wrote a
		// response, so we return status>0 to signal "do not write again".
		body, _ := io.ReadAll(res.Body)
		var envelope httpresp.ErrorBody
		if err := json.Unmarshal(body, &envelope); err == nil && envelope.Code != "" {
			httpresp.Error(c, res.StatusCode, envelope.Code, envelope.Message, envelope.Details)
		} else {
			httpresp.Error(c, res.StatusCode, httpresp.CodeUpstreamError,
				fmt.Sprintf("upstream returned %d", res.StatusCode), nil)
		}
		return nil, res.StatusCode, errors.New("upstream non-2xx")
	}

	var loc upstreamLocator
	if err := json.NewDecoder(res.Body).Decode(&loc); err != nil {
		return nil, 0, fmt.Errorf("decode upstream: %w", err)
	}
	return &loc, 0, nil
}
