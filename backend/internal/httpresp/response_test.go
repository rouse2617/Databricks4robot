package httpresp

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// captureLogs swaps the default slog logger for one writing JSON to a buffer,
// runs fn, and returns what was logged. Restores the previous default after.
func captureLogs(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	defer slog.SetDefault(prev)
	fn()
	return buf.String()
}

func testCtx(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, rec
}

// CYB-3569: 5xx must be logged server-side (previously every 500 was silent,
// making deploy/projection/argo failures undiagnosable from logs).
func TestError_Logs5xxServerSide(t *testing.T) {
	var rec *httptest.ResponseRecorder
	logs := captureLogs(t, func() {
		var c *gin.Context
		c, rec = testCtx(http.MethodPost, "/api/v1/runs/template/abc")
		Internal(c, "create workflow: boom")
	})

	if !strings.Contains(logs, "http error response") {
		t.Fatalf("expected a server-side error log for 5xx, got: %q", logs)
	}
	if !strings.Contains(logs, "create workflow: boom") {
		t.Fatalf("expected the error message in the log, got: %q", logs)
	}
	// The response body is still the standardized envelope.
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body.Code != CodeInternalError || body.Message != "create workflow: boom" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}

// 4xx are expected client errors and must NOT be logged as server errors
// (avoids drowning the real 5xx signal in noise).
func TestError_Does_Not_Log4xx(t *testing.T) {
	logs := captureLogs(t, func() {
		c, _ := testCtx(http.MethodGet, "/api/v1/things")
		BadRequest(c, CodeInvalidArgument, "nope", nil)
	})
	if strings.Contains(logs, "http error response") {
		t.Fatalf("4xx should not emit a server error log, got: %q", logs)
	}
}
