package storage

import (
	"context"
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
// Grace API resolver (temporary — will be replaced)
// ---------------------------------------------------------------------------

type GraceResolver struct {
	devURL, devUsername, devPassword     string
	prodURL, prodUsername, prodPassword string
}

type graceAuthResp struct {
	Token string `json:"token"`
}

type graceVideoResp struct {
	StorageMeta *graceStorageMeta `json:"storage_meta"`
}

type graceStorageMeta struct {
	Gcs *graceGcsMeta `json:"gcs"`
}

type graceGcsMeta struct {
	Video       *graceGcsRef  `json:"video"`
	AlgoInputs  *graceGcsAlgo `json:"algo_inputs"`
}

type graceGcsRef struct {
	URI string `json:"uri"`
}

type graceGcsAlgo struct {
	Mcap *graceGcsRef `json:"mcap"`
}

type graceAlgoInputResp struct {
	ResultRef *graceAlgoResultRef `json:"result_ref"`
}

type graceAlgoResultRef struct {
	GcsURI string `json:"gcs_uri"`
}

func (g *GraceResolver) Resolve(ctx context.Context, id, env string) (string, error) {
	// Parse id: "video_id" or "video_id/sub_path"
	videoID, subPath, _ := strings.Cut(id, "/")
	baseURL, username, password := g.devURL, g.devUsername, g.devPassword
	if env == "prod" {
		baseURL, username, password = g.prodURL, g.prodUsername, g.prodPassword
	}
	if baseURL == "" || username == "" || password == "" {
		return "", fmt.Errorf("grace resolver: %s credentials not configured", env)
	}

	// 1. Authenticate
	token, err := g.authenticate(ctx, baseURL, username, password)
	if err != nil {
		return "", fmt.Errorf("grace auth: %w", err)
	}

	// 2. Resolve based on sub-path
	var gcsURI string
	switch subPath {
	case "", "algo_input":
		gcsURI, err = g.getAlgoInput(ctx, baseURL, token, videoID)
		if err != nil {
			return "", fmt.Errorf("grace get_algo_input: %w", err)
		}
	case "raw":
		gcsURI, err = g.getRawVideo(ctx, baseURL, token, videoID)
		if err != nil {
			return "", fmt.Errorf("grace get_raw_video: %w", err)
		}
	default:
		return "", fmt.Errorf("unknown grace sub-path: %s", subPath)
	}
	if err != nil {
		return "", fmt.Errorf("grace resolve: %w", err)
	}

	gcsPath := strings.TrimPrefix(gcsURI, "gs://")
	return gcsPath, nil
}

func (g *GraceResolver) authenticate(ctx context.Context, baseURL, username, password string) (string, error) {
	url := fmt.Sprintf("%s/auth/token", strings.TrimRight(baseURL, "/"))
	body := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("grace auth returned %d: %s", resp.StatusCode, string(b))
	}

	var authResp graceAuthResp
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}
	return authResp.Token, nil
}

func (g *GraceResolver) getRawVideo(ctx context.Context, baseURL, token, videoID string) (string, error) {
	url := fmt.Sprintf("%s/video/%s", strings.TrimRight(baseURL, "/"), videoID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("grace get_video returned %d: %s", resp.StatusCode, string(b))
	}

	var result graceVideoResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.StorageMeta != nil && result.StorageMeta.Gcs != nil && result.StorageMeta.Gcs.Video != nil {
		return result.StorageMeta.Gcs.Video.URI, nil
	}
	return "", fmt.Errorf("get_video %s: no storage_meta.gcs.video.uri field", videoID)
}

func (g *GraceResolver) getAlgoInput(ctx context.Context, baseURL, token, videoID string) (string, error) {
	url := fmt.Sprintf("%s/video/%s/algo-input", strings.TrimRight(baseURL, "/"), videoID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("grace algo-input returned %d: %s", resp.StatusCode, string(b))
	}

	var result graceAlgoInputResp
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
