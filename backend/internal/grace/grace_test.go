package grace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// graceStub serves paginated /grace/videos and records the auth + UA it saw.
func graceStub(t *testing.T, pages [][]video, total int) (*httptest.Server, *struct {
	gotUA   string
	gotUser string
	gotPass string
}) {
	t.Helper()
	seen := &struct {
		gotUA   string
		gotUser string
		gotPass string
	}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.gotUA = r.Header.Get("User-Agent")
		seen.gotUser, seen.gotPass, _ = r.BasicAuth()
		if r.URL.Path != "/grace/videos" {
			w.WriteHeader(404)
			return
		}
		pageStr := r.URL.Query().Get("page")
		idx := 0
		if pageStr == "2" {
			idx = 1
		}
		var data []video
		if idx < len(pages) {
			data = pages[idx]
		}
		_ = json.NewEncoder(w).Encode(videoListResp{Data: data, Total: total})
	}))
	t.Cleanup(srv.Close)
	return srv, seen
}

func TestFetchVideoDurations_PaginatesAndFiltersAndAuths(t *testing.T) {
	pages := [][]video{
		{{ID: "a", DurationSec: 120.5}, {ID: "b", DurationSec: 0}}, // b has no duration → skipped
		{{ID: "c", DurationSec: 300}},
	}
	srv, seen := graceStub(t, pages, 3)
	c := NewClient(Config{BaseURL: srv.URL, Username: "u", Password: "p"})
	got, err := c.FetchVideoDurations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"a": 120.5, "c": 300}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("durations = %v, want %v", got, want)
	}
	if seen.gotUA != "curl/8" {
		t.Fatalf("User-Agent = %q, want curl/8 (Cloudflare bypass)", seen.gotUA)
	}
	if seen.gotUser != "u" || seen.gotPass != "p" {
		t.Fatalf("basic auth = %q/%q, want u/p", seen.gotUser, seen.gotPass)
	}
}

type fakeRepo struct{ got map[string]float64 }

func (f *fakeRepo) Upsert(_ context.Context, d map[string]float64) error {
	f.got = d
	return nil
}

func TestSyncAll_FetchesAndUpserts(t *testing.T) {
	srv, _ := graceStub(t, [][]video{{{ID: "x", DurationSec: 42}}}, 1)
	repo := &fakeRepo{}
	s := NewSyncer(NewClient(Config{BaseURL: srv.URL, Username: "u", Password: "p"}), repo)
	if err := s.SyncAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repo.got["x"] != 42 {
		t.Fatalf("upserted = %v, want x=42", repo.got)
	}
}

func TestConfigEnabledAndPasswordJSON(t *testing.T) {
	if (Config{}).Enabled() {
		t.Fatal("empty config should be disabled")
	}
	if !(Config{BaseURL: "x", Username: "u", Password: "p"}).Enabled() {
		t.Fatal("full config should be enabled")
	}
	if got := parsePassword(`{"AUTH_PASSWORD":"secret"}`); got != "secret" { // pragma: allowlist secret
		t.Fatalf("json password parse = %q", got)
	}
	if got := parsePassword("plain"); got != "plain" {
		t.Fatalf("plain password = %q", got)
	}
}
