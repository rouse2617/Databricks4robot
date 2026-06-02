package k8s

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestMetadataTokenSource_Token(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			t.Errorf("missing Metadata-Flavor header, got %q", r.Header.Get("Metadata-Flavor"))
		}
		if got := r.URL.Query().Get("audience"); got != "test-audience" {
			t.Errorf("audience query = %q, want %q", got, "test-audience")
		}
		atomic.AddInt32(&calls, 1)
		_ = json.NewEncoder(w).Encode(metadataTokenResponse{
			AccessToken: "ya29.fake-token",
			ExpiresIn:   3600,
			TokenType:   "Bearer",
		})
	}))
	defer srv.Close()

	m := newMetadataTokenSource("test-audience")
	m.baseURL = srv.URL
	m.httpClient = srv.Client()

	tok, err := m.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok.AccessToken != "ya29.fake-token" {
		t.Fatalf("AccessToken = %q, want %q", tok.AccessToken, "ya29.fake-token")
	}
	if tok.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want Bearer", tok.TokenType)
	}
	if tok.Expiry.IsZero() {
		t.Error("Expiry is zero, want non-zero")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("server calls = %d, want 1", got)
	}
}

func TestMetadataTokenSource_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	m := newMetadataTokenSource("x")
	m.baseURL = srv.URL
	m.httpClient = srv.Client()
	if _, err := m.Token(); err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}

func TestOauth2RoundTripper_InjectsBearer(t *testing.T) {
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	static := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "static-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	})
	rt := &oauth2RoundTripper{base: http.DefaultTransport, token: static}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/foo", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if seenAuth != "Bearer static-token" {
		t.Fatalf("auth header = %q, want %q", seenAuth, "Bearer static-token")
	}
}

func TestOauth2RoundTripper_RespectsExistingAuth(t *testing.T) {
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "should-not-be-used", TokenType: "Bearer"})
	rt := &oauth2RoundTripper{base: http.DefaultTransport, token: static}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/foo", nil)
	req.Header.Set("Authorization", "Bearer per-request-override")
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if seenAuth != "Bearer per-request-override" {
		t.Fatalf("auth header = %q, want explicit override", seenAuth)
	}
}
