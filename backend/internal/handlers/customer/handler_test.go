package customer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type stubCustomerRepo struct {
	byID map[string]*models.Customer
}

func (s *stubCustomerRepo) Insert(_ context.Context, c *models.Customer) error {
	if s.byID == nil {
		s.byID = map[string]*models.Customer{}
	}
	if _, ok := s.byID[c.CustomerID]; ok {
		return repository.ErrDuplicateCustomerID
	}
	cp := *c
	s.byID[c.CustomerID] = &cp
	return nil
}

func (s *stubCustomerRepo) Get(_ context.Context, id string) (*models.Customer, error) {
	if c, ok := s.byID[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, nil
}

func (s *stubCustomerRepo) Update(_ context.Context, c *models.Customer) error {
	if _, ok := s.byID[c.CustomerID]; !ok {
		return repository.ErrOptimisticLock
	}
	cp := *c
	s.byID[c.CustomerID] = &cp
	return nil
}

func (s *stubCustomerRepo) Exists(_ context.Context, id string) (bool, error) {
	_, ok := s.byID[id]
	return ok, nil
}

func (s *stubCustomerRepo) List(_ context.Context, _, _, _ string, _ int, _ string) ([]*models.Customer, error) {
	var out []*models.Customer
	for _, c := range s.byID {
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
}

func TestValidateCustomerID(t *testing.T) {
	if err := ValidateCustomerID("cust_alpha"); err != nil {
		t.Fatalf("expected valid slug: %v", err)
	}
	if err := ValidateCustomerID("BAD"); err == nil {
		t.Fatal("expected invalid slug")
	}
}

func TestCreateAndGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubCustomerRepo{byID: map[string]*models.Customer{}}
	h := New(repo)
	r := gin.New()
	r.POST("/customers", h.Create)
	r.GET("/customers/:customer_id", h.Get)

	body := `{"customer_id":"cust_test01","display_name":"Test Co"}`
	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/customers/cust_test01", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", w2.Code, w2.Body.String())
	}
}
