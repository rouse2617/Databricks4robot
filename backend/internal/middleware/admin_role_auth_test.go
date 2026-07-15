package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// CYB-3229: read-only admin endpoints allow an admin-role principal (web session)
// OR the static admin token, but nothing else.
func TestAdminTokenOrAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const adminToken = "admintok"

	// run returns 200 when the middleware calls Next (allowed), else the aborted
	// status code.
	run := func(setup func(*gin.Context), header string) int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/admin/search/reindex-jobs", nil)
		if header != "" {
			req.Header.Set("X-Admin-Token", header)
		}
		c.Request = req
		if setup != nil {
			setup(c)
		}
		AdminTokenOrAdminRole(adminToken, "dbt", "production")(c)
		if c.IsAborted() {
			return w.Code
		}
		return http.StatusOK
	}

	asAdmin := func(c *gin.Context) { setPrincipal(c, Principal{Role: "admin"}) }
	asUser := func(c *gin.Context) { setPrincipal(c, Principal{Role: "user"}) }

	if code := run(asAdmin, ""); code != http.StatusOK {
		t.Fatalf("admin-role principal (no token): got %d, want 200", code)
	}
	if code := run(nil, adminToken); code != http.StatusOK {
		t.Fatalf("valid static admin token (no principal): got %d, want 200", code)
	}
	if code := run(asUser, ""); code == http.StatusOK {
		t.Fatalf("non-admin principal (no token): got 200, want denied")
	}
	if code := run(nil, ""); code == http.StatusOK {
		t.Fatalf("anonymous (no principal, no token): got 200, want denied")
	}
	if code := run(nil, "wrong"); code == http.StatusOK {
		t.Fatalf("wrong token (no principal): got 200, want denied")
	}
}
