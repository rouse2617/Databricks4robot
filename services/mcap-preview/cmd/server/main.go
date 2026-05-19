// mcap-preview server entry point.
//
// Reads its config from environment variables and wires a GCS-backed
// io.ReadSeeker for MCAP objects. Local file paths are accepted for dev /
// testing convenience but not expected in production deployments.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/gcsrs"
	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/server"
)

func main() {
	cfg := loadEnv()
	configureLogging(cfg.LogLevel, cfg.LogFormat)

	ctx := context.Background()

	openMCAP, gcsClose, err := buildOpenMCAP(ctx, cfg)
	if err != nil {
		slog.Error("init mcap opener", "err", err)
		os.Exit(1)
	}
	if gcsClose != nil {
		defer gcsClose()
	}

	r := server.New(server.Config{
		UpstreamBaseURL:       cfg.UpstreamBaseURL,
		GraceTokenPassthrough: cfg.GraceTokenPassthrough,
		HTTPClient:            &http.Client{Timeout: 15 * time.Second},
		OpenMCAP:              openMCAP,
		StrictWindowValidation: cfg.StrictWindowValidation,
	})

	addr := ":" + cfg.Port
	slog.Info("mcap-preview listening", "addr", addr, "upstream", cfg.UpstreamBaseURL)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

type appConfig struct {
	Port                  string
	UpstreamBaseURL       string
	GraceTokenPassthrough bool
	GCSPageSizeBytes      int64
	GCSPageCacheBytes     int64
	LogLevel              string
	LogFormat             string
	StrictWindowValidation bool
}

func loadEnv() appConfig {
	return appConfig{
		Port:                  envOr("PORT", "8090"),
		UpstreamBaseURL:       strings.TrimRight(envOr("UPSTREAM_BASE_URL", ""), "/"),
		GraceTokenPassthrough: envOr("GRACE_TOKEN_PASSTHROUGH", "true") != "false",
		GCSPageSizeBytes:      envInt64("GCS_PAGE_SIZE_BYTES", gcsrs.DefaultPageSize),
		GCSPageCacheBytes:     envInt64("GCS_PAGE_CACHE_BYTES", gcsrs.DefaultCacheBytes),
		LogLevel:              envOr("LOG_LEVEL", "info"),
		LogFormat:             envOr("LOG_FORMAT", "json"),
		StrictWindowValidation: envOr("STRICT_WINDOW_VALIDATION", "false") == "true",
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt64(k string, def int64) int64 {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func configureLogging(level, format string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if strings.ToLower(format) == "text" {
		h = slog.NewTextHandler(os.Stderr, opts)
	} else {
		h = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(h))
}

// buildOpenMCAP returns an opener that wraps a GCS object in the page-cached
// reader. Local paths (anything not gs://) fall back to os.File so devs can
// run the service against fixtures without needing GCS creds.
func buildOpenMCAP(ctx context.Context, cfg appConfig) (server.Config_OpenMCAP, func(), error) {
	cl, err := storage.NewClient(ctx)
	if err != nil {
		// Allow boot without GCS creds for local dev — only gs:// reads
		// will then fail, which surfaces as 502 to the caller.
		slog.Warn("gcs client init failed; gs:// reads will error", "err", err)
		return localOnlyOpener(), nil, nil
	}
	closer := func() { _ = cl.Close() }
	opts := &gcsrs.Options{
		PageSize:   cfg.GCSPageSizeBytes,
		CacheBytes: cfg.GCSPageCacheBytes,
	}
	return func(ctx context.Context, uri string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
		if bucket, object, ok := gcsrs.ParseGSURI(uri); ok {
			obj := cl.Bucket(bucket).Object(object)
			attrs, err := obj.Attrs(ctx)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("gcs attrs: %w", err)
			}
			rs := gcsrs.New(ctx, obj, attrs.Size, opts)
			stats := rs.Stats()
			return rs, &stats, nil, nil
		}
		return openLocal(uri)
	}, closer, nil
}

// server.Config_OpenMCAP is a type alias for readability in main.go.
// Keep the cmd code free of long function-type spellings.
//
// (This intentionally lives here, not in package server, so the public API
// stays a bare function field.)

func localOnlyOpener() server.Config_OpenMCAP {
	return func(_ context.Context, uri string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
		if strings.HasPrefix(uri, "gs://") {
			return nil, nil, nil, fmt.Errorf("gcs client not initialized")
		}
		return openLocal(uri)
	}
}

func openLocal(path string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	return f, nil, func() { _ = f.Close() }, nil
}
