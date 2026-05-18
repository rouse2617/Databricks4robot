// Package bigquery is the BigQuery-backed implementation of
// lakehouse.Querier. It queries BigLake-managed Iceberg tables registered as
// BigQuery external tables (see deploy/cloudrun/bronze-incremental/ for how
// the external-table pointer is refreshed at the end of every ingest run).
package bigquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	bq "cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"github.com/CyberOrigin2077/cyber-databrew/internal/lakehouse"
)

// Client is the BigQuery-backed Querier.
type Client struct {
	bq      *bq.Client
	project string
	dataset string
}

// Config gates construction. All fields are required.
type Config struct {
	Project string
	Dataset string
}

// New constructs a BigQuery-backed Querier. It performs a lightweight health
// probe (`SELECT 1`) so a failed call returns an error instead of producing a
// silently-broken client.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Project == "" || cfg.Dataset == "" {
		return nil, errors.New("bigquery: project and dataset are required")
	}
	c, err := bq.NewClient(ctx, cfg.Project)
	if err != nil {
		return nil, fmt.Errorf("bigquery: new client: %w", err)
	}
	cl := &Client{bq: c, project: cfg.Project, dataset: cfg.Dataset}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := cl.ping(pingCtx); err != nil {
		c.Close()
		return nil, err
	}
	return cl, nil
}

func (c *Client) ping(ctx context.Context) error {
	it, err := c.bq.Query("SELECT 1 AS one").Read(ctx)
	if err != nil {
		return fmt.Errorf("bigquery: ping read: %w", err)
	}
	var row []bq.Value
	if err := it.Next(&row); err != nil && !errors.Is(err, iterator.Done) {
		return fmt.Errorf("bigquery: ping next: %w", err)
	}
	return nil
}

// Status reports the live health of the BigQuery connection.
func (c *Client) Status(ctx context.Context) lakehouse.Status {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	st := lakehouse.Status{
		Enabled: true,
		Backend: "bigquery",
		Project: c.project,
		Dataset: c.dataset,
	}
	if err := c.ping(probeCtx); err != nil {
		st.Healthy = false
		st.Error = err.Error()
		return st
	}
	st.Healthy = true
	return st
}

// Query runs a read-only SQL statement and returns rows as column-keyed maps.
// Native BQ Go types are preserved (int64, float64, string, time.Time,
// civil.Date, etc.) — the handler is responsible for any further coercion
// before JSON-encoding.
func (c *Client) Query(ctx context.Context, sql string) ([]lakehouse.Row, error) {
	q := c.bq.Query(sql)
	q.DefaultProjectID = c.project
	q.DefaultDatasetID = c.dataset
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery: query read: %w", err)
	}
	var rows []lakehouse.Row
	for {
		bqRow := make(map[string]bq.Value)
		if err := it.Next(&bqRow); err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, fmt.Errorf("bigquery: query iterate: %w", err)
		}
		row := make(lakehouse.Row, len(bqRow))
		for k, v := range bqRow {
			row[k] = v
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// Close releases the underlying BigQuery client.
func (c *Client) Close() {
	if c != nil && c.bq != nil {
		_ = c.bq.Close()
	}
}
