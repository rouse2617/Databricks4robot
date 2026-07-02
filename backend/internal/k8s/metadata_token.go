package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// metadataTokenSource fetches a Google Cloud metadata-server access token
// bound to a specific audience, suitable for Workload Identity access to
// GKE (or any GCP service that accepts Google-issued OIDC tokens).
//
// Used by buildConfig when K8S_USE_METADATA_TOKEN=true. The metadata server
// is reachable at http://metadata.google.internal/computeMetadata/v1 from
// any GCE, GKE, or Cloud Run instance with the appropriate access.
//
// Implements oauth2.TokenSource. Wrap with transport.NewCachedTokenSource to
// fit k8s.io/client-go's BearerTokenSource contract. Tokens have a 1h
// lifetime; the cached wrapper refreshes on expiry.
type metadataTokenSource struct {
	audience   string
	baseURL    string
	httpClient *http.Client
}

type metadataTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

const defaultMetadataBaseURL = "http://metadata.google.internal/computeMetadata/v1"

func newMetadataTokenSource(audience string) *metadataTokenSource {
	return &metadataTokenSource{
		audience: audience,
		baseURL:  defaultMetadataBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Token satisfies oauth2.TokenSource.
func (m *metadataTokenSource) Token() (*oauth2.Token, error) {
	return m.TokenCtx(context.Background())
}

// TokenCtx is the context-aware variant; preferred so client-go's retries
// respect deadlines.
func (m *metadataTokenSource) TokenCtx(ctx context.Context) (*oauth2.Token, error) {
	url := fmt.Sprintf("%s/instance/service-accounts/default/token?audience=%s", m.baseURL, m.audience)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build metadata request: %w", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call metadata server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("metadata server returned %d: %s", resp.StatusCode, string(body))
	}
	var out metadataTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode metadata response: %w", err)
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("metadata server returned empty access_token")
	}
	lifetime := time.Duration(out.ExpiresIn) * time.Second
	if lifetime <= 0 {
		lifetime = time.Hour
	}
	return &oauth2.Token{
		AccessToken: out.AccessToken,
		TokenType:   out.TokenType,
		Expiry:      time.Now().Add(lifetime),
	}, nil
}

// oauth2RoundTripper wraps a base http.RoundTripper, fetching a fresh
// bearer token from the supplied oauth2.TokenSource for every request
// and injecting it as the Authorization header. The token source
// itself handles caching / refresh (e.g. transport.NewCachedTokenSource).
type oauth2RoundTripper struct {
	base  http.RoundTripper
	token oauth2.TokenSource
}

func newOauth2RoundTripper(src oauth2.TokenSource) func(http.RoundTripper) http.RoundTripper {
	return func(rt http.RoundTripper) http.RoundTripper {
		return &oauth2RoundTripper{base: rt, token: src}
	}
}

func (rt *oauth2RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") != "" {
		// Respect explicit per-request override.
		return rt.base.RoundTrip(req)
	}
	tok, err := rt.token.Token()
	if err != nil {
		return nil, err
	}
	r2 := req.Clone(req.Context())
	if r2.Header == nil {
		r2.Header = make(http.Header)
	} else {
		r2.Header = req.Header.Clone()
	}
	r2.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	return rt.base.RoundTrip(r2)
}
