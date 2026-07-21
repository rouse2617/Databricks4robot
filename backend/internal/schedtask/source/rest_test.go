package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type stubResolver struct {
	m   map[string]string
	err error
}

func (s stubResolver) Resolve(_ context.Context, ref string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.m[ref], nil
}

// graceLikeServer serves a Grace-shaped response over N pages: page P returns
// items [P*size .. P*size+size), each item's last_status_at = base + i seconds.
// It also asserts that the client sent the expected Grace-style filters +
// basic auth.
func graceLikeServer(t *testing.T, total int, base time.Time) *httptest.Server {
	t.Helper()
	pageSize := 200
	mux := http.NewServeMux()
	mux.HandleFunc("/grace/video_steps", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got == "" || strings.Contains(strings.ToLower(got), "go-http-client") {
			t.Errorf("expected browser-like UA, got %q", got)
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
			t.Errorf("expected basic auth, got %q", got)
		} else {
			raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(got, "Basic "))
			if string(raw) != "u1:secretPW" {
				t.Errorf("basic auth mismatch: got %q, want u1:secretPW", string(raw))
			}
		}
		filters := r.URL.Query()["filter"]
		if len(filters) < 3 {
			t.Errorf("expected 3+ filter params (step, status, gte, lt), got %v", filters)
		}
		page := 1
		fmt.Sscanf(r.URL.Query().Get("page"), "%d", &page)
		start := (page - 1) * pageSize
		if start >= total {
			_, _ = w.Write([]byte(`{"data":[],"total":` + itoa(total) + `}`))
			return
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		type item struct {
			VideoID      string `json:"video_id"`
			LastStatusAt string `json:"last_status_at"`
		}
		items := make([]item, 0, end-start)
		for i := start; i < end; i++ {
			items = append(items, item{
				VideoID:      fmt.Sprintf("vid-%04d", i),
				LastStatusAt: base.Add(time.Duration(i) * time.Second).UTC().Format(time.RFC3339),
			})
		}
		resp := map[string]any{"data": items, "total": total, "page": page, "size": pageSize}
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(mux)
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }

// graceConfig builds a RestConfig that mirrors what services/grace-sync does:
// step_key + status + last_status_at filters, page/size paging, data[].video_id.
func graceConfig(baseURL string) RestConfig {
	return RestConfig{
		BaseURL: baseURL,
		Path:    "/grace/video_steps",
		Auth:    AuthConfig{Type: "basic", Username: "u1", SecretRef: "grace/pw"},
		Query: QueryConfig{
			Static: map[string][]string{
				"filter": {"step_key:eq:body_heatmap", "status:eq:success"},
			},
			FilterParam: "filter",
			TimeField:   "last_status_at",
		},
		Paging: PagingConfig{Mode: "page_size", PageParam: "page", SizeParam: "size", PageSize: 200, TotalPath: "total", DataPath: "data"},
		IDPath: "data[].video_id",
	}
}

// TestRestSource_GraceShape_PagesAndAdvancesCursor exercises the Grace
// replication: multi-page fetch, id extraction, basic auth, and cursor
// advance to the max last_status_at.
func TestRestSource_GraceShape_PagesAndAdvancesCursor(t *testing.T) {
	base := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	total := 450 // → 3 pages of size 200
	srv := graceLikeServer(t, total, base)
	defer srv.Close()

	start := base.Add(-time.Hour)
	end := base.Add(time.Hour)
	src := NewREST(graceConfig(srv.URL), stubResolver{m: map[string]string{"grace/pw": "secretPW"}})
	ids, cursor, err := src.Fetch(context.Background(), "", Window{Start: &start, End: &end})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(ids) != total {
		t.Fatalf("ids count = %d, want %d", len(ids), total)
	}
	if ids[0] != "vid-0000" || ids[total-1] != fmt.Sprintf("vid-%04d", total-1) {
		t.Fatalf("first/last ids wrong: %s..%s", ids[0], ids[total-1])
	}
	wantCursor := base.Add(time.Duration(total-1) * time.Second).UTC().Format(time.RFC3339)
	if cursor != wantCursor {
		t.Fatalf("cursor = %q, want %q", cursor, wantCursor)
	}
}

// TestRestSource_EmptyPageTerminates: when the API returns empty data before
// total, we stop cleanly (Grace does this on the page past the last one).
func TestRestSource_EmptyPageTerminates(t *testing.T) {
	base := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	srv := graceLikeServer(t, 0, base) // total=0 → first page returns empty
	defer srv.Close()
	src := NewREST(graceConfig(srv.URL), stubResolver{m: map[string]string{"grace/pw": "secretPW"}})
	end := base
	ids, cursor, err := src.Fetch(context.Background(), "prev-cursor", Window{Start: &base, End: &end})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected 0 ids, got %d", len(ids))
	}
	if cursor != "prev-cursor" {
		t.Fatalf("empty fetch should leave cursor unchanged, got %q", cursor)
	}
}

// TestRestSource_IDsModeShortCircuits: window.IDs → return verbatim, no HTTP.
func TestRestSource_IDsModeShortCircuits(t *testing.T) {
	// A server that MUST NOT be called.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("HTTP called in ids mode")
	}))
	defer srv.Close()
	src := NewREST(graceConfig(srv.URL), stubResolver{m: map[string]string{"grace/pw": "secretPW"}})
	want := []string{"a", "b", "c"}
	got, cur, err := src.Fetch(context.Background(), "keep-me", Window{IDs: want})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	if cur != "keep-me" {
		t.Fatalf("cursor changed on ids-mode: %q", cur)
	}
}

// TestRestSource_FailsClosedWithoutResolver: a secret_ref-bearing rule with a
// nil resolver MUST fail — never fall back to env or plaintext.
func TestRestSource_FailsClosedWithoutResolver(t *testing.T) {
	cfg := graceConfig("http://example.invalid")
	src := NewREST(cfg, nil)
	base := time.Now()
	_, _, err := src.Fetch(context.Background(), "", Window{Start: &base, End: &base})
	if err == nil {
		t.Fatal("expected fail-closed error without resolver")
	}
	if !strings.Contains(err.Error(), "resolver") {
		t.Fatalf("expected resolver error, got %v", err)
	}
}

// TestRestSource_TimeQueryParamsInsteadOfFilter: not every API uses Grace's
// filter= repetition. When TimeQueryStart/End are set (and FilterParam empty),
// the source emits ordinary ?since=&until= params.
func TestRestSource_TimeQueryParamsInsteadOfFilter(t *testing.T) {
	var sawStart, sawEnd string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawStart = r.URL.Query().Get("since")
		sawEnd = r.URL.Query().Get("until")
		if len(r.URL.Query()["filter"]) != 0 {
			t.Errorf("did not expect filter= params, got %v", r.URL.Query()["filter"])
		}
		_, _ = w.Write([]byte(`{"data":[],"total":0}`))
	}))
	defer srv.Close()
	cfg := RestConfig{
		BaseURL: srv.URL,
		Path:    "/x",
		Query: QueryConfig{
			TimeField:      "ts",
			TimeQueryStart: "since",
			TimeQueryEnd:   "until",
		},
		Paging: PagingConfig{Mode: "page_size", PageSize: 100, PageParam: "page", SizeParam: "size", DataPath: "data"},
		IDPath: "data[].id",
	}
	src := NewREST(cfg, nil)
	start := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	if _, _, err := src.Fetch(context.Background(), "", Window{Start: &start, End: &end}); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.Contains(sawStart, "2026-07-21T10:00:00") {
		t.Fatalf("since=%q missing expected value", sawStart)
	}
	if !strings.Contains(sawEnd, "2026-07-21T10:30:00") {
		t.Fatalf("until=%q missing expected value", sawEnd)
	}
}

// TestExtractIDs_Basic sanity-checks the id_path walker directly.
func TestExtractIDs_Basic(t *testing.T) {
	body := []byte(`{"data":[{"video_id":"a"},{"video_id":"b"},{}],"total":3}`)
	got, err := extractIDs(body, "data[].video_id")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != "[a b]" {
		t.Fatalf("got %v, want [a b]", got)
	}
}
