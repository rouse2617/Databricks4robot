package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type DeliveryRuleRepo struct {
	c *Client
}

var _ repository.DeliveryRuleRepository = (*DeliveryRuleRepo)(nil)

func NewDeliveryRuleRepo(c *Client) *DeliveryRuleRepo {
	return &DeliveryRuleRepo{c: c}
}

func (r *DeliveryRuleRepo) Insert(ctx context.Context, rule *models.DeliveryRule) error {
	dslBytes, err := normalizeQueryDSL(rule.QueryDSL)
	if err != nil {
		return err
	}
	var customerID any
	if rule.CustomerID != "" {
		customerID = rule.CustomerID
	}
	if rule.DSLVersion == "" {
		rule.DSLVersion = "v1"
	}
	if rule.EnforceMode == "" {
		rule.EnforceMode = "block"
	}
	if rule.RatingScope == "" {
		rule.RatingScope = "current"
	}

	const q = `
INSERT INTO delivery_rules (
  rule_id, name, owner, customer_id, query_dsl, dsl_version,
  enforce_mode, rating_scope, is_active, version, created_at, updated_at
) VALUES (
  COALESCE(NULLIF($1::text, '')::uuid, gen_random_uuid()),
  $2, $3, $4, $5::jsonb, $6, $7, $8, $9, 1, now(), now()
)
RETURNING rule_id::text, created_at, updated_at`
	var ruleIDArg any
	if rule.RuleID != "" {
		ruleIDArg = rule.RuleID
	}
	err = r.c.db.QueryRow(ctx, q,
		ruleIDArg, rule.Name, rule.Owner, customerID, dslBytes,
		rule.DSLVersion, rule.EnforceMode, rule.RatingScope, rule.IsActive,
	).Scan(&rule.RuleID, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return wrapDeliveryRuleSchema(err, "Insert")
	}
	return nil
}

func (r *DeliveryRuleRepo) ListActiveForCustomer(ctx context.Context, customerID string) ([]*models.DeliveryRule, error) {
	const q = `
SELECT rule_id::text, name, owner, COALESCE(customer_id, ''), query_dsl, dsl_version,
       enforce_mode, rating_scope, is_active, version, created_at, updated_at
FROM delivery_rules
WHERE is_active = true
  AND (customer_id IS NULL OR customer_id = $1)
ORDER BY customer_id NULLS FIRST, created_at ASC`
	return r.scanRules(ctx, q, customerID)
}

func (r *DeliveryRuleRepo) List(ctx context.Context, customerID string) ([]*models.DeliveryRule, error) {
	if customerID == "" {
		const q = `
SELECT rule_id::text, name, owner, COALESCE(customer_id, ''), query_dsl, dsl_version,
       enforce_mode, rating_scope, is_active, version, created_at, updated_at
FROM delivery_rules
ORDER BY created_at DESC`
		return r.scanRules(ctx, q)
	}
	const q = `
SELECT rule_id::text, name, owner, COALESCE(customer_id, ''), query_dsl, dsl_version,
       enforce_mode, rating_scope, is_active, version, created_at, updated_at
FROM delivery_rules
WHERE customer_id IS NULL OR customer_id = $1
ORDER BY created_at DESC`
	return r.scanRules(ctx, q, customerID)
}

func (r *DeliveryRuleRepo) scanRules(ctx context.Context, q string, args ...any) ([]*models.DeliveryRule, error) {
	rows, err := r.c.db.Query(ctx, q, args...)
	if err != nil {
		return nil, wrapDeliveryRuleSchema(err, "scanRules")
	}
	defer rows.Close()
	var out []*models.DeliveryRule
	for rows.Next() {
		var rule models.DeliveryRule
		var dsl []byte
		if err := rows.Scan(
			&rule.RuleID, &rule.Name, &rule.Owner, &rule.CustomerID, &dsl,
			&rule.DSLVersion, &rule.EnforceMode, &rule.RatingScope, &rule.IsActive,
			&rule.Version, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres DeliveryRuleRepo.scan: %w", err)
		}
		rule.QueryDSL = dsl
		out = append(out, &rule)
	}
	return out, nil
}

func normalizeQueryDSL(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("query_dsl is required")
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("query_dsl must be valid JSON")
	}
	return raw, nil
}

func wrapDeliveryRuleSchema(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return fmt.Errorf("%w: delivery_rules (%s)", repository.ErrSchemaMismatch, op)
	}
	return fmt.Errorf("postgres DeliveryRuleRepo.%s: %w", op, err)
}
