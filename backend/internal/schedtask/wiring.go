package schedtask

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/schedtask/source"
)

// EnvSecretResolver resolves `secret_ref` to the plaintext credential by
// reading it from an env var of the same name.
//
// Rationale (see CYB-3744 decisions.md): grace-sync and the rest of the
// backend already receive credentials via Cloud Run --set-secrets, which mounts
// a Secret Manager version at an env var name. Reusing that pipeline avoids
// adding a Secret Manager client dependency + IAM binding for one use case; a
// future "real Secret Manager" resolver can drop in behind the same interface.
//
// The env var may hold a plain string OR a JSON blob (Cloud Run mounts the
// whole grace-api-dev secret as one env var); parseSecretValue extracts a
// nested key if the ref carries a #key suffix, so one env var can supply
// multiple credentials.
//
//	secret_ref = "GRACE_PASSWORD"              → return $GRACE_PASSWORD verbatim              // pragma: allowlist secret
//	secret_ref = "GRACE_PASSWORD#AUTH_PASSWORD" → parse $GRACE_PASSWORD as JSON,              // pragma: allowlist secret
//	                                              return .AUTH_PASSWORD
type EnvSecretResolver struct{}

func NewEnvSecretResolver() *EnvSecretResolver { return &EnvSecretResolver{} }

// jsonKeyRE matches a `"KEY": "value"` or unquoted `KEY: "value"` pair, since
// GCP Secret Manager sometimes stores JSON-ish blobs with unquoted keys (same
// tolerance as grace-sync's parsePassword).
var jsonKeyRE = regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]+)"|([A-Za-z_][A-Za-z0-9_]*)\s*:\s*"([^"]+)"`)

func (EnvSecretResolver) Resolve(_ context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty secret_ref")
	}
	envName := ref
	subKey := ""
	if i := strings.Index(ref, "#"); i > 0 {
		envName = ref[:i]
		subKey = ref[i+1:]
	}
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return "", fmt.Errorf("env %s is empty", envName)
	}
	if subKey == "" {
		return extractStringValue(raw), nil
	}
	// Look inside a JSON (or JSON-ish) blob for the requested key.
	if strings.HasPrefix(raw, "{") {
		var m map[string]string
		if err := json.Unmarshal([]byte(raw), &m); err == nil {
			if v, ok := m[subKey]; ok && v != "" {
				return v, nil
			}
		}
		// Tolerate unquoted keys (grace-sync's shape).
		for _, mm := range jsonKeyRE.FindAllStringSubmatch(raw, -1) {
			key, val := mm[1], mm[2]
			if key == "" {
				key, val = mm[3], mm[4]
			}
			if key == subKey && val != "" {
				return val, nil
			}
		}
	}
	return "", fmt.Errorf("env %s missing sub-key %q", envName, subKey)
}

// extractStringValue returns raw as-is unless it's a JSON object with a single
// AUTH_PASSWORD key (grace-sync's back-compat shape): then it returns that.
// This keeps existing Grace secrets working without a sub-key ref.
func extractStringValue(raw string) string {
	if !strings.HasPrefix(raw, "{") {
		return raw
	}
	var m struct {
		AuthPassword string `json:"AUTH_PASSWORD"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err == nil && m.AuthPassword != "" {
		return m.AuthPassword
	}
	return raw
}

// ─── source factory ───────────────────────────────────────────

// SourceFactoryFn is a function adapter for the SourceFactory interface, so
// wiring code doesn't have to declare a struct.
type SourceFactoryFn func(sourceType string, sourceConfig json.RawMessage) (source.AssetSource, error)

func (f SourceFactoryFn) Build(t string, cfg json.RawMessage) (source.AssetSource, error) {
	return f(t, cfg)
}

// NewDefaultSourceFactory returns the v1 source factory: only "rest" is
// supported, resolving to source.NewREST with the given SecretResolver. Add
// more source types here (or wrap this factory) when a real second one
// appears.
func NewDefaultSourceFactory(resolver source.SecretResolver) SourceFactory {
	return SourceFactoryFn(func(t string, raw json.RawMessage) (source.AssetSource, error) {
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "rest":
			var cfg source.RestConfig
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &cfg); err != nil {
					return nil, fmt.Errorf("decode rest source config: %w", err)
				}
			}
			return source.NewREST(cfg, resolver), nil
		default:
			return nil, fmt.Errorf("unsupported source_type %q", t)
		}
	})
}

// ─── batch creator adapter ────────────────────────────────────

// BatchCreatorFn adapts pipeline.Usecase.CreateBatchJob(...) to
// schedtask.BatchCreator.CreateBatch(...). Wire in cmd/server:
//
//	sc := schedtask.BatchCreatorFn(func(ctx, tmpl, name, ids, target, ver, owner string) (string, error) {
//	    job, err := puc.CreateBatchJob(ctx, tmpl, name, ids, target, ver, 0, owner)
//	    if err != nil || job == nil { return "", err }
//	    return job.ID, nil
//	})
type BatchCreatorFn func(ctx context.Context, templateID, name string, assetIDs []string, targetID string, templateVersion int, owner string) (string, error)

func (f BatchCreatorFn) CreateBatch(ctx context.Context, tmpl, name string, ids []string, target string, ver int, owner string) (string, error) {
	return f(ctx, tmpl, name, ids, target, ver, owner)
}
