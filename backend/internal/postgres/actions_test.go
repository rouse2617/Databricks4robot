package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsActionIDSchemaMismatch(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if isActionIDSchemaMismatch(nil) {
			t.Fatal("expected false for nil pgErr")
		}
	})

	t.Run("uuid parse error", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "22P02",
			Message: `invalid input syntax for type uuid: "Z4PZDKBQ"`,
		}
		if !isActionIDSchemaMismatch(pgErr) {
			t.Fatal("expected true for 22P02 type uuid parse error")
		}
	})

	t.Run("other parse error", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "22P02",
			Message: `invalid input syntax for type integer: "abc"`,
		}
		if isActionIDSchemaMismatch(pgErr) {
			t.Fatal("expected false for non-uuid parse error")
		}
	})

	t.Run("different code", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "23505",
			Message: "duplicate key value violates unique constraint",
		}
		if isActionIDSchemaMismatch(pgErr) {
			t.Fatal("expected false for non-22P02 error code")
		}
	})
}
