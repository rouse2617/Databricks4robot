package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ── Extended mock helpers ─────────────────────────────────────────────────

type mockEventRepo struct {
	appended []repository.AssetEventAppendInput
	events   map[string][]*models.AssetEvent
}

func (m *mockEventRepo) Append(_ context.Context, in repository.AssetEventAppendInput) error {
	m.appended = append(m.appended, in)
	return nil
}
func (m *mockEventRepo) ListByAsset(_ context.Context, assetID string, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return m.events[assetID], nil
}
func (m *mockEventRepo) ListPending(_ context.Context, _ int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) ListPendingSafe(_ context.Context, _ time.Duration, _ int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) ListVersionPromotedByLogical(_ context.Context, _ string) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) ListGlobal(_ context.Context, _ repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) MarkPublished(_ context.Context, _ []int64) error      { return nil }
func (m *mockEventRepo) MarkFailed(_ context.Context, _ int64, _ string) error { return nil }
func (m *mockEventRepo) CountPending(_ context.Context) (int64, error)         { return 0, nil }
func (m *mockEventRepo) CountPendingClaimable(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}
func (m *mockEventRepo) CountProcessing(_ context.Context) (int64, error)    { return 0, nil }
func (m *mockEventRepo) OldestPendingAge(_ context.Context) (float64, error) { return 0, nil }
func (m *mockEventRepo) PublishStateCounts(_ context.Context) (map[string]int64, error) {
	return nil, nil
}
func (m *mockEventRepo) ListBetweenSeq(_ context.Context, _, _ int64, _ int) ([]*models.AssetEvent, error) {
	return nil, nil
}

type mockRelationWriter struct {
	relations []relationRecord
}

type mockLogicalAssetRepo struct {
	inserted []*models.LogicalAsset
}

func (m *mockLogicalAssetRepo) Get(_ context.Context, _ string) (*models.LogicalAsset, error) {
	return nil, repository.ErrLogicalAssetNotFound
}
func (m *mockLogicalAssetRepo) Insert(_ context.Context, la *models.LogicalAsset) error {
	m.inserted = append(m.inserted, la)
	return nil
}
func (m *mockLogicalAssetRepo) BumpRevision(_ context.Context, _ string, _ int64) error { return nil }
func (m *mockLogicalAssetRepo) MaxRevision(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockLogicalAssetRepo) CurrentAssetID(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (m *mockLogicalAssetRepo) ClearCurrentForLogical(_ context.Context, _ string) error { return nil }

type relationRecord struct {
	srcAssetID    string
	dstAssetID    string
	relationType  string
	relationRunID string
}

func (m *mockRelationWriter) InsertRevisionOf(_ context.Context, _, _, _ string) error { return nil }
func (m *mockRelationWriter) InsertRelation(_ context.Context, srcAssetID, dstAssetID, relationType, relationRunID string) error {
	m.relations = append(m.relations, relationRecord{
		srcAssetID:    srcAssetID,
		dstAssetID:    dstAssetID,
		relationType:  relationType,
		relationRunID: relationRunID,
	})
	return nil
}

// ── SaveTemplate ──────────────────────────────────────────────────────────

func TestSaveTemplate(t *testing.T) {
	ctx := context.Background()

	t.Run("saves template with auto-incremented version", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipe := map[string]interface{}{
			"name": "test-pipe",
			"nodes": []interface{}{
				map[string]interface{}{"id": "step-1", "component": map[string]interface{}{"name": "a", "image": "img"}},
				map[string]interface{}{"id": "step-2", "component": map[string]interface{}{"name": "b", "image": "img"}},
			},
			"edges": []interface{}{},
		}

		tmpl, err := uc.SaveTemplate(ctx, "test-pipe", pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tmpl == nil {
			t.Fatal("expected template, got nil")
		}
		if tmpl.Name != "test-pipe" {
			t.Fatalf("expected name 'test-pipe', got %q", tmpl.Name)
		}
		if tmpl.Version != 1 {
			t.Fatalf("expected version 1, got %d", tmpl.Version)
		}
		if tmpl.NodeCount != 2 {
			t.Fatalf("expected nodeCount 2, got %d", tmpl.NodeCount)
		}
		if tmpl.ID == "" {
			t.Fatal("expected non-empty ID")
		}
	})

	t.Run("increments version on subsequent saves", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipe := map[string]interface{}{
			"name": "multi-ver",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}

		t1, err := uc.SaveTemplate(ctx, "multi-ver", pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("first save: %v", err)
		}
		if t1.Version != 1 {
			t.Fatalf("expected version 1, got %d", t1.Version)
		}

		t2, err := uc.SaveTemplate(ctx, "multi-ver", pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("second save: %v", err)
		}
		if t2.Version != 2 {
			t.Fatalf("expected version 2, got %d", t2.Version)
		}
	})

	t.Run("zero nodes is valid", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipe := map[string]interface{}{
			"name":  "empty",
			"nodes": []interface{}{},
			"edges": []interface{}{},
		}

		tmpl, err := uc.SaveTemplate(ctx, "empty", pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tmpl.NodeCount != 0 {
			t.Fatalf("expected nodeCount 0, got %d", tmpl.NodeCount)
		}
	})

	t.Run("rejects duplicate fan-in target input", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipe := map[string]interface{}{
			"name": "bad-fanin",
			"nodes": []interface{}{
				map[string]interface{}{
					"id": "a",
					"component": map[string]interface{}{
						"name":    "a",
						"image":   "busybox",
						"command": []interface{}{"sh", "-c"},
						"args": []interface{}{
							map[string]interface{}{"name": "script", "value": "echo a > /tmp/outputs/output"},
						},
					},
					"outputs": []interface{}{map[string]interface{}{"name": "output", "type": "string"}},
				},
				map[string]interface{}{
					"id": "b",
					"component": map[string]interface{}{
						"name":    "b",
						"image":   "busybox",
						"command": []interface{}{"sh", "-c"},
						"args": []interface{}{
							map[string]interface{}{"name": "script", "value": "echo b > /tmp/outputs/output"},
						},
					},
					"outputs": []interface{}{map[string]interface{}{"name": "output", "type": "string"}},
				},
				map[string]interface{}{
					"id": "join",
					"component": map[string]interface{}{
						"name":    "join",
						"image":   "busybox",
						"command": []interface{}{"sh", "-c"},
						"args": []interface{}{
							map[string]interface{}{"name": "script", "value": "echo join"},
						},
					},
					"inputs": []interface{}{map[string]interface{}{"name": "input", "type": "string"}},
				},
			},
			"edges": []interface{}{
				map[string]interface{}{"source": "a.output", "target": "join.input"},
				map[string]interface{}{"source": "b.output", "target": "join.input"},
			},
		}

		_, err := uc.SaveTemplate(ctx, "bad-fanin", pipe, "dev", "legacy")
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("expected ErrInvalidArgument, got %v", err)
		}
	})

	t.Run("normalizes duplicated shell args before saving", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipe := map[string]interface{}{
			"name": "normalize-shell",
			"nodes": []interface{}{
				map[string]interface{}{
					"id": "step-1",
					"component": map[string]interface{}{
						"name":    "a",
						"image":   "busybox",
						"command": []interface{}{"sh", "-c"},
						"args": []interface{}{
							map[string]interface{}{"name": "sh", "value": "sh"},
							map[string]interface{}{"name": "-c", "value": "-c"},
							map[string]interface{}{"name": "script", "value": "echo ok"},
						},
					},
				},
			},
			"edges": []interface{}{},
		}

		tmpl, err := uc.SaveTemplate(ctx, "normalize-shell", pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		nodes, ok := tmpl.Pipeline["nodes"].([]interface{})
		if !ok || len(nodes) != 1 {
			t.Fatalf("nodes = %#v", tmpl.Pipeline["nodes"])
		}
		component := nodes[0].(map[string]interface{})["component"].(map[string]interface{})
		args := component["args"].([]interface{})
		if len(args) != 1 {
			t.Fatalf("args = %#v, want one script arg", args)
		}
		value := args[0].(map[string]interface{})["value"]
		if value != "echo ok" {
			t.Fatalf("arg value = %#v, want echo ok", value)
		}
	})
}

// ── Template CRUD ─────────────────────────────────────────────────────────

func TestTemplateCRUD(t *testing.T) {
	ctx := context.Background()
	uc := newUsecase(newMockAssetRepo())

	// Initially empty.
	list, err := uc.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d items", len(list))
	}

	// Save a template.
	pipe := map[string]interface{}{
		"name": "crud-test", "nodes": []interface{}{}, "edges": []interface{}{},
	}
	tmpl, err := uc.SaveTemplate(ctx, "crud-test", pipe, "dev", "legacy")
	if err != nil {
		t.Fatalf("SaveTemplate: %v", err)
	}

	// GetTemplate by ID.
	got, err := uc.GetTemplate(ctx, tmpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if got == nil {
		t.Fatal("expected template, got nil")
	}
	if got.ID != tmpl.ID {
		t.Fatalf("expected ID %q, got %q", tmpl.ID, got.ID)
	}

	// DeleteTemplate.
	if err := uc.DeleteTemplate(ctx, tmpl.ID, ""); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}

	// Verify deleted.
	deleted, err := uc.GetTemplate(ctx, tmpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate after delete: %v", err)
	}
	if deleted != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestDeleteTemplateRemovesAssociatedDeployments(t *testing.T) {
	ctx := context.Background()
	templateRepo := &mockTemplateRepo{byID: make(map[string]*models.PipelineTemplate)}
	deploymentRepo := &mockDeploymentRepo{}
	uc := &Usecase{
		templateRepo:   templateRepo,
		deploymentRepo: deploymentRepo,
		assetRepo:      newMockAssetRepo(),
		wfClient:       &mockWorkflowClient{},
	}
	pipe := map[string]interface{}{
		"name": "delete-with-deployments",
		"nodes": []interface{}{
			map[string]interface{}{"id": "step-1", "component": map[string]interface{}{"name": "a", "image": "img"}},
		},
		"edges": []interface{}{},
	}
	tmpl, err := uc.SaveTemplate(ctx, "delete-with-deployments", pipe, "dev", "legacy")
	if err != nil {
		t.Fatalf("SaveTemplate: %v", err)
	}
	dep, err := uc.DeployByTemplateID(ctx, tmpl.ID, "", nil)
	if err != nil {
		t.Fatalf("DeployByTemplateID: %v", err)
	}
	if dep.TemplateID == nil || *dep.TemplateID != tmpl.ID {
		t.Fatalf("expected deployment templateID %q, got %v", tmpl.ID, dep.TemplateID)
	}

	if err := uc.DeleteTemplate(ctx, tmpl.ID, ""); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if got, err := uc.GetTemplate(ctx, tmpl.ID); err != nil || got != nil {
		t.Fatalf("expected deleted template, got template=%v err=%v", got, err)
	}
	if got, err := deploymentRepo.FindByID(ctx, dep.ID); err != nil || got != nil {
		t.Fatalf("expected deleted deployment, got deployment=%v err=%v", got, err)
	}
	if len(deploymentRepo.saved) != 0 {
		t.Fatalf("expected no saved deployments, got %d", len(deploymentRepo.saved))
	}
}

// ── DeployByTemplateID ────────────────────────────────────────────────────

func TestDeployByTemplateID(t *testing.T) {
	ctx := context.Background()

	t.Run("deploys existing template", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "tmpl-deploy",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		tmpl, err := uc.SaveTemplate(ctx, "tmpl-deploy", pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("SaveTemplate: %v", err)
		}

		dep, err := uc.DeployByTemplateID(ctx, tmpl.ID, "", nil)
		if err != nil {
			t.Fatalf("DeployByTemplateID: %v", err)
		}
		if dep == nil {
			t.Fatal("expected deployment, got nil")
		}
		if dep.PipelineName != "tmpl-deploy" {
			t.Fatalf("expected pipeline name 'tmpl-deploy', got %q", dep.PipelineName)
		}
		if dep.TemplateID == nil || *dep.TemplateID != tmpl.ID {
			t.Fatalf("expected templateID %q, got %v", tmpl.ID, dep.TemplateID)
		}
	})

	t.Run("returns error for missing template", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		_, err := uc.DeployByTemplateID(ctx, "nonexistent", "", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrTemplateNotFound) {
			t.Fatalf("expected ErrTemplateNotFound, got: %v", err)
		}
	})

	t.Run("deploys with custom name override", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "original",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		tmpl, err := uc.SaveTemplate(ctx, "original", pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("SaveTemplate: %v", err)
		}

		dep, err := uc.DeployByTemplateID(ctx, tmpl.ID, "custom-name", nil)
		if err != nil {
			t.Fatalf("DeployByTemplateID: %v", err)
		}
		if dep.PipelineName != "custom-name" {
			t.Fatalf("expected pipeline name 'custom-name', got %q", dep.PipelineName)
		}
	})

	t.Run("deploys requested saved version snapshot", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		v1Pipe := map[string]interface{}{
			"name": "versioned",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		v1, err := uc.SaveTemplate(ctx, "versioned", v1Pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("SaveTemplate v1: %v", err)
		}
		v2Pipe := map[string]interface{}{
			"name": "versioned",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
				map[string]interface{}{"id": "s2", "component": map[string]interface{}{"name": "b", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		v2, err := uc.SaveTemplate(ctx, "versioned", v2Pipe, "dev", "legacy")
		if err != nil {
			t.Fatalf("SaveTemplate v2: %v", err)
		}

		dep, err := uc.DeployByTemplateID(ctx, v2.ID, "", nil, DeployOptions{TemplateVersion: 1})
		if err != nil {
			t.Fatalf("DeployByTemplateID: %v", err)
		}
		if dep.TemplateID == nil || *dep.TemplateID != v1.ID {
			t.Fatalf("expected v1 templateID %q, got %v", v1.ID, dep.TemplateID)
		}
		if dep.TemplateVersion == nil || *dep.TemplateVersion != 1 {
			t.Fatalf("expected template version 1, got %v", dep.TemplateVersion)
		}
		if dep.NodeCount != 1 {
			t.Fatalf("expected v1 node count 1, got %d", dep.NodeCount)
		}
	})
}

// ── SaveFromDeployment ────────────────────────────────────────────────────

func TestSaveFromDeployment(t *testing.T) {
	ctx := context.Background()

	t.Run("creates template from deployment", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "src", "nodes": []interface{}{}, "edges": []interface{}{},
		}
		dep, err := uc.Deploy(ctx, pipe, "", nil)
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}

		tmpl, err := uc.SaveFromDeployment(ctx, dep.ID, "")
		if err != nil {
			t.Fatalf("SaveFromDeployment: %v", err)
		}
		if tmpl == nil {
			t.Fatal("expected template, got nil")
		}
		if tmpl.Name != "src-from-deployment" {
			t.Fatalf("expected 'src-from-deployment', got %q", tmpl.Name)
		}
	})

	t.Run("uses custom template name", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "src2", "nodes": []interface{}{}, "edges": []interface{}{},
		}
		dep, err := uc.Deploy(ctx, pipe, "", nil)
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}

		tmpl, err := uc.SaveFromDeployment(ctx, dep.ID, "my-template")
		if err != nil {
			t.Fatalf("SaveFromDeployment: %v", err)
		}
		if tmpl.Name != "my-template" {
			t.Fatalf("expected 'my-template', got %q", tmpl.Name)
		}
	})

	t.Run("returns error for missing deployment", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		_, err := uc.SaveFromDeployment(ctx, "nonexistent", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDeploymentNotFound) {
			t.Fatalf("expected ErrDeploymentNotFound, got: %v", err)
		}
	})
}

// ── ListDeployments ───────────────────────────────────────────────────────

func TestListDeployments(t *testing.T) {
	ctx := context.Background()

	t.Run("returns non-empty list", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "test", "nodes": []interface{}{}, "edges": []interface{}{},
		}
		_, err := uc.Deploy(ctx, pipe, "", nil)
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}

		list, err := uc.ListDeployments(ctx)
		if err != nil {
			t.Fatalf("ListDeployments: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 deployment, got %d", len(list))
		}
		if list[0].PipelineName != "test" {
			t.Fatalf("expected pipeline name 'test', got %q", list[0].PipelineName)
		}
	})

	t.Run("returns empty list when no deployments", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		list, err := uc.ListDeployments(ctx)
		if err != nil {
			t.Fatalf("ListDeployments: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected empty list, got %d", len(list))
		}
	})
}

func TestBatchCreateRunsByTemplateID(t *testing.T) {
	ctx := context.Background()
	repo := newMockAssetRepo()
	repo.assets["asset-a"] = &models.Asset{AssetID: "asset-a", AssetType: "dataset"}
	repo.assets["asset-b"] = &models.Asset{AssetID: "asset-b", AssetType: "dataset"}
	uc := newUsecase(repo)
	pipe := map[string]interface{}{
		"name": "batch-tmpl",
		"nodes": []interface{}{
			map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
		},
		"edges": []interface{}{},
	}
	tmpl, err := uc.SaveTemplate(ctx, "batch-tmpl", pipe, "dev", "tester")
	if err != nil {
		t.Fatalf("SaveTemplate: %v", err)
	}

	result, err := uc.BatchCreateRunsByTemplateID(ctx, tmpl.ID, "", []string{"asset-a", "asset-b"})
	if err != nil {
		t.Fatalf("BatchCreateRunsByTemplateID: %v", err)
	}
	if result.BatchID == "" {
		t.Fatal("expected batch id")
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(result.Items))
	}
	for _, run := range result.Items {
		if run.BatchRunID != result.BatchID {
			t.Fatalf("expected batchRunId %q on run %s, got %q", result.BatchID, run.ID, run.BatchRunID)
		}
		if len(run.AssetIDs) != 1 {
			t.Fatalf("expected 1 asset per run, got %v", run.AssetIDs)
		}
	}
	if result.Items[0].WorkflowName == result.Items[1].WorkflowName {
		t.Fatal("expected distinct workflow names per asset fan-out")
	}
}

// ── GetDeployment / DeleteDeployment ──────────────────────────────────────

func TestDeploymentCRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("get returns saved deployment", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "get-test", "nodes": []interface{}{}, "edges": []interface{}{},
		}
		dep, err := uc.Deploy(ctx, pipe, "", nil)
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}

		got, err := uc.GetDeployment(ctx, dep.ID)
		if err != nil {
			t.Fatalf("GetDeployment: %v", err)
		}
		if got == nil {
			t.Fatal("expected deployment, got nil")
		}
		if got.ID != dep.ID {
			t.Fatalf("expected ID %q, got %q", dep.ID, got.ID)
		}
	})

	t.Run("get preserves scope and owner", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "scope-test", "nodes": []interface{}{}, "edges": []interface{}{},
		}
		dep, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{Owner: "test@example.com"})
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		if dep.Scope != "dev" {
			t.Fatalf("expected scope dev, got %q", dep.Scope)
		}
		if dep.Owner != "test@example.com" {
			t.Fatalf("expected owner test@example.com, got %q", dep.Owner)
		}

		got, err := uc.GetDeployment(ctx, dep.ID)
		if err != nil {
			t.Fatalf("GetDeployment: %v", err)
		}
		if got.Scope != "dev" {
			t.Fatalf("expected persisted scope dev, got %q", got.Scope)
		}
		if got.Owner != "test@example.com" {
			t.Fatalf("expected persisted owner test@example.com, got %q", got.Owner)
		}
	})

	t.Run("get returns nil for missing", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		got, err := uc.GetDeployment(ctx, "missing-id")
		if err != nil {
			t.Fatalf("GetDeployment: %v", err)
		}
		if got != nil {
			t.Fatal("expected nil for missing deployment")
		}
	})

	t.Run("delete removes deployment", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "del-test", "nodes": []interface{}{}, "edges": []interface{}{},
		}
		dep, err := uc.Deploy(ctx, pipe, "", nil)
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}

		if err := uc.DeleteDeployment(ctx, dep.ID); err != nil {
			t.Fatalf("DeleteDeployment: %v", err)
		}

		got, _ := uc.GetDeployment(ctx, dep.ID)
		if got != nil {
			t.Fatal("expected nil after delete")
		}
	})
}

// ── RegisterOutput ────────────────────────────────────────────────────────

func TestRegisterOutput(t *testing.T) {
	ctx := context.Background()

	t.Run("registers output asset with lineage", func(t *testing.T) {
		repo := newMockAssetRepo()
		repo.assets["input-1"] = &models.Asset{AssetID: "input-1", AssetType: "dataset", StorageURI: "gs://bucket/in1"}
		repo.assets["input-2"] = &models.Asset{AssetID: "input-2", AssetType: "dataset", StorageURI: "gs://bucket/in2"}
		eventRepo := &mockEventRepo{}
		relWriter := &mockRelationWriter{}
		logicalRepo := &mockLogicalAssetRepo{}
		uc := newUsecase(repo)
		uc.assetEventRepo = eventRepo
		uc.relationWriter = relWriter
		uc.logicalRepo = logicalRepo

		pipe := map[string]interface{}{
			"name": "output-test",
			"nodes": []interface{}{
				map[string]interface{}{"id": "step-1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		dep, err := uc.Deploy(ctx, pipe, "", []string{"input-1", "input-2"})
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}

		asset, err := uc.RegisterOutput(ctx, RegisterPipelineOutputInput{
			DeploymentID: dep.ID,
			NodeID:       "step-1",
			AssetID:      "output-1",
			StorageURI:   "gs://bucket/result",
			AssetType:    "dataset",
		})
		if err != nil {
			t.Fatalf("RegisterOutput: %v", err)
		}
		if asset == nil {
			t.Fatal("expected asset, got nil")
		}
		if asset.AssetID != "output-1" {
			t.Fatalf("expected 'output-1', got %q", asset.AssetID)
		}
		if asset.StorageURI != "gs://bucket/result" {
			t.Fatalf("expected 'gs://bucket/result', got %q", asset.StorageURI)
		}
		if asset.LogicalAssetID != "output-1" || asset.Revision != 1 || !asset.IsCurrent {
			t.Fatalf("expected versioned pipeline output asset, got %#v", asset)
		}
		if len(logicalRepo.inserted) != 1 {
			t.Fatalf("expected logical asset insert, got %d", len(logicalRepo.inserted))
		}

		// Verify event was appended (2 from Deploy input assets + 1 from RegisterOutput).
		if len(eventRepo.appended) != 3 {
			t.Fatalf("expected 3 events, got %d", len(eventRepo.appended))
		}
		lastEvent := eventRepo.appended[len(eventRepo.appended)-1]
		if lastEvent.EventType != "pipeline_output" {
			t.Fatalf("expected event type 'pipeline_output', got %q", lastEvent.EventType)
		}
		if lastEvent.AssetID != "output-1" {
			t.Fatalf("expected asset 'output-1', got %q", lastEvent.AssetID)
		}

		// Verify relations were created from input assets.
		if len(relWriter.relations) != 2 {
			t.Fatalf("expected 2 relations, got %d", len(relWriter.relations))
		}
		if relWriter.relations[0].srcAssetID != "input-1" || relWriter.relations[1].srcAssetID != "input-2" {
			t.Fatal("relations have wrong source asset IDs")
		}
		if relWriter.relations[0].relationType != "pipeline_output" {
			t.Fatalf("expected relation type 'pipeline_output', got %q", relWriter.relations[0].relationType)
		}
	})

	t.Run("returns error for missing deployment", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		_, err := uc.RegisterOutput(ctx, RegisterPipelineOutputInput{
			DeploymentID: "nonexistent",
			NodeID:       "step-1",
			AssetID:      "out-1",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDeploymentNotFound) {
			t.Fatalf("expected ErrDeploymentNotFound, got: %v", err)
		}
	})

	t.Run("auto-generates asset ID when empty", func(t *testing.T) {
		repo := newMockAssetRepo()
		eventRepo := &mockEventRepo{}
		uc := newUsecase(repo)
		uc.assetEventRepo = eventRepo

		pipe := map[string]interface{}{
			"name": "auto-id",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		dep, err := uc.Deploy(ctx, pipe, "", nil)
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}

		asset, err := uc.RegisterOutput(ctx, RegisterPipelineOutputInput{
			DeploymentID: dep.ID,
			NodeID:       "s1",
			AssetID:      "",
			StorageURI:   "gs://bucket/auto",
			AssetType:    "dataset",
		})
		if err != nil {
			t.Fatalf("RegisterOutput: %v", err)
		}
		if asset.AssetID == "" {
			t.Fatal("expected non-empty auto-generated asset ID")
		}
	})
}

// ── GetLineage ────────────────────────────────────────────────────────────

func TestGetLineage(t *testing.T) {
	ctx := context.Background()

	t.Run("returns lineage from events and deployment", func(t *testing.T) {
		repo := newMockAssetRepo()
		repo.assets["input-a"] = &models.Asset{AssetID: "input-a", AssetType: "dataset"}
		eventRepo := &mockEventRepo{events: make(map[string][]*models.AssetEvent)}
		uc := newUsecase(repo)
		uc.assetEventRepo = eventRepo

		pipe := map[string]interface{}{
			"name": "lineage-test",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		dep, err := uc.Deploy(ctx, pipe, "", []string{"input-a"})
		if err != nil {
			t.Fatalf("Deploy: %v", err)
		}

		// Simulate the event that RegisterOutput would produce.
		payload, _ := json.Marshal(map[string]interface{}{
			"deployment_id": dep.ID,
			"node_id":       "s1",
			"pipeline_name": "lineage-test",
		})
		eventRepo.events["output-1"] = []*models.AssetEvent{
			{
				EventType:    "pipeline_output",
				EventPayload: payload,
				OccurredAt:   time.Now().UTC(),
			},
		}

		lineage, err := uc.GetLineage(ctx, "output-1")
		if err != nil {
			t.Fatalf("GetLineage: %v", err)
		}
		if lineage == nil {
			t.Fatal("expected lineage, got nil")
		}
		if lineage.AssetID != "output-1" {
			t.Fatalf("expected assetID 'output-1', got %q", lineage.AssetID)
		}
		if lineage.DeploymentID != dep.ID {
			t.Fatalf("expected deploymentID %q, got %q", dep.ID, lineage.DeploymentID)
		}
		if lineage.PipelineName != "lineage-test" {
			t.Fatalf("expected pipelineName 'lineage-test', got %q", lineage.PipelineName)
		}
		if lineage.WorkflowName != dep.WorkflowName {
			t.Fatalf("expected workflowName %q, got %q", dep.WorkflowName, lineage.WorkflowName)
		}
		if lineage.NodeID != "s1" {
			t.Fatalf("expected nodeID 's1', got %q", lineage.NodeID)
		}
		if len(lineage.InputAssets) != 1 || lineage.InputAssets[0] != "input-a" {
			t.Fatalf("expected inputAssets ['input-a'], got %v", lineage.InputAssets)
		}
		if lineage.ProducedAt == "" {
			t.Fatal("expected non-empty ProducedAt")
		}
	})

	t.Run("returns basic info when no events exist", func(t *testing.T) {
		repo := newMockAssetRepo()
		eventRepo := &mockEventRepo{events: make(map[string][]*models.AssetEvent)}
		uc := newUsecase(repo)
		uc.assetEventRepo = eventRepo

		lineage, err := uc.GetLineage(ctx, "unknown-asset")
		if err != nil {
			t.Fatalf("GetLineage: %v", err)
		}
		if lineage == nil {
			t.Fatal("expected lineage, got nil")
		}
		if lineage.AssetID != "unknown-asset" {
			t.Fatalf("expected assetID 'unknown-asset', got %q", lineage.AssetID)
		}
		if lineage.DeploymentID != "" {
			t.Fatalf("expected empty deploymentID for asset with no events")
		}
	})
}

// ── DiffTemplates ─────────────────────────────────────────────────────────

func TestDiffTemplates(t *testing.T) {
	ctx := context.Background()

	t.Run("empty diff for identical templates", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "diff-test",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		t1, _ := uc.SaveTemplate(ctx, "diff-test", pipe, "dev", "legacy")
		t2, _ := uc.SaveTemplate(ctx, "diff-test", pipe, "dev", "legacy")

		diff, err := uc.DiffTemplates(ctx, t1.ID, t2.ID)
		if err != nil {
			t.Fatalf("DiffTemplates: %v", err)
		}
		if diff == nil {
			t.Fatal("expected diff, got nil")
		}
		if len(diff.AddedNodes) != 0 {
			t.Fatalf("expected 0 added nodes, got %d", len(diff.AddedNodes))
		}
		if len(diff.RemovedNodes) != 0 {
			t.Fatalf("expected 0 removed nodes, got %d", len(diff.RemovedNodes))
		}
		if len(diff.ModifiedNodes) != 0 {
			t.Fatalf("expected 0 modified nodes, got %d", len(diff.ModifiedNodes))
		}
		if len(diff.AddedEdges) != 0 {
			t.Fatalf("expected 0 added edges, got %d", len(diff.AddedEdges))
		}
		if len(diff.RemovedEdges) != 0 {
			t.Fatalf("expected 0 removed edges, got %d", len(diff.RemovedEdges))
		}
	})

	t.Run("detects added nodes", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		basePipe := map[string]interface{}{
			"name": "diff-add",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		extendedPipe := map[string]interface{}{
			"name": "diff-add",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
				map[string]interface{}{"id": "s2", "component": map[string]interface{}{"name": "b", "image": "img"}},
			},
			"edges": []interface{}{},
		}

		t1, _ := uc.SaveTemplate(ctx, "diff-add", basePipe, "dev", "legacy")
		t2, _ := uc.SaveTemplate(ctx, "diff-add", extendedPipe, "dev", "legacy")

		diff, err := uc.DiffTemplates(ctx, t1.ID, t2.ID)
		if err != nil {
			t.Fatalf("DiffTemplates: %v", err)
		}
		if len(diff.AddedNodes) != 1 || diff.AddedNodes[0].ID != "s2" {
			t.Fatalf("expected 1 added node 's2', got %v", diff.AddedNodes)
		}
	})

	t.Run("detects removed nodes", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		fullPipe := map[string]interface{}{
			"name": "diff-rem",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
				map[string]interface{}{"id": "s2", "component": map[string]interface{}{"name": "b", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		reducedPipe := map[string]interface{}{
			"name": "diff-rem",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
			},
			"edges": []interface{}{},
		}

		t1, _ := uc.SaveTemplate(ctx, "diff-rem", fullPipe, "dev", "legacy")
		t2, _ := uc.SaveTemplate(ctx, "diff-rem", reducedPipe, "dev", "legacy")

		diff, err := uc.DiffTemplates(ctx, t1.ID, t2.ID)
		if err != nil {
			t.Fatalf("DiffTemplates: %v", err)
		}
		if len(diff.RemovedNodes) != 1 || diff.RemovedNodes[0].ID != "s2" {
			t.Fatalf("expected 1 removed node 's2', got %v", diff.RemovedNodes)
		}
	})

	t.Run("detects modified nodes", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		oldPipe := map[string]interface{}{
			"name": "diff-mod",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img:v1"}},
			},
			"edges": []interface{}{},
		}
		newPipe := map[string]interface{}{
			"name": "diff-mod",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img:v2"}},
			},
			"edges": []interface{}{},
		}

		t1, _ := uc.SaveTemplate(ctx, "diff-mod", oldPipe, "dev", "legacy")
		t2, _ := uc.SaveTemplate(ctx, "diff-mod", newPipe, "dev", "legacy")

		diff, err := uc.DiffTemplates(ctx, t1.ID, t2.ID)
		if err != nil {
			t.Fatalf("DiffTemplates: %v", err)
		}
		if len(diff.ModifiedNodes) != 1 || diff.ModifiedNodes[0].ID != "s1" {
			t.Fatalf("expected 1 modified node 's1', got %v", diff.ModifiedNodes)
		}
	})

	t.Run("detects added and removed edges", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		noEdge := map[string]interface{}{
			"name": "diff-edge",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
				map[string]interface{}{"id": "s2", "component": map[string]interface{}{"name": "b", "image": "img"}},
			},
			"edges": []interface{}{},
		}
		withEdge := map[string]interface{}{
			"name": "diff-edge",
			"nodes": []interface{}{
				map[string]interface{}{"id": "s1", "component": map[string]interface{}{"name": "a", "image": "img"}},
				map[string]interface{}{"id": "s2", "component": map[string]interface{}{"name": "b", "image": "img"}},
			},
			"edges": []interface{}{
				map[string]interface{}{"source": "s1.output", "target": "s2.input"},
			},
		}

		t1, _ := uc.SaveTemplate(ctx, "diff-edge", noEdge, "dev", "legacy")
		t2, _ := uc.SaveTemplate(ctx, "diff-edge", withEdge, "dev", "legacy")

		diff, err := uc.DiffTemplates(ctx, t1.ID, t2.ID)
		if err != nil {
			t.Fatalf("DiffTemplates: %v", err)
		}
		if len(diff.AddedEdges) != 1 {
			t.Fatalf("expected 1 added edge, got %d", len(diff.AddedEdges))
		}
		if diff.AddedEdges[0].Source != "s1.output" || diff.AddedEdges[0].Target != "s2.input" {
			t.Fatalf("unexpected edge: %+v", diff.AddedEdges[0])
		}
	})

	t.Run("returns error for missing templates", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		pipe := map[string]interface{}{
			"name": "err", "nodes": []interface{}{}, "edges": []interface{}{},
		}
		t1, _ := uc.SaveTemplate(ctx, "err", pipe, "dev", "legacy")

		_, err := uc.DiffTemplates(ctx, t1.ID, "nonexistent")
		if err == nil {
			t.Fatal("expected error for missing template id, got nil")
		}
	})
}
