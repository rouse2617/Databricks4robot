package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SourceResolver resolves a source-specific ID to a GCS path (bucket/object).
type SourceResolver interface {
	// Resolve returns "bucket/object" for the given ID and environment.
	Resolve(ctx context.Context, id, env string) (string, error)
}

// ---------------------------------------------------------------------------
// Grace API resolver
//
// Uses Basic Auth (base64-encoded username:password). Grace does not use
// Bearer tokens — the /auth/token endpoint does not exist on its API.
// This is a temporary integration; the resolver interface exists so Grace
// can be swapped out later.
// ---------------------------------------------------------------------------

type GraceResolver struct {
	devURL, devUsername, devPassword    string
	prodURL, prodUsername, prodPassword string
}

// -- Grace API response types (nested from full Video model) --
//
// GET /grace/videos/{uuid}
// → storage_meta.gcs.video is a plain string (GCS URI of raw MCAP/video)
// → storage_meta.gcs.algo_inputs is a dict {name: {uri, width, height, …}}

type graceVideoResp struct {
	StorageMeta *graceStorageMeta `json:"storage_meta"`
}

type graceStorageMeta struct {
	Gcs *graceStorageMedium `json:"gcs"`
}

// StorageMediumMetadata (GCS branch) — "video" is a raw GCS string, not a nested object.
type graceStorageMedium struct {
	Video      string                      `json:"video"`
	AlgoInputs map[string]graceVariantMeta `json:"algo_inputs"`
}

type graceVariantMeta struct {
	URI string `json:"uri"`
}

// GET /grace/videos/{uuid}/algo-input
// → Direct VideoVariantMetadata: {uri, width, height, fps, …}
type graceAlgoVariantResp struct {
	URI    string   `json:"uri"`
	Width  *int     `json:"width,omitempty"`
	Height *int     `json:"height,omitempty"`
	FPS    *float64 `json:"fps,omitempty"`
}

// ---------------------------------------------------------------------------

func (g *GraceResolver) Resolve(ctx context.Context, id, env string) (string, error) {
	// Parse id: "video_id" or "video_id/sub_path"
	videoID, subPath, _ := strings.Cut(id, "/")
	baseURL, username, password := g.devURL, g.devUsername, g.devPassword // pragma: allowlist secret
	if env == "prod" {
		baseURL, username, password = g.prodURL, g.prodUsername, g.prodPassword // pragma: allowlist secret
	}
	if baseURL == "" || username == "" || password == "" {
		return "", fmt.Errorf("grace resolver: %s credentials not configured", env)
	}

	// Build the static Basic Auth header (no token endpoint required).
	authHeader := basicAuthHeader(username, password)

	// Resolve based on sub-path
	var gcsURI string
	var err error
	switch subPath {
	case "", "algo_input":
		gcsURI, err = g.getAlgoInput(ctx, baseURL, authHeader, videoID)
		if err != nil {
			return "", fmt.Errorf("grace get_algo_input: %w", err)
		}
	case "raw":
		gcsURI, err = g.getRawVideo(ctx, baseURL, authHeader, videoID)
		if err != nil {
			return "", fmt.Errorf("grace get_raw_video: %w", err)
		}
	default:
		return "", fmt.Errorf("unknown grace sub-path: %s", subPath)
	}

	gcsPath := strings.TrimPrefix(gcsURI, "gs://")
	return gcsPath, nil
}

// basicAuthHeader builds a static Basic Authorization header value.
func basicAuthHeader(username, password string) string {
	raw := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	return "Basic " + encoded
}

// getRawVideo calls GET /grace/videos/{id} and extracts storage_meta.gcs.video.
func (g *GraceResolver) getRawVideo(ctx context.Context, baseURL, authHeader, videoID string) (string, error) {
	u := fmt.Sprintf("%s/grace/videos/%s", strings.TrimRight(baseURL, "/"), videoID)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("grace get_video returned %d: %s", resp.StatusCode, string(b))
	}

	// The response is the full Video JSON model.
	// storage_meta.gcs.video is a plain GCS string.
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", err
	}

	// Navigate the nested structure using raw messages to extract just the video field.
	var nested struct {
		StorageMeta *struct {
			Gcs *struct {
				Video *string `json:"video"`
			} `json:"gcs"`
		} `json:"storage_meta"`
	}
	if err := json.Unmarshal(raw, &nested); err != nil {
		return "", err
	}
	if nested.StorageMeta != nil && nested.StorageMeta.Gcs != nil && nested.StorageMeta.Gcs.Video != nil {
		return *nested.StorageMeta.Gcs.Video, nil
	}
	return "", fmt.Errorf("get_video %s: no storage_meta.gcs.video field", videoID)
}

// getAlgoInput calls GET /grace/videos/{id}/algo-input and extracts the URI.
func (g *GraceResolver) getAlgoInput(ctx context.Context, baseURL, authHeader, videoID string) (string, error) {
	u := fmt.Sprintf("%s/grace/videos/%s/algo-input", strings.TrimRight(baseURL, "/"), videoID)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("grace algo-input returned %d: %s", resp.StatusCode, string(b))
	}

	var result graceAlgoVariantResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.URI, nil
}

// ---------------------------------------------------------------------------
// Resolver registry
// ---------------------------------------------------------------------------

var defaultResolvers = map[string]SourceResolver{}

// RegisterResolver registers a source resolver for the given source name.
func RegisterResolver(source string, r SourceResolver) {
	defaultResolvers[source] = r
}

// GetResolver returns the resolver for the given source.
func GetResolver(source string) (SourceResolver, bool) {
	r, ok := defaultResolvers[source]
	return r, ok
}
