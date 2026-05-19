package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/foxglove/mcap/go/mcap"
	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/gcsrs"
	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/manifest"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeUpstream returns an httptest.Server that serves /api/v1/assets/:id/mcap-locator
// with handler f. f receives the asset id and the request itself.
func fakeUpstream(t *testing.T, f func(t *testing.T, id string, r *http.Request) (status int, body string)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/assets/", func(w http.ResponseWriter, r *http.Request) {
		// /api/v1/assets/{id}/mcap-locator
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/")
		path = strings.TrimSuffix(path, "/mcap-locator")
		status, body := f(t, path, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	return httptest.NewServer(mux)
}

// buildFixtureMCAP synthesizes a tiny indexed MCAP with one video-ish channel
// and two chunks so manifest tests can verify topic + chunk enumeration
// without touching GCS.
func buildFixtureMCAP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := mcap.NewWriter(&buf, &mcap.WriterOptions{
		Chunked:     true,
		ChunkSize:   64,
		Compression: "",
		IncludeCRC:  true,
	})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.WriteHeader(&mcap.Header{Profile: "test", Library: "mcap-preview-test"}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if err := w.WriteSchema(&mcap.Schema{
		ID:       1,
		Name:     "foxglove.CompressedVideo",
		Encoding: "protobuf",
		Data:     []byte("fake-schema"),
	}); err != nil {
		t.Fatalf("WriteSchema: %v", err)
	}
	if err := w.WriteSchema(&mcap.Schema{
		ID:       2,
		Name:     "std_msgs/String",
		Encoding: "ros1",
		Data:     []byte("fake"),
	}); err != nil {
		t.Fatalf("WriteSchema: %v", err)
	}
	if err := w.WriteChannel(&mcap.Channel{
		ID:              1,
		SchemaID:        1,
		Topic:           "/camera/front",
		MessageEncoding: "protobuf",
	}); err != nil {
		t.Fatalf("WriteChannel video: %v", err)
	}
	if err := w.WriteChannel(&mcap.Channel{
		ID:              2,
		SchemaID:        2,
		Topic:           "/diagnostics",
		MessageEncoding: "ros1",
	}); err != nil {
		t.Fatalf("WriteChannel string: %v", err)
	}
	// Two chunks, well separated in time, so the window filter has work
	// to do. ChunkSize is 64 bytes so each message flushes a chunk.
	for i, ts := range []uint64{1_000_000_000, 2_000_000_000, 5_000_000_000, 6_000_000_000} {
		ch := uint16(1)
		if i%2 == 1 {
			ch = 2
		}
		if err := w.WriteMessage(&mcap.Message{
			ChannelID: ch,
			Sequence:  uint32(i),
			LogTime:   ts,
			Data:      []byte("payload-" + string(rune('A'+i))),
		}); err != nil {
			t.Fatalf("WriteMessage: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}
	return buf.Bytes()
}

func TestHealthz(t *testing.T) {
	r := New(Config{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v", body)
	}
}

func TestManifest_HappyPath(t *testing.T) {
	mcapBytes := buildFixtureMCAP(t)

	upstream := fakeUpstream(t, func(t *testing.T, id string, r *http.Request) (int, string) {
		if got := r.Header.Get("X-Grace-Token"); got != "tok" {
			t.Fatalf("upstream did not see token: %q", got)
		}
		if id != "abc123" {
			t.Fatalf("upstream got id=%q", id)
		}
		return 200, `{
			"asset_id":"abc123",
			"lifecycle_state":"ready",
			"mcap":{"mcap_uri":"gs://b/o.mcap","size_bytes":` +
			itoa(len(mcapBytes)) + `},
			"window":{"start_timestamp_ns":1500000000,"end_timestamp_ns":2500000000,"duration_ms":1000}
		}`
	})
	defer upstream.Close()

	openMCAP := func(_ context.Context, uri string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
		if uri != "gs://b/o.mcap" {
			t.Fatalf("unexpected uri %q", uri)
		}
		return bytes.NewReader(mcapBytes), &gcsrs.Stats{RangeCalls: 3, BytesRead: 16384}, nil, nil
	}

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP:              openMCAP,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/manifest", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp manifest.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.AssetID != "abc123" {
		t.Errorf("asset_id=%q", resp.AssetID)
	}
	if resp.Mcap.McapURI != "gs://b/o.mcap" {
		t.Errorf("mcap_uri=%q", resp.Mcap.McapURI)
	}
	if len(resp.CandidateVideoTopics) != 1 || resp.CandidateVideoTopics[0].Topic != "/camera/front" {
		t.Errorf("candidate topics=%+v", resp.CandidateVideoTopics)
	}
	if len(resp.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(resp.Sources))
	}
	if resp.Sources[0].Kind != "live" {
		t.Fatalf("expected live source, got %+v", resp.Sources[0])
	}
	if !strings.Contains(resp.Sources[0].URL, "/segment.mp4?topic=%2Fcamera%2Ffront") {
		t.Fatalf("unexpected source url: %s", resp.Sources[0].URL)
	}
	if resp.RecommendedSourceID == "" {
		t.Fatalf("recommended_source_id must be set")
	}
	if len(resp.ChunksInWindow) == 0 {
		t.Errorf("expected at least one chunk in window, got 0")
	}
	if got := resp.Stats["gcs_range_requests"]; got == nil {
		t.Errorf("missing stats: %+v", resp.Stats)
	}
}

func TestManifest_UpstreamNotFoundPropagates(t *testing.T) {
	upstream := fakeUpstream(t, func(_ *testing.T, _ string, _ *http.Request) (int, string) {
		return 404, `{"code":"ASSET_NOT_FOUND","message":"asset abc not found","request_id":"r1"}`
	})
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			t.Fatal("OpenMCAP should not be called when upstream returns 404")
			return nil, nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/manifest", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"ASSET_NOT_FOUND"`) {
		t.Errorf("expected upstream code propagated, got %s", w.Body.String())
	}
}

func TestManifest_UpstreamServiceUnavailablePropagates(t *testing.T) {
	upstream := fakeUpstream(t, func(_ *testing.T, _ string, _ *http.Request) (int, string) {
		return 503, `{"code":"SERVICE_UNAVAILABLE","message":"upstream unavailable","request_id":"r1","details":{"hint":"try later"}}`
	})
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			t.Fatal("OpenMCAP should not be called when upstream returns 503")
			return nil, nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/manifest", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != 503 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"SERVICE_UNAVAILABLE"`) {
		t.Errorf("expected upstream code propagated, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"try later"`) {
		t.Errorf("expected upstream details propagated, got %s", w.Body.String())
	}
}

func TestManifest_UpstreamInvalidRangePropagates(t *testing.T) {
	upstream := fakeUpstream(t, func(_ *testing.T, _ string, _ *http.Request) (int, string) {
		return 416, `{"code":"INVALID_RANGE","message":"invalid Range header","request_id":"r1","details":{"range":"bytes=50-60","size":10}}`
	})
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			t.Fatal("OpenMCAP should not be called when upstream returns 416")
			return nil, nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/manifest", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != 416 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"INVALID_RANGE"`) {
		t.Errorf("expected upstream code propagated, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"bytes=50-60"`) {
		t.Errorf("expected upstream details propagated, got %s", w.Body.String())
	}
}

func TestManifest_MissingTokenIs401(t *testing.T) {
	r := New(Config{UpstreamBaseURL: "http://unused", OpenMCAP: func(context.Context, string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
		t.Fatal("should not be called")
		return nil, nil, nil, nil
	}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/manifest", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestManifest_BadAssetIDIs400(t *testing.T) {
	r := New(Config{UpstreamBaseURL: "http://unused", OpenMCAP: func(context.Context, string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
		return nil, nil, nil, nil
	}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/bad%20id/manifest", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

// tiny helper to avoid importing strconv in the JSON literal above.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
