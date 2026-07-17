package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/audit"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ClusterCacheInvalidator is the seam CYB-3486 uses to tell per-cluster caches
// (k8s.ClientFactory, argo.ClientFactory) that a row changed. The interface
// lives here so the admin package doesn't need to import the runtime
// factory packages — avoids an import cycle and keeps 4a purely additive.
type ClusterCacheInvalidator interface {
	Invalidate(clusterID string)
}

// ClusterHandler serves admin CRUD for K8s clusters (CYB-3425). Reads are
// exposed to any authenticated user (targets need to render the cluster name)
// but writes are gated by adminAuth in routes.go.
type ClusterHandler struct {
	repo         repository.ClusterRepository
	invalidators []ClusterCacheInvalidator
}

// NewClusterHandler wires the cluster repo plus zero or more caches that want
// to be told when a cluster row is created / updated / soft-deleted.
func NewClusterHandler(repo repository.ClusterRepository, invalidators ...ClusterCacheInvalidator) *ClusterHandler {
	return &ClusterHandler{repo: repo, invalidators: invalidators}
}

func (h *ClusterHandler) invalidate(id string) {
	for _, inv := range h.invalidators {
		inv.Invalidate(id)
	}
}

type clusterRequest struct {
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	IsDefault      bool   `json:"isDefault"`
	Status         string `json:"status"`
	K8sAPIEndpoint string `json:"k8sApiEndpoint"`
	K8sAudience    string `json:"k8sAudience"`
	K8sCAData      string `json:"k8sCaData"`
	ArgoServerURL  string `json:"argoServerUrl"`
	ArgoNamespace  string `json:"argoNamespace"`
	KoordInstalled bool   `json:"koordInstalled"`
	// ClientQPS / ClientBurst tune the backend's K8s client rate limit to this
	// cluster (rest.Config QPS/Burst). Zero → repo default (50/100). CYB-3486.
	ClientQPS   float32 `json:"clientQps"`
	ClientBurst int     `json:"clientBurst"`
}

func normalizeClusterReq(req *clusterRequest) string {
	req.Name = strings.TrimSpace(req.Name)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		req.Status = "available"
	}
	if req.Name == "" {
		return "name is required"
	}
	// Name is used in log tags / URLs — lock to a safe alphabet.
	for _, r := range req.Name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return "name must contain only [a-zA-Z0-9-_]"
		}
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Name
	}
	return ""
}

// List returns all active clusters. GET /api/v1/clusters (any authed user).
func (h *ClusterHandler) List(c *gin.Context) {
	items, err := h.repo.List(c.Request.Context(), false)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	// Materialize an empty array (not null) so the FE can rely on .length.
	if items == nil {
		items = []*models.Cluster{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Get returns one cluster by id. GET /api/v1/clusters/:id
func (h *ClusterHandler) Get(c *gin.Context) {
	id := c.Param("id")
	cluster, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if cluster == nil {
		httpresp.NotFound(c, "cluster_not_found", "cluster not found")
		return
	}
	c.JSON(http.StatusOK, cluster)
}

// Create inserts a new cluster. POST /api/v1/admin/clusters (adminAuth)
func (h *ClusterHandler) Create(c *gin.Context) {
	var req clusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if msg := normalizeClusterReq(&req); msg != "" {
		httpresp.Unprocessable(c, httpresp.CodeInvalidArgument, msg, nil)
		return
	}
	cluster := &models.Cluster{
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		IsDefault:      req.IsDefault,
		Status:         req.Status,
		K8sAPIEndpoint: req.K8sAPIEndpoint,
		K8sAudience:    req.K8sAudience,
		K8sCAData:      req.K8sCAData,
		ArgoServerURL:  req.ArgoServerURL,
		ArgoNamespace:  req.ArgoNamespace,
		KoordInstalled: req.KoordInstalled,
		ClientQPS:      req.ClientQPS,
		ClientBurst:    req.ClientBurst,
	}
	created, err := h.repo.Create(c.Request.Context(), cluster)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateClusterName) {
			httpresp.Conflict(c, "cluster_name_exists", "cluster name already exists", map[string]any{"name": req.Name})
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	audit.Log(c.Request.Context(), "cluster.create", "clusters", []string{created.ID}, map[string]any{"name": created.Name})
	h.invalidate(created.ID)
	c.JSON(http.StatusCreated, created)
}

// Update mutates an existing cluster. PUT /api/v1/admin/clusters/:id (adminAuth)
func (h *ClusterHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req clusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if msg := normalizeClusterReq(&req); msg != "" {
		httpresp.Unprocessable(c, httpresp.CodeInvalidArgument, msg, nil)
		return
	}
	cluster := &models.Cluster{
		ID:             id,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		IsDefault:      req.IsDefault,
		Status:         req.Status,
		K8sAPIEndpoint: req.K8sAPIEndpoint,
		K8sAudience:    req.K8sAudience,
		K8sCAData:      req.K8sCAData,
		ArgoServerURL:  req.ArgoServerURL,
		ArgoNamespace:  req.ArgoNamespace,
		KoordInstalled: req.KoordInstalled,
		ClientQPS:      req.ClientQPS,
		ClientBurst:    req.ClientBurst,
	}
	updated, err := h.repo.Update(c.Request.Context(), cluster)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateClusterName) {
			httpresp.Conflict(c, "cluster_name_exists", "cluster name already exists", map[string]any{"name": req.Name})
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	if updated == nil {
		httpresp.NotFound(c, "cluster_not_found", "cluster not found")
		return
	}
	audit.Log(c.Request.Context(), "cluster.update", "clusters", []string{id}, map[string]any{"name": updated.Name})
	h.invalidate(id)
	c.JSON(http.StatusOK, updated)
}

// Delete soft-deletes a cluster. DELETE /api/v1/admin/clusters/:id (adminAuth).
// Returns 409 if any ExecutionTarget still references it.
func (h *ClusterHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.SoftDelete(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrClusterInUse) {
			n, _ := h.repo.CountReferencingTargets(c.Request.Context(), id)
			httpresp.Conflict(c, "cluster_in_use",
				"cluster is still referenced by execution_targets",
				map[string]any{"references": n})
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	audit.Log(c.Request.Context(), "cluster.delete", "clusters", []string{id}, nil)
	h.invalidate(id)
	c.Status(http.StatusNoContent)
}
