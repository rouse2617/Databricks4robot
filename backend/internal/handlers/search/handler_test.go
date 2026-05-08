package search

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandler_SyncStatus_WithCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, func() SyncInfo {
		return SyncInfo{
			ElasticsearchOK: true,
			CDCEnabled:      true,
			CDCSourceDriver: "debezium-kafka",
			SearchIndexMode: "cdc",
			Env:             "development",
		}
	})
	r := gin.New()
	r.GET("/sync-status", h.SyncStatus)

	req := httptest.NewRequest(http.MethodGet, "/sync-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"search_index_mode":"cdc"`) {
		t.Fatalf("body %s", w.Body.String())
	}
}

func TestHandler_SyncStatus_NoCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, nil)
	r := gin.New()
	r.GET("/sync-status", h.SyncStatus)

	req := httptest.NewRequest(http.MethodGet, "/sync-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"search_index_mode":"unavailable"`) {
		t.Fatalf("body %s", w.Body.String())
	}
}
