package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"data-platform/internal/config"
)

type Client struct {
	db pgDB
}

var errNoRows = pgx.ErrNoRows

type rowScanner interface {
	Scan(dest ...any) error
}

type rowsScanner interface {
	Next() bool
	Scan(dest ...any) error
	Close()
}

type pgDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) rowScanner
	Query(ctx context.Context, sql string, args ...any) (rowsScanner, error)
	Exec(ctx context.Context, sql string, args ...any) error
	// ExecResult executes a statement and returns the number of rows affected.
	ExecResult(ctx context.Context, sql string, args ...any) (int64, error)
	Ping(ctx context.Context) error
	Close()
}

type realDB struct {
	pool *pgxpool.Pool
}

func (r *realDB) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return r.pool.QueryRow(ctx, sql, args...)
}

func (r *realDB) Query(ctx context.Context, sql string, args ...any) (rowsScanner, error) {
	return r.pool.Query(ctx, sql, args...)
}

func (r *realDB) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := r.pool.Exec(ctx, sql, args...)
	return err
}

func (r *realDB) ExecResult(ctx context.Context, sql string, args ...any) (int64, error) {
	ct, err := r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *realDB) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }
func (r *realDB) Close()                         { r.pool.Close() }

var newPool = func(ctx context.Context, dsn string) (pgDB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &realDB{pool: pool}, nil
}

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)
	db, err := newPool(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &Client{db: db}, nil
}

func (c *Client) Close() {
	if c != nil && c.db != nil {
		c.db.Close()
	}
}

// Exec executes a SQL statement. Exposed for packages outside postgres (e.g. audit).
func (c *Client) Exec(ctx context.Context, sql string, args ...any) error {
	return c.db.Exec(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row.
// Exposed for packages outside postgres (e.g. lakehouse handler).
func (c *Client) QueryRow(ctx context.Context, sql string, args ...any) interface{ Scan(dest ...any) error } {
	return c.db.QueryRow(ctx, sql, args...)
}
