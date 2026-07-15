package repository

import (
	"context"
	"errors"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ErrDuplicateClusterName is returned when creating a cluster whose name
// already exists (case-sensitive, ignoring soft-deleted rows).
var ErrDuplicateClusterName = errors.New("duplicate cluster name")

// ErrClusterInUse is returned when trying to delete a cluster that has one or
// more ExecutionTargets referencing it.
var ErrClusterInUse = errors.New("cluster is still referenced by execution_targets")

// ClusterRepository persists K8s cluster connection metadata (CYB-3425).
// Consumers include the admin CRUD API and the client factory that decides
// which K8s / Argo endpoint to talk to for a given ExecutionTarget.
type ClusterRepository interface {
	// List returns all clusters ordered by name. Soft-deleted clusters are
	// excluded unless includeDeleted is true.
	List(ctx context.Context, includeDeleted bool) ([]*models.Cluster, error)
	// Get returns one cluster by id, or (nil, nil) when not found or soft-deleted.
	Get(ctx context.Context, id string) (*models.Cluster, error)
	// GetByName returns one cluster by name, or (nil, nil) when not found. Ignores soft-deleted.
	GetByName(ctx context.Context, name string) (*models.Cluster, error)
	// Create inserts a new cluster; returns ErrDuplicateClusterName on conflict.
	Create(ctx context.Context, c *models.Cluster) (*models.Cluster, error)
	// Update mutates an existing cluster; returns (nil, nil) when the id is absent.
	Update(ctx context.Context, c *models.Cluster) (*models.Cluster, error)
	// SoftDelete flags the cluster deleted_at=now(); returns ErrClusterInUse if
	// any ExecutionTarget still references it.
	SoftDelete(ctx context.Context, id string) error
	// CountReferencingTargets counts non-soft-deleted execution_targets whose
	// cluster_id matches. Used by SoftDelete to enforce the safety check and
	// by the admin API to preview whether delete is safe.
	CountReferencingTargets(ctx context.Context, clusterID string) (int, error)
}
