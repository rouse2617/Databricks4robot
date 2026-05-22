package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type CustomerRepo struct {
	c *Client
}

var _ repository.CustomerRepository = (*CustomerRepo)(nil)

func NewCustomerRepo(c *Client) *CustomerRepo {
	return &CustomerRepo{c: c}
}

func (r *CustomerRepo) Insert(ctx context.Context, c *models.Customer) error {
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = "active"
	}
	if c.SLATier == "" {
		c.SLATier = "standard"
	}
	complianceJSON, excludeJSON, metadataJSON, extraJSON := customerJSONFields(c)

	const q = `
INSERT INTO customers(
  customer_id, display_name, legal_name, status, region, sla_tier, account_owner,
  compliance_tags, exclude_tags, metadata, extra,
  onboarded_at, offboarded_at, created_at, updated_at, row_version
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,
  $8::jsonb,$9::jsonb,$10::jsonb,$11::jsonb,
  $12,$13,$14,$15,$16
)`
	err := r.c.db.Exec(ctx, q,
		c.CustomerID, c.DisplayName, nullIfEmpty(c.LegalName), c.Status, nullIfEmpty(c.Region), c.SLATier, nullIfEmpty(c.AccountOwner),
		complianceJSON, excludeJSON, metadataJSON, extraJSON,
		c.OnboardedAt, c.OffboardedAt, c.CreatedAt, c.UpdatedAt, c.RowVersion,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repository.ErrDuplicateCustomerID
		}
		return fmt.Errorf("postgres CustomerRepo.Insert: %w", err)
	}
	return nil
}

func (r *CustomerRepo) Get(ctx context.Context, customerID string) (*models.Customer, error) {
	const q = `
SELECT customer_id, display_name, COALESCE(legal_name, ''), status, COALESCE(region, ''),
  sla_tier, COALESCE(account_owner, ''),
  compliance_tags, exclude_tags, metadata, extra,
  onboarded_at, offboarded_at, created_at, updated_at, row_version
FROM customers WHERE customer_id = $1`
	var (
		c              models.Customer
		complianceJSON []byte
		excludeJSON    []byte
		metadataJSON   []byte
		extraJSON      []byte
	)
	err := r.c.db.QueryRow(ctx, q, customerID).Scan(
		&c.CustomerID, &c.DisplayName, &c.LegalName, &c.Status, &c.Region,
		&c.SLATier, &c.AccountOwner,
		&complianceJSON, &excludeJSON, &metadataJSON, &extraJSON,
		&c.OnboardedAt, &c.OffboardedAt, &c.CreatedAt, &c.UpdatedAt, &c.RowVersion,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres CustomerRepo.Get: %w", err)
	}
	decodeCustomerJSON(&c, complianceJSON, excludeJSON, metadataJSON, extraJSON)
	return &c, nil
}

func (r *CustomerRepo) Update(ctx context.Context, c *models.Customer) error {
	c.UpdatedAt = time.Now().UTC()
	c.RowVersion++
	complianceJSON, excludeJSON, metadataJSON, extraJSON := customerJSONFields(c)

	const q = `
UPDATE customers SET
  display_name = $2,
  legal_name = $3,
  status = $4,
  region = $5,
  sla_tier = $6,
  account_owner = $7,
  compliance_tags = $8::jsonb,
  exclude_tags = $9::jsonb,
  metadata = $10::jsonb,
  extra = $11::jsonb,
  onboarded_at = $12,
  offboarded_at = $13,
  updated_at = $14,
  row_version = $15
WHERE customer_id = $1 AND row_version = $16 - 1`
	rows, err := r.c.db.ExecResult(ctx, q,
		c.CustomerID, c.DisplayName, nullIfEmpty(c.LegalName), c.Status, nullIfEmpty(c.Region), c.SLATier, nullIfEmpty(c.AccountOwner),
		complianceJSON, excludeJSON, metadataJSON, extraJSON,
		c.OnboardedAt, c.OffboardedAt, c.UpdatedAt, c.RowVersion,
		c.RowVersion,
	)
	if err != nil {
		return fmt.Errorf("postgres CustomerRepo.Update: %w", err)
	}
	if rows == 0 {
		return repository.ErrOptimisticLock
	}
	return nil
}

func (r *CustomerRepo) Exists(ctx context.Context, customerID string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM customers WHERE customer_id = $1)`
	var ok bool
	if err := r.c.db.QueryRow(ctx, q, customerID).Scan(&ok); err != nil {
		return false, fmt.Errorf("postgres CustomerRepo.Exists: %w", err)
	}
	return ok, nil
}

func customerJSONFields(c *models.Customer) (compliance, exclude, metadata, extra []byte) {
	if c.ComplianceTags == nil {
		compliance = []byte("[]")
	} else {
		compliance, _ = json.Marshal(c.ComplianceTags)
	}
	if c.ExcludeTags == nil {
		exclude = []byte("[]")
	} else {
		exclude, _ = json.Marshal(c.ExcludeTags)
	}
	if c.Metadata == nil {
		metadata = []byte("{}")
	} else {
		metadata, _ = json.Marshal(c.Metadata)
	}
	if c.Extra == nil {
		extra = []byte("{}")
	} else {
		extra, _ = json.Marshal(c.Extra)
	}
	return compliance, exclude, metadata, extra
}

func decodeCustomerJSON(c *models.Customer, complianceJSON, excludeJSON, metadataJSON, extraJSON []byte) {
	if len(complianceJSON) > 0 {
		_ = json.Unmarshal(complianceJSON, &c.ComplianceTags)
	}
	if len(excludeJSON) > 0 {
		_ = json.Unmarshal(excludeJSON, &c.ExcludeTags)
	}
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &c.Metadata)
	}
	if len(extraJSON) > 0 {
		_ = json.Unmarshal(extraJSON, &c.Extra)
	}
}
