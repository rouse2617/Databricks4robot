package trino

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	_ "github.com/trinodb/trino-go-client/trino"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
)

type Client struct {
	db      *sql.DB
	catalog string
	schema  string
}

type TableCount struct {
	TableName string `json:"table_name"`
	RowCount  int64  `json:"row_count"`
}

type Status struct {
	Enabled bool   `json:"enabled"`
	Healthy bool   `json:"healthy"`
	Catalog string `json:"catalog"`
	Schema  string `json:"schema"`
	Error   string `json:"error,omitempty"`
}

var identifierRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg.TrinoEnabled != "true" {
		return nil, nil
	}
	if !validIdentifier(cfg.TrinoCatalog) || !validIdentifier(cfg.TrinoSchema) {
		return nil, fmt.Errorf("invalid trino catalog or schema")
	}

	db, err := sql.Open("trino", cfg.TrinoURL)
	if err != nil {
		return nil, fmt.Errorf("trino open: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)

	c := &Client{db: db, catalog: cfg.TrinoCatalog, schema: cfg.TrinoSchema}
	if err := c.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() {
	if c != nil && c.db != nil {
		c.db.Close()
	}
}

func (c *Client) Ping(ctx context.Context) error {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.TrinoRequestsTotal.WithLabelValues("ping", outcome).Inc()
		metrics.TrinoRequestDurationSeconds.WithLabelValues("ping", outcome).Observe(time.Since(start).Seconds())
	}()

	if c == nil || c.db == nil {
		outcome = "error"
		return fmt.Errorf("trino disabled")
	}
	if err := c.db.PingContext(ctx); err != nil {
		outcome = "error"
		return fmt.Errorf("trino ping: %w", err)
	}
	return nil
}

func (c *Client) Status(ctx context.Context) Status {
	status := Status{Enabled: c != nil, Healthy: false}
	if c == nil {
		return status
	}
	status.Catalog = c.catalog
	status.Schema = c.schema
	if err := c.Ping(ctx); err != nil {
		status.Error = err.Error()
		return status
	}
	status.Healthy = true
	return status
}

// KnownTables lists all Iceberg tables the platform manages.
// bronze_asset_events is the CDC-driven append-only event lake (§5.6.2).
var KnownTables = []string{
	"bronze_asset_events",
	"bronze_asset_algo_events",
	"bronze_delivery_items",
	"silver_mcap_files_current",
	"silver_assets_current",
	"silver_deliveries_current",
	"silver_asset_algo_latest",
	"silver_asset_tags",
	"gold_dataset_snapshot_items",
}

func (c *Client) Tables(ctx context.Context) ([]TableCount, error) {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.TrinoRequestsTotal.WithLabelValues("tables", outcome).Inc()
		metrics.TrinoRequestDurationSeconds.WithLabelValues("tables", outcome).Observe(time.Since(start).Seconds())
	}()

	parts := make([]string, 0, len(KnownTables))
	for _, tableName := range KnownTables {
		parts = append(parts, fmt.Sprintf(
			"SELECT '%s' AS table_name, count(*) AS row_count FROM %s",
			tableName,
			c.table(tableName),
		))
	}

	rows, err := c.db.QueryContext(ctx, strings.Join(parts, " UNION ALL "))
	if err != nil {
		if !strings.Contains(err.Error(), "does not exist") {
			outcome = "error"
			slog.Warn("trino tables query failed, returning empty list", "err", err)
			return []TableCount{}, nil
		}
		items, bestErr := c.tablesBestEffort(ctx)
		if bestErr != nil {
			outcome = "error"
			slog.Warn("trino tables best-effort query failed, returning empty list", "err", bestErr)
			return []TableCount{}, nil
		}
		return items, nil
	}
	defer rows.Close()

	var result []TableCount
	for rows.Next() {
		var item TableCount
		if err := rows.Scan(&item.TableName, &item.RowCount); err != nil {
			outcome = "error"
			slog.Warn("trino tables scan failed, returning partial results", "err", err)
			return result, nil
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		outcome = "error"
		slog.Warn("trino tables rows iteration failed, returning partial results", "err", err)
		return result, nil
	}
	return result, nil
}

func (c *Client) tablesBestEffort(ctx context.Context) ([]TableCount, error) {
	result := make([]TableCount, 0, len(KnownTables))
	for _, tableName := range KnownTables {
		var count int64
		err := c.db.QueryRowContext(
			ctx,
			fmt.Sprintf("SELECT count(*) FROM %s", c.table(tableName)),
		).Scan(&count)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return nil, err
		}
		result = append(result, TableCount{TableName: tableName, RowCount: count})
	}
	return result, nil
}

// BronzeEventCount returns the total row count in bronze_asset_events.
// Used by the sync-status endpoint to compare the Bronze-covered event window
// against PostgreSQL.
func (c *Client) BronzeEventCount(ctx context.Context) (int64, error) {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.TrinoRequestsTotal.WithLabelValues("bronze_event_count", outcome).Inc()
		metrics.TrinoRequestDurationSeconds.WithLabelValues("bronze_event_count", outcome).Observe(time.Since(start).Seconds())
	}()

	var count int64
	err := c.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT count(*) FROM %s", c.table("bronze_asset_events")),
	).Scan(&count)
	if err != nil {
		outcome = "error"
	}
	return count, err
}

// BronzeMaxEventSeq returns the maximum event_seq in bronze_asset_events.
func (c *Client) BronzeMaxEventSeq(ctx context.Context) (int64, error) {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.TrinoRequestsTotal.WithLabelValues("bronze_max_event_seq", outcome).Inc()
		metrics.TrinoRequestDurationSeconds.WithLabelValues("bronze_max_event_seq", outcome).Observe(time.Since(start).Seconds())
	}()

	var seq int64
	err := c.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COALESCE(max(event_seq), 0) FROM %s", c.table("bronze_asset_events")),
	).Scan(&seq)
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		outcome = "error"
		return 0, errors.New("bronze_asset_events table missing event_seq; no realtime cursor fallback available")
	}
	if err != nil {
		outcome = "error"
	}
	return seq, err
}

// BronzeEventSeqBounds returns the [min, max] event_seq currently materialized
// in bronze_asset_events. When the table is empty, both bounds are 0.
func (c *Client) BronzeEventSeqBounds(ctx context.Context) (int64, int64, error) {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.TrinoRequestsTotal.WithLabelValues("bronze_seq_bounds", outcome).Inc()
		metrics.TrinoRequestDurationSeconds.WithLabelValues("bronze_seq_bounds", outcome).Observe(time.Since(start).Seconds())
	}()

	var minSeq, maxSeq int64
	err := c.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COALESCE(min(event_seq), 0), COALESCE(max(event_seq), 0) FROM %s", c.table("bronze_asset_events")),
	).Scan(&minSeq, &maxSeq)
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		outcome = "error"
		return 0, 0, errors.New("bronze_asset_events table missing event_seq bounds")
	}
	if err != nil {
		outcome = "error"
	}
	return minSeq, maxSeq, err
}

func (c *Client) QueryRows(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	start := time.Now()
	outcome := "ok"
	defer func() {
		metrics.TrinoRequestsTotal.WithLabelValues("query_rows", outcome).Inc()
		metrics.TrinoRequestDurationSeconds.WithLabelValues("query_rows", outcome).Observe(time.Since(start).Seconds())
	}()

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		outcome = "error"
		return nil, err
	}
	defer rows.Close()
	items, err := rowsToMaps(rows)
	if err != nil {
		outcome = "error"
		return nil, err
	}
	return items, nil
}

func (c *Client) Table(name string) string {
	return c.table(name)
}

func (c *Client) table(name string) string {
	if !validIdentifier(name) {
		panic("invalid hardcoded trino table name")
	}
	return fmt.Sprintf("%s.%s.%s", c.catalog, c.schema, name)
}

func validIdentifier(value string) bool {
	return identifierRE.MatchString(value)
}

func rowsToMaps(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = normalizeValue(values[i])
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		return v
	}
}
