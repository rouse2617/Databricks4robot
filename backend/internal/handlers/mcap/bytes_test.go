package mcap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

type memBytesSource struct {
	b   []byte
	ct  string
	uri string
}

func (m *memBytesSource) Stat(context.Context, string) (int64, string, error) {
	return int64(len(m.b)), m.ct, nil
}
func (m *memBytesSource) OpenRange(ctx context.Context, _ string, start, length int64) (io.ReadCloser, error) {
	_ = ctx
	end := start + length
	if start < 0 || length < 0 || end > int64(len(m.b)) {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	return io.NopCloser(bytes.NewReader(m.b[start:end])), nil
}
func (m *memBytesSource) OpenFull(ctx context.Context, _ string) (io.ReadCloser, error) {
	_ = ctx
	return io.NopCloser(bytes.NewReader(m.b)), nil
}

type errBytesSource struct {
	statErr      error
	openFullErr  error
	openRangeErr error
}

func (e *errBytesSource) Stat(context.Context, string) (int64, string, error) {
	if e.statErr != nil {
		return 0, "", e.statErr
	}
	return 10, "application/octet-stream", nil
}

func (e *errBytesSource) OpenRange(context.Context, string, int64, int64) (io.ReadCloser, error) {
	if e.openRangeErr != nil {
		return nil, e.openRangeErr
	}
	return io.NopCloser(bytes.NewReader([]byte("2345"))), nil
}

func (e *errBytesSource) OpenFull(context.Context, string) (io.ReadCloser, error) {
	if e.openFullErr != nil {
		return nil, e.openFullErr
	}
	return io.NopCloser(bytes.NewReader([]byte("0123456789"))), nil
}

type headAwareBytesSource struct{}

func (h *headAwareBytesSource) Stat(context.Context, string) (int64, string, error) {
	return 10, "application/octet-stream", nil
}
func (h *headAwareBytesSource) OpenRange(context.Context, string, int64, int64) (io.ReadCloser, error) {
	return nil, errors.New("OpenRange should not be called for HEAD")
}
func (h *headAwareBytesSource) OpenFull(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("OpenFull should not be called for HEAD")
}

func TestBytes_NoRange_Returns200Full(t *testing.T) {
	repo := &mockMcapRepo{
		getFn: func(context.Context, string) (*models.McapFile, error) {
			return &models.McapFile{McapFileID: "abcd1234", GCSPath: "gs://b/o"}, nil
		},
	}
	h := New(repo)
	h.SetBytesSource(&memBytesSource{b: []byte("0123456789"), ct: "application/octet-stream"})
	r := setupMcapRouter(http.MethodGet, "/mcap-files/:id/bytes", h.Bytes)

	w := doMcapReq(t, r, http.MethodGet, "/mcap-files/abcd1234/bytes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("Accept-Ranges expected bytes, got %q", got)
	}
	if got := w.Header().Get("Content-Length"); got != "10" {
		t.Fatalf("Content-Length expected 10, got %q", got)
	}
	if got := w.Body.String(); got != "0123456789" {
		t.Fatalf("unexpected body %q", got)
	}
}

func TestBytes_WithRange_Returns206(t *testing.T) {
	repo := &mockMcapRepo{
		getFn: func(context.Context, string) (*models.McapFile, error) {
			return &models.McapFile{McapFileID: "abcd1234", GCSPath: "gs://b/o"}, nil
		},
	}
	h := New(repo)
	h.SetBytesSource(&memBytesSource{b: []byte("0123456789")})
	r := setupMcapRouter(http.MethodGet, "/mcap-files/:id/bytes", h.Bytes)

	req := httptest.NewRequest(http.MethodGet, "/mcap-files/abcd1234/bytes", nil)
	req.Header.Set("Range", "bytes=2-5")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusPartialContent {
		t.Fatalf("expected 206, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Fatalf("Content-Range expected %q, got %q", "bytes 2-5/10", got)
	}
	if got := w.Header().Get("Content-Length"); got != "4" {
		t.Fatalf("Content-Length expected 4, got %q", got)
	}
	if got := w.Body.String(); got != "2345" {
		t.Fatalf("unexpected body %q", got)
	}
}

func TestBytes_Head_DoesNotReadBody(t *testing.T) {
	repo := &mockMcapRepo{
		getFn: func(context.Context, string) (*models.McapFile, error) {
			return &models.McapFile{McapFileID: "abcd1234", GCSPath: "gs://b/o"}, nil
		},
	}
	h := New(repo)
	h.SetBytesSource(&headAwareBytesSource{})
	r := gin.New()
	r.GET("/mcap-files/:id/bytes", h.Bytes)
	r.HEAD("/mcap-files/:id/bytes", h.Bytes)

	req := httptest.NewRequest(http.MethodHead, "/mcap-files/abcd1234/bytes", nil)
	req.Header.Set("Range", "bytes=2-5")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusPartialContent {
		t.Fatalf("expected 206, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Fatalf("Content-Range expected %q, got %q", "bytes 2-5/10", got)
	}
	if got := w.Header().Get("Content-Length"); got != "4" {
		t.Fatalf("Content-Length expected 4, got %q", got)
	}
}

func TestBytes_InvalidRange_Returns416(t *testing.T) {
	repo := &mockMcapRepo{
		getFn: func(context.Context, string) (*models.McapFile, error) {
			return &models.McapFile{McapFileID: "abcd1234", GCSPath: "gs://b/o"}, nil
		},
	}
	h := New(repo)
	h.SetBytesSource(&memBytesSource{b: []byte("0123456789")})
	r := setupMcapRouter(http.MethodGet, "/mcap-files/:id/bytes", h.Bytes)

	req := httptest.NewRequest(http.MethodGet, "/mcap-files/abcd1234/bytes", nil)
	req.Header.Set("Range", "bytes=50-60")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("expected 416, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Range"); got != "bytes */10" {
		t.Fatalf("Content-Range expected %q, got %q", "bytes */10", got)
	}
	var body httpresp.ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v body=%s", err, w.Body.String())
	}
	if body.Code != httpresp.CodeInvalidRange {
		t.Fatalf("expected code %q, got %q", httpresp.CodeInvalidRange, body.Code)
	}
	if body.Message == "" {
		t.Fatalf("expected non-empty message")
	}
	if body.Details == nil || body.Details["size"] == nil {
		t.Fatalf("expected details.size, got %#v", body.Details)
	}
}

func TestBytes_RepoNotConfigured_Returns503(t *testing.T) {
	h := New(nil)
	h.SetBytesSource(&memBytesSource{b: []byte("0123456789")})
	r := setupMcapRouter(http.MethodGet, "/mcap-files/:id/bytes", h.Bytes)

	w := doMcapReq(t, r, http.MethodGet, "/mcap-files/abcd1234/bytes", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
	var body httpresp.ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v body=%s", err, w.Body.String())
	}
	if body.Code != httpresp.CodeServiceUnavailable {
		t.Fatalf("expected code %q, got %q", httpresp.CodeServiceUnavailable, body.Code)
	}
}

func TestBytes_StorageObjectMissing_Returns404(t *testing.T) {
	repo := &mockMcapRepo{
		getFn: func(context.Context, string) (*models.McapFile, error) {
			return &models.McapFile{McapFileID: "abcd1234", GCSPath: "gs://b/o"}, nil
		},
	}

	t.Run("stat missing", func(t *testing.T) {
		h := New(repo)
		h.SetBytesSource(&errBytesSource{statErr: storage.ErrObjectNotExist})
		r := setupMcapRouter(http.MethodGet, "/mcap-files/:id/bytes", h.Bytes)
		w := doMcapReq(t, r, http.MethodGet, "/mcap-files/abcd1234/bytes", nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
		}
		var body httpresp.ErrorBody
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal body: %v body=%s", err, w.Body.String())
		}
		if body.Code != httpresp.CodeMcapFileNotFound {
			t.Fatalf("expected code %q, got %q", httpresp.CodeMcapFileNotFound, body.Code)
		}
	})

	t.Run("full read missing", func(t *testing.T) {
		h := New(repo)
		h.SetBytesSource(&errBytesSource{openFullErr: storage.ErrObjectNotExist})
		r := setupMcapRouter(http.MethodGet, "/mcap-files/:id/bytes", h.Bytes)
		w := doMcapReq(t, r, http.MethodGet, "/mcap-files/abcd1234/bytes", nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
		}
		var body httpresp.ErrorBody
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal body: %v body=%s", err, w.Body.String())
		}
		if body.Code != httpresp.CodeMcapFileNotFound {
			t.Fatalf("expected code %q, got %q", httpresp.CodeMcapFileNotFound, body.Code)
		}
	})

	t.Run("range read missing", func(t *testing.T) {
		h := New(repo)
		h.SetBytesSource(&errBytesSource{openRangeErr: storage.ErrObjectNotExist})
		r := setupMcapRouter(http.MethodGet, "/mcap-files/:id/bytes", h.Bytes)
		req := httptest.NewRequest(http.MethodGet, "/mcap-files/abcd1234/bytes", nil)
		req.Header.Set("Range", "bytes=2-5")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
		}
		var body httpresp.ErrorBody
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal body: %v body=%s", err, w.Body.String())
		}
		if body.Code != httpresp.CodeMcapFileNotFound {
			t.Fatalf("expected code %q, got %q", httpresp.CodeMcapFileNotFound, body.Code)
		}
	})
}
