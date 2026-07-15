package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// inMemoryClusterRepo is a purely in-memory ClusterRepository for handler
// tests (no DB round-trip). Enforces the same invariants as the real repo:
// unique names, soft-delete respecting references, empty items → [] not nil.
type inMemoryClusterRepo struct {
	mu       sync.Mutex
	clusters map[string]*models.Cluster
	refCount map[string]int // clusterID → number of ExecutionTargets referencing it
	nextID   int
}

func newInMemoryClusterRepo() *inMemoryClusterRepo {
	return &inMemoryClusterRepo{
		clusters: map[string]*models.Cluster{},
		refCount: map[string]int{},
	}
}

func (r *inMemoryClusterRepo) List(_ context.Context, includeDeleted bool) ([]*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*models.Cluster, 0, len(r.clusters))
	for _, c := range r.clusters {
		if !includeDeleted && c.DeletedAt != nil {
			continue
		}
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
}

func (r *inMemoryClusterRepo) Get(_ context.Context, id string) (*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clusters[id]
	if !ok || c.DeletedAt != nil {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *inMemoryClusterRepo) GetByName(_ context.Context, name string) (*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.clusters {
		if c.Name == name && c.DeletedAt == nil {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *inMemoryClusterRepo) Create(_ context.Context, c *models.Cluster) (*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.clusters {
		if existing.Name == c.Name && existing.DeletedAt == nil {
			return nil, repository.ErrDuplicateClusterName
		}
	}
	if c.ID == "" {
		r.nextID++
		c.ID = "cluster-test-" + itoa(r.nextID)
	}
	cp := *c
	r.clusters[c.ID] = &cp
	return &cp, nil
}

func (r *inMemoryClusterRepo) Update(_ context.Context, c *models.Cluster) (*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.clusters[c.ID]
	if !ok || existing.DeletedAt != nil {
		return nil, nil
	}
	// Reject name collision (mirrors DB uniqueness).
	for id, other := range r.clusters {
		if id != c.ID && other.Name == c.Name && other.DeletedAt == nil {
			return nil, repository.ErrDuplicateClusterName
		}
	}
	cp := *c
	r.clusters[c.ID] = &cp
	return &cp, nil
}

func (r *inMemoryClusterRepo) SoftDelete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.refCount[id] > 0 {
		return repository.ErrClusterInUse
	}
	c, ok := r.clusters[id]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	c.DeletedAt = &now
	return nil
}

func (r *inMemoryClusterRepo) CountReferencingTargets(_ context.Context, clusterID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.refCount[clusterID], nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func newClusterTestServer(repo *inMemoryClusterRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewClusterHandler(repo)
	r.GET("/api/v1/clusters", h.List)
	r.GET("/api/v1/clusters/:id", h.Get)
	r.POST("/api/v1/admin/clusters", h.Create)
	r.PUT("/api/v1/admin/clusters/:id", h.Update)
	r.DELETE("/api/v1/admin/clusters/:id", h.Delete)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&b).Encode(body)
	}
	req := httptest.NewRequest(method, path, &b)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── tests ──────────────────────────────────────────────────────────────────

func TestClusterHandler_ListEmpty_ReturnsEmptyArray(t *testing.T) {
	r := newClusterTestServer(newInMemoryClusterRepo())
	w := doJSON(t, r, http.MethodGet, "/api/v1/clusters", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []*models.Cluster `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Items == nil {
		t.Fatal("items must be []; got null")
	}
	if len(resp.Items) != 0 {
		t.Fatalf("want 0 items, got %d", len(resp.Items))
	}
}

func TestClusterHandler_Create_Success(t *testing.T) {
	r := newClusterTestServer(newInMemoryClusterRepo())
	body := map[string]any{
		"name":           "delivery-clust",
		"displayName":    "Delivery cluster",
		"description":    "cyberorigin-delivery project",
		"koordInstalled": true,
	}
	w := doJSON(t, r, http.MethodPost, "/api/v1/admin/clusters", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d body=%s", w.Code, w.Body.String())
	}
	var created models.Cluster
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Name != "delivery-clust" || created.ID == "" {
		t.Fatalf("unexpected cluster: %+v", created)
	}
	if created.Status != "available" {
		t.Errorf("expected default status 'available', got %q", created.Status)
	}
	if created.DisplayName != "Delivery cluster" {
		t.Errorf("displayName not persisted: %+v", created)
	}
}

func TestClusterHandler_Create_DuplicateName(t *testing.T) {
	repo := newInMemoryClusterRepo()
	r := newClusterTestServer(repo)
	body := map[string]any{"name": "delivery-clust"}
	_ = doJSON(t, r, http.MethodPost, "/api/v1/admin/clusters", body)
	w := doJSON(t, r, http.MethodPost, "/api/v1/admin/clusters", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("second create should 409, got %d", w.Code)
	}
}

func TestClusterHandler_Create_MissingName(t *testing.T) {
	r := newClusterTestServer(newInMemoryClusterRepo())
	w := doJSON(t, r, http.MethodPost, "/api/v1/admin/clusters", map[string]any{"displayName": "no name"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing name should 422, got %d", w.Code)
	}
}

func TestClusterHandler_Create_InvalidNameChars(t *testing.T) {
	r := newClusterTestServer(newInMemoryClusterRepo())
	w := doJSON(t, r, http.MethodPost, "/api/v1/admin/clusters", map[string]any{"name": "bad name!"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad-name-chars should 422, got %d", w.Code)
	}
}

func TestClusterHandler_UpdateAndDelete_HappyPath(t *testing.T) {
	repo := newInMemoryClusterRepo()
	r := newClusterTestServer(repo)
	// create
	w := doJSON(t, r, http.MethodPost, "/api/v1/admin/clusters", map[string]any{"name": "target-clust"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", w.Code)
	}
	var created models.Cluster
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	// update
	w = doJSON(t, r, http.MethodPut, "/api/v1/admin/clusters/"+created.ID, map[string]any{
		"name":        "target-clust",
		"description": "renamed",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %d", w.Code)
	}
	// delete
	w = doJSON(t, r, http.MethodDelete, "/api/v1/admin/clusters/"+created.ID, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete should 204, got %d", w.Code)
	}
	// list should now be empty
	w = doJSON(t, r, http.MethodGet, "/api/v1/clusters", nil)
	var resp struct {
		Items []*models.Cluster `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Items) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(resp.Items))
	}
}

func TestClusterHandler_Delete_InUse(t *testing.T) {
	repo := newInMemoryClusterRepo()
	r := newClusterTestServer(repo)
	w := doJSON(t, r, http.MethodPost, "/api/v1/admin/clusters", map[string]any{"name": "busy-clust"})
	var created models.Cluster
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	// simulate an ExecutionTarget referencing this cluster
	repo.refCount[created.ID] = 3

	w = doJSON(t, r, http.MethodDelete, "/api/v1/admin/clusters/"+created.ID, nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("delete with references should 409, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestClusterHandler_Get_NotFound(t *testing.T) {
	r := newClusterTestServer(newInMemoryClusterRepo())
	w := doJSON(t, r, http.MethodGet, "/api/v1/clusters/no-such-id", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing get should 404, got %d", w.Code)
	}
}
