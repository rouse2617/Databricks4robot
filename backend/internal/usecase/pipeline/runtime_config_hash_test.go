package pipeline

import (
	"context"
	"strings"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ── CYB-3680: canonical content hashing ──────────────────────────────────────

func TestRuntimeConfigProjectionHashCanonical(t *testing.T) {
	multi := RuntimeConfigProjection{Files: map[string]string{
		"a.yaml": "a: 1\n", "b.yaml": "b: 2\n",
	}}
	// Same logical content, different construction: single-file form folded in.
	folded := RuntimeConfigProjection{
		FileName: "a.yaml", Content: "a: 1\n",
		Files: map[string]string{"b.yaml": "b: 2\n"},
	}
	if runtimeConfigProjectionHash(multi) != runtimeConfigProjectionHash(folded) {
		t.Fatal("equivalent projections must hash identically")
	}

	// Content sensitivity: any byte change changes the hash.
	changed := RuntimeConfigProjection{Files: map[string]string{
		"a.yaml": "a: 1\n", "b.yaml": "b: 3\n",
	}}
	if runtimeConfigProjectionHash(multi) == runtimeConfigProjectionHash(changed) {
		t.Fatal("different content must hash differently")
	}

	// Name sensitivity: same bytes under a different filename is a different config.
	renamed := RuntimeConfigProjection{Files: map[string]string{
		"a.yaml": "a: 1\n", "c.yaml": "b: 2\n",
	}}
	if runtimeConfigProjectionHash(multi) == runtimeConfigProjectionHash(renamed) {
		t.Fatal("different filenames must hash differently")
	}

	// Boundary ambiguity: {"ab":"c"} vs {"a":"bc"} must differ (length prefixes).
	x := RuntimeConfigProjection{Files: map[string]string{"ab": "c"}}
	y := RuntimeConfigProjection{Files: map[string]string{"a": "bc"}}
	if runtimeConfigProjectionHash(x) == runtimeConfigProjectionHash(y) {
		t.Fatal("field-boundary ambiguity: length prefixing failed")
	}

	// Determinism across invocations (map order must never leak).
	h := runtimeConfigProjectionHash(multi)
	for i := 0; i < 20; i++ {
		if runtimeConfigProjectionHash(multi) != h {
			t.Fatal("hash must be deterministic")
		}
	}
}

func TestRuntimeConfigVolumeNameForContent(t *testing.T) {
	h := runtimeConfigProjectionHash(RuntimeConfigProjection{Files: map[string]string{"f": "x"}})
	name := runtimeConfigVolumeNameForContent(h)
	if !strings.HasPrefix(name, "runtime-config-") {
		t.Fatalf("name %q missing prefix", name)
	}
	if len(name) != len("runtime-config-")+32 {
		t.Fatalf("name length = %d, want prefix+32", len(name))
	}
	if len(name) > 63 {
		t.Fatalf("name %q exceeds DNS label limit", name)
	}
	// Short input passes through untruncated.
	if got := runtimeConfigVolumeNameForContent("abc"); got != "runtime-config-abc" {
		t.Fatalf("short hash name = %q", got)
	}
}

// ── CYB-3680: blob persistence ───────────────────────────────────────────────

type fakeBlobStore struct {
	hashes []string
	files  []map[string]string
	err    error
}

func (f *fakeBlobStore) Upsert(_ context.Context, hash string, files map[string]string) error {
	f.hashes = append(f.hashes, hash)
	f.files = append(f.files, files)
	return f.err
}

func TestPersistRuntimeConfigBlob(t *testing.T) {
	ctx := context.Background()

	t.Run("nil store is a no-op", func(t *testing.T) {
		(&Usecase{}).persistRuntimeConfigBlob(ctx, &RuntimeConfigProjection{ContentHash: "h"})
	})
	t.Run("nil projection and empty hash are no-ops", func(t *testing.T) {
		s := &fakeBlobStore{}
		uc := &Usecase{}
		uc.SetRuntimeConfigBlobStore(s)
		uc.persistRuntimeConfigBlob(ctx, nil)
		uc.persistRuntimeConfigBlob(ctx, &RuntimeConfigProjection{})
		if len(s.hashes) != 0 {
			t.Fatalf("upserts = %v, want none", s.hashes)
		}
	})
	t.Run("folds FileName into files and upserts by hash", func(t *testing.T) {
		s := &fakeBlobStore{}
		uc := &Usecase{}
		uc.SetRuntimeConfigBlobStore(s)
		uc.persistRuntimeConfigBlob(ctx, &RuntimeConfigProjection{
			ContentHash: "h1",
			FileName:    "main.yaml", Content: "m: 1\n",
			Files: map[string]string{"extra.yaml": "e: 1\n"},
		})
		if len(s.hashes) != 1 || s.hashes[0] != "h1" {
			t.Fatalf("hashes = %v, want [h1]", s.hashes)
		}
		if s.files[0]["main.yaml"] != "m: 1\n" || s.files[0]["extra.yaml"] != "e: 1\n" {
			t.Fatalf("files = %#v", s.files[0])
		}
	})
	t.Run("upsert error is tolerated", func(t *testing.T) {
		s := &fakeBlobStore{err: context.DeadlineExceeded}
		uc := &Usecase{}
		uc.SetRuntimeConfigBlobStore(s)
		uc.persistRuntimeConfigBlob(ctx, &RuntimeConfigProjection{ContentHash: "h2", Files: map[string]string{"f": "x"}})
		if len(s.hashes) != 1 {
			t.Fatal("upsert must have been attempted")
		}
	})
}

// ── CYB-3680: ensure-before-submit ordering ──────────────────────────────────

type failingRuntimeConfigStore struct{ err error }

func (f *failingRuntimeConfigStore) Create(context.Context, string, string, RuntimeConfigProjection, *RuntimeConfigOwnerReference) (string, error) {
	return "", f.err
}

// A failing ConfigMap ensure must abort Deploy BEFORE any Workflow is
// created — proving the CYB-3680 ordering (CM first) and killing the class
// of "Workflow exists, CM never will" stuck pods.
func TestDeploy_RuntimeConfigEnsureFailureAbortsBeforeWorkflow(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "test-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id":        "step-1",
				"component": map[string]interface{}{"name": "test", "image": "busybox"},
			},
		},
		"edges": []interface{}{},
	}
	uc := newUsecase(newMockAssetRepo())
	created := false
	uc.wfClient = &mockWorkflowClient{createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
		created = true
		return nil
	}}
	uc.pipelineConfigRepo = &mockPipelineConfigRepo{
		configs: map[string]*models.PipelineConfig{"cfg-1": {ID: "cfg-1", Name: "detector.yaml"}},
		versions: map[string]*models.PipelineConfigVersion{
			"cfg-1:2": {ConfigID: "cfg-1", Version: 2, Content: "threshold: 0.8\n"},
		},
	}
	uc.runtimeConfigStore = &failingRuntimeConfigStore{err: context.DeadlineExceeded}

	_, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{
		ConfigSelection: &RuntimeConfigSelection{
			Mode: "saved", ConfigID: "cfg-1", Version: 2,
			FileName: "detector.yaml", MountPath: "/workspace/configs", TargetFilename: "effective.yaml",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "ensure runtime config") {
		t.Fatalf("err = %v, want ensure runtime config failure", err)
	}
	if created {
		t.Fatal("workflow must NOT be created when the ConfigMap ensure fails (CM-first ordering)")
	}
}

type failingStoreFactory struct{}

func (failingStoreFactory) ForTarget(context.Context, *models.ExecutionTarget) (RuntimeConfigStore, error) {
	return nil, context.DeadlineExceeded
}

// A store-resolution failure (per-cluster factory error) aborts before any
// Workflow too.
func TestDeploy_RuntimeConfigStoreResolveFailureAborts(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "test-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id":        "step-1",
				"component": map[string]interface{}{"name": "test", "image": "busybox"},
			},
		},
		"edges": []interface{}{},
	}
	uc := newUsecase(newMockAssetRepo())
	created := false
	uc.wfClient = &mockWorkflowClient{createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
		created = true
		return nil
	}}
	uc.pipelineConfigRepo = &mockPipelineConfigRepo{
		configs: map[string]*models.PipelineConfig{"cfg-1": {ID: "cfg-1", Name: "detector.yaml"}},
		versions: map[string]*models.PipelineConfigVersion{
			"cfg-1:2": {ConfigID: "cfg-1", Version: 2, Content: "threshold: 0.8\n"},
		},
	}
	uc.runtimeConfigStoreFactory = failingStoreFactory{}

	_, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{
		ConfigSelection: &RuntimeConfigSelection{
			Mode: "saved", ConfigID: "cfg-1", Version: 2,
			FileName: "detector.yaml", MountPath: "/workspace/configs", TargetFilename: "effective.yaml",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "resolve runtime config store") {
		t.Fatalf("err = %v, want store resolve failure", err)
	}
	if created {
		t.Fatal("workflow must NOT be created when store resolution fails")
	}
}
