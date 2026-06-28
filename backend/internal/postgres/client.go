package postgres

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

// Client owns a connection pool and dispatches all SQL through `db`. When
// inside a transaction (via WithTx), `db` is a tx-bound implementation and
// repos using `dbFromCtx(ctx, c.db)` automatically pick it up.
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
	Err() error
	Close()
}

// pgDB abstracts the few database operations repos need so that the same
// repo code path runs against a pool or a transaction interchangeably.
type pgDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) rowScanner
	Query(ctx context.Context, sql string, args ...any) (rowsScanner, error)
	Exec(ctx context.Context, sql string, args ...any) error
	// ExecResult executes a statement and returns the number of rows affected.
	ExecResult(ctx context.Context, sql string, args ...any) (int64, error)
	Ping(ctx context.Context) error
	Close()
}

// txKey is the context key under which a tx-bound pgDB is stashed by WithTx.
type txKey struct{}

// dbFromCtx returns the tx-bound pgDB if the context was created by WithTx,
// otherwise the fallback pool-bound pgDB. Repo methods that need to run
// inside a caller-provided transaction call this; methods that don't need
// tx awareness can keep using c.db directly.
func dbFromCtx(ctx context.Context, fallback pgDB) pgDB {
	if v := ctx.Value(txKey{}); v != nil {
		if tx, ok := v.(pgDB); ok {
			return tx
		}
	}
	return fallback
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

// realTx wraps pgx.Tx so that the same repo code path works inside a
// transaction. It is *not* directly closeable; the lifecycle is owned by
// WithTx via Commit / Rollback on the underlying tx.
type realTx struct {
	tx pgx.Tx
}

func (r *realTx) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return r.tx.QueryRow(ctx, sql, args...)
}

func (r *realTx) Query(ctx context.Context, sql string, args ...any) (rowsScanner, error) {
	return r.tx.Query(ctx, sql, args...)
}

func (r *realTx) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := r.tx.Exec(ctx, sql, args...)
	return err
}

func (r *realTx) ExecResult(ctx context.Context, sql string, args ...any) (int64, error) {
	ct, err := r.tx.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// Ping / Close are not meaningful on a transaction; provide no-ops so realTx
// still satisfies pgDB.
func (r *realTx) Ping(_ context.Context) error { return nil }
func (r *realTx) Close()                       {}

var newPool = func(ctx context.Context, dsn string) (pgDB, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// Explicit pool sizing — avoid relying on pgx defaults (MaxConns=4).
	poolCfg.MaxConns = envInt32("PG_MAX_CONNS", 20)
	poolCfg.MinConns = envInt32("PG_MIN_CONNS", 4)
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.MaxConnLifetime = 1 * time.Hour
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}
	return &realDB{pool: pool}, nil
}

func envInt32(key string, fallback int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			return int32(n)
		}
	}
	return fallback
}

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	// Use key=value format to avoid URL encoding issues with special chars in
	// the password (e.g. >, &).  pgx natively supports both formats.
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	return NewFromDSN(ctx, dsn)
}

// NewFromDSN creates a Client from a raw DSN string. Useful for integration
// tests where the DSN comes from a testcontainer rather than config.Config.
func NewFromDSN(ctx context.Context, dsn string) (*Client, error) {
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

// Ping verifies the database connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("postgres: client not initialized")
	}
	return c.db.Ping(ctx)
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

// Query executes a query that returns multiple rows.
// Exposed for packages outside postgres (e.g. lakehouse handler).
func (c *Client) Query(ctx context.Context, sql string, args ...any) (interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}, error) {
	return c.db.Query(ctx, sql, args...)
}

// WithTx runs fn inside a single PostgreSQL transaction. Repo methods that
// use dbFromCtx(ctx, ...) inside fn will transparently use the tx; methods
// that don't use it will run on the pool and are NOT part of the tx — when
// in doubt, every repo method called from within fn should be tx-aware.
//
// On non-pool-backed Clients (e.g. tests with an in-memory pgDB mock that
// does not support real transactions), WithTx degrades to calling fn with
// the plain ctx; the caller is then responsible for ensuring its mock
// honours the contract.
func (c *Client) WithTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	real, ok := c.db.(*realDB)
	if !ok {
		// Test/mocked client: no real transaction available; pass ctx as-is.
		return fn(ctx)
	}
	tx, err := real.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres WithTx begin: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		// Roll back on any non-commit exit (error return or panic). A fresh
		// background ctx is used so a cancelled request ctx — the very reason
		// fn often fails — cannot prevent the rollback from being sent.
		rbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(rbCtx)
	}()
	txCtx := context.WithValue(ctx, txKey{}, &realTx{tx: tx})
	if err = fn(txCtx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres WithTx commit: %w", err)
	}
	committed = true
	return nil
}
