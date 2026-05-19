package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHandler_SyncStatus_WithCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, func() SyncInfo {
		return SyncInfo{
			ElasticsearchOK:           true,
			OutboxRelayEnabled:        true,
			OutboxESSubscriberEnabled: true,
			SearchIndexMode:           "outbox_es_subscriber",
			Env:                       "development",
		}
	}, nil)
	r := gin.New()
	r.GET("/sync-status", h.SyncStatus)

	req := httptest.NewRequest(http.MethodGet, "/sync-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"search_index_mode":"outbox_es_subscriber"`) {
		t.Fatalf("body %s", w.Body.String())
	}
}

func TestHandler_SyncStatus_NoCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, nil, nil)
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

func TestHandler_SyncProgress_WithCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, nil, func(context.Context) (SyncProgress, error) {
		return SyncProgress{
			PostgresAssetsTotal:     100,
			ElasticsearchDocsTotal:  95,
			PGESGap:                 5,
			PGESSyncRatio:           0.95,
			OutboxPendingEvents:     12,
			OutboxPendingClaimable:  2,
			OutboxProcessingEvents:  3,
			OutboxRelaySafetyLagSec: 2,
			OldestPendingAgeSec:     3.2,
			CheckedAt:               time.Unix(0, 0).UTC(),
		}, nil
	})
	r := gin.New()
	r.GET("/sync-progress", h.SyncProgress)

	req := httptest.NewRequest(http.MethodGet, "/sync-progress", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"pg_es_gap":5`) {
		t.Fatalf("body %s", w.Body.String())
	}
}
