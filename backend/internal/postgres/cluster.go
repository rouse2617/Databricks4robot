package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ClusterRepo persists K8s cluster connection metadata (CYB-3425).
type ClusterRepo struct {
	c *Client
}

// NewClusterRepo constructs a ClusterRepo backed by the shared pg client.
func NewClusterRepo(c *Client) *ClusterRepo { return &ClusterRepo{c: c} }

var _ repository.ClusterRepository = (*ClusterRepo)(nil)

const clusterCols = `id, name, display_name, description, is_default, status,
	k8s_api_endpoint, k8s_audience, k8s_ca_data,
	auth_type, auth_secret_ref,
	argo_server_url, argo_namespace,
	koord_installed,
	created_at, updated_at, deleted_at`

func scanCluster(s rowScanner) (*models.Cluster, error) {
	var c models.Cluster
	if err := s.Scan(
		&c.ID, &c.Name, &c.DisplayName, &c.Description, &c.IsDefault, &c.Status,
		&c.K8sAPIEndpoint, &c.K8sAudience, &c.K8sCAData,
		&c.AuthType, &c.AuthSecretRef,
		&c.ArgoServerURL, &c.ArgoNamespace,
		&c.KoordInstalled,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ClusterRepo) List(ctx context.Context, includeDeleted bool) ([]*models.Cluster, error) {
	q := `SELECT ` + clusterCols + ` FROM clusters`
	if !includeDeleted {
		q += ` WHERE deleted_at IS NULL`
	}
	q += ` ORDER BY name`
	rows, err := r.c.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres ClusterRepo.List: %w", err)
	}
	defer rows.Close()
	var out []*models.Cluster
	for rows.Next() {
		c, err := scanCluster(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres ClusterRepo.List scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ClusterRepo) Get(ctx context.Context, id string) (*models.Cluster, error) {
	row := r.c.db.QueryRow(ctx, `SELECT `+clusterCols+` FROM clusters WHERE id = $1 AND deleted_at IS NULL`, id)
	c, err := scanCluster(row)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres ClusterRepo.Get: %w", err)
	}
	return c, nil
}

func (r *ClusterRepo) GetByName(ctx context.Context, name string) (*models.Cluster, error) {
	row := r.c.db.QueryRow(ctx, `SELECT `+clusterCols+` FROM clusters WHERE name = $1 AND deleted_at IS NULL`, name)
	c, err := scanCluster(row)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres ClusterRepo.GetByName: %w", err)
	}
	return c, nil
}

func (r *ClusterRepo) Create(ctx context.Context, c *models.Cluster) (*models.Cluster, error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	authType := c.AuthType
	if authType == "" {
		authType = "gke_wif"
	}
	const q = `INSERT INTO clusters (
		id, name, display_name, description, is_default, status,
		k8s_api_endpoint, k8s_audience, k8s_ca_data,
		auth_type, auth_secret_ref,
		argo_server_url, argo_namespace, koord_installed
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	RETURNING ` + clusterCols
	row := r.c.db.QueryRow(ctx, q,
		c.ID, c.Name, c.DisplayName, c.Description, c.IsDefault, c.Status,
		c.K8sAPIEndpoint, c.K8sAudience, c.K8sCAData,
		authType, c.AuthSecretRef,
		c.ArgoServerURL, c.ArgoNamespace, c.KoordInstalled,
	)
	out, err := scanCluster(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, repository.ErrDuplicateClusterName
		}
		return nil, fmt.Errorf("postgres ClusterRepo.Create: %w", err)
	}
	return out, nil
}

func (r *ClusterRepo) Update(ctx context.Context, c *models.Cluster) (*models.Cluster, error) {
	authType := c.AuthType
	if authType == "" {
		authType = "gke_wif"
	}
	const q = `UPDATE clusters SET
		name = $2, display_name = $3, description = $4, is_default = $5, status = $6,
		k8s_api_endpoint = $7, k8s_audience = $8, k8s_ca_data = $9,
		auth_type = $10, auth_secret_ref = $11,
		argo_server_url = $12, argo_namespace = $13, koord_installed = $14,
		updated_at = now()
	WHERE id = $1 AND deleted_at IS NULL
	RETURNING ` + clusterCols
	row := r.c.db.QueryRow(ctx, q,
		c.ID, c.Name, c.DisplayName, c.Description, c.IsDefault, c.Status,
		c.K8sAPIEndpoint, c.K8sAudience, c.K8sCAData,
		authType, c.AuthSecretRef,
		c.ArgoServerURL, c.ArgoNamespace, c.KoordInstalled,
	)
	out, err := scanCluster(row)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, repository.ErrDuplicateClusterName
		}
		return nil, fmt.Errorf("postgres ClusterRepo.Update: %w", err)
	}
	return out, nil
}

func (r *ClusterRepo) SoftDelete(ctx context.Context, id string) error {
	// FK ON DELETE RESTRICT protects us at the DB layer, but we surface a
	// nicer error via a pre-check so the admin API can return 409 with a
	// count instead of a raw pg error.
	n, err := r.CountReferencingTargets(ctx, id)
	if err != nil {
		return fmt.Errorf("postgres ClusterRepo.SoftDelete pre-check: %w", err)
	}
	if n > 0 {
		return repository.ErrClusterInUse
	}
	if err := r.c.db.Exec(ctx,
		`UPDATE clusters SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	); err != nil {
		return fmt.Errorf("postgres ClusterRepo.SoftDelete: %w", err)
	}
	return nil
}

func (r *ClusterRepo) CountReferencingTargets(ctx context.Context, clusterID string) (int, error) {
	var n int
	if err := r.c.db.QueryRow(ctx,
		`SELECT count(*) FROM execution_targets WHERE cluster_id = $1`, clusterID,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres ClusterRepo.CountReferencingTargets: %w", err)
	}
	return n, nil
}
