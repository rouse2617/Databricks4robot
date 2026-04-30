// Standalone outbox worker binary.
//
// Reuses the in-process ESWorker from internal/outbox but runs as an
// independent process with its own health endpoint. Designed for future
// independent deployment once P0-4 has been stable for 90 days.
//
// Environment variables are identical to the main server's outbox config
// plus the standard PG and ES connection vars. See backend/.env.example.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	espkg "data-platform/internal/elasticsearch"
	"data-platform/internal/outbox"
	"data-platform/internal/postgres"
	"data-platform/internal/searchindex"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file, using environment variables")
	}

	// ── Configuration ───────────────────────────────────────────────
	port := getenv("OUTBOX_WORKER_PORT", "8081")
	tickSec, _ := strconv.Atoi(getenv("OUTBOX_TICK_INTERVAL_SEC", "30"))
	if tickSec <= 0 {
		tickSec = 30
	}
	batchSize, _ := strconv.Atoi(getenv("OUTBOX_BATCH_SIZE", "1000"))
	if batchSize <= 0 {
		batchSize = 1000
	}
	retryLimit, _ := strconv.Atoi(getenv("OUTBOX_RETRY_LIMIT", "10"))
	if retryLimit <= 0 {
		retryLimit = 10
	}
	fatalOnPanic := getenv("OUTBOX_FATAL_ON_PANIC", "true") == "true"

	// ── PostgreSQL ──────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgCfg := &pgConfig{
		Host:     getenv("DB_HOST", "localhost"),
		Port:     getenv("DB_PORT", "5432"),
		User:     getenv("DB_USER", "postgres"),
		Password: getenv("DB_PASSWORD", "postgres"),
		DBName:   getenv("DB_NAME", "data4cyber"),
	}
	pgClient, err := postgres.NewFromDSN(ctx, pgCfg.DSN())
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pgClient.Close()

	// ── Elasticsearch ───────────────────────────────────────────────
	esURL := getenv("ELASTICSEARCH_URL", "http://localhost:9200")
	esIndex := getenv("ELASTICSEARCH_INDEX", "assets")
	esClient := espkg.New(esURL, esIndex)
	if err := esClient.Ping(ctx); err != nil {
		slog.Error("elasticsearch unavailable", "err", err)
		os.Exit(1)
	}
	slog.Info("elasticsearch connected", "url", esURL, "index", esIndex)

	// ── Worker ──────────────────────────────────────────────────────
	health := outbox.NewHealthStatus()
	worker := &outbox.ESWorker{
		Events: postgres.NewAssetEventRepo(pgClient),
		Indexer: &searchindex.Builder{
			Assets: postgres.NewAssetRepo(pgClient),
			Tags:   postgres.NewAssetTagRepo(pgClient),
			Algos:  postgres.NewAssetAlgoLatestRepo(pgClient),
			Mcap:   postgres.NewMcapFileRepo(pgClient),
		},
		ES:           esClient,
		BatchSize:    batchSize,
		SinkName:     "es_assets",
		FatalOnPanic: fatalOnPanic,
		RetryLimit:   retryLimit,
		Health:       health,
	}

	// ── Health endpoint ─────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if health.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "ok")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "unhealthy")
		}
	})
	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		slog.Info("health endpoint listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server error", "err", err)
		}
	}()

	// ── Start worker ────────────────────────────────────────────────
	slog.Info("outbox worker starting",
		"tick_sec", tickSec,
		"batch_size", batchSize,
		"retry_limit", retryLimit,
		"fatal_on_panic", fatalOnPanic,
	)
	go worker.Run(ctx, time.Duration(tickSec)*time.Second)

	// ── Graceful shutdown ───────────────────────────────────────────
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	slog.Info("shutting down outbox worker")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

// pgConfig holds PostgreSQL connection parameters.
type pgConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// DSN returns a PostgreSQL connection string.
func (c *pgConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User, c.Password, c.Host, c.Port, c.DBName)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
