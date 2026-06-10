package pipeline_component

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ── Mock repository ─────────────────────────────────────────────────────────

type mockComponentRepo struct {
	byID      map[string]*models.PipelineComponent
	all       []models.PipelineComponent
	saved     []*models.PipelineComponent
	updateErr error
	deleteErr error
}

func newMockComponentRepo() *mockComponentRepo {
	return &mockComponentRepo{
		byID: make(map[string]*models.PipelineComponent),
	}
}

func (m *mockComponentRepo) Save(_ context.Context, pc *models.PipelineComponent) error {
	if m.byID == nil {
		m.byID = make(map[string]*models.PipelineComponent)
	}
	cp := *pc
	m.byID[pc.ID] = &cp
	m.saved = append(m.saved, &cp)
	// Also update the all slice
	m.all = nil
	for _, v := range m.byID {
		m.all = append(m.all, *v)
	}
	return nil
}

func (m *mockComponentRepo) FindAll(_ context.Context, filter *repository.ComponentFilter) ([]models.PipelineComponent, error) {
	if filter == nil {
		return m.all, nil
	}
	var out []models.PipelineComponent
	for _, pc := range m.all {
		if filter.Query != "" && !contains(pc.Name, filter.Query) {
			continue
		}
		if filter.Source != "" && pc.Source != filter.Source {
			continue
		}
		out = append(out, pc)
	}
	return out, nil
}

func (m *mockComponentRepo) FindByID(_ context.Context, id string) (*models.PipelineComponent, error) {
	pc, ok := m.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *pc
	return &cp, nil
}

func (m *mockComponentRepo) Update(_ context.Context, pc *models.PipelineComponent) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.byID[pc.ID]; !ok {
		return errors.New("component not found")
	}
	cp := *pc
	m.byID[pc.ID] = &cp
	// Refresh all
	m.all = nil
	for _, v := range m.byID {
		m.all = append(m.all, *v)
	}
	return nil
}

func (m *mockComponentRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.byID, id)
	// Refresh all
	m.all = nil
	for _, v := range m.byID {
		m.all = append(m.all, *v)
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0)
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestCreate(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	pc := &models.PipelineComponent{
		Name:  "test-component",
		Type:  "container",
		Image: "nginx:latest",
		Tag:   "latest",
	}

	created, err := uc.Create(context.Background(), pc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == "" {
		t.Error("Create should set a non-empty ID")
	}
	if created.Source != "custom" {
		t.Errorf("Source should default to 'custom', got %q", created.Source)
	}
	if created.InputPorts == nil {
		t.Error("InputPorts should not be nil")
	}
	if created.OutputPorts == nil {
		t.Error("OutputPorts should not be nil")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if created.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	// Verify it was persisted
	found, err := uc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get after Create failed: %v", err)
	}
	if found == nil {
		t.Fatal("Created component not found via Get")
	}
	if found.Name != "test-component" {
		t.Errorf("Got name %q, want %q", found.Name, "test-component")
	}
}

func TestCreate_EmptyList(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	items, err := uc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected empty list, got %d items", len(items))
	}
}

func TestCreateAndList(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	_, _ = uc.Create(context.Background(), &models.PipelineComponent{
		Name: "alpha", Type: "container", Image: "alpine:latest", Tag: "latest",
	})
	_, _ = uc.Create(context.Background(), &models.PipelineComponent{
		Name: "beta", Type: "container", Image: "busybox:latest", Tag: "latest",
	})

	// List all
	items, err := uc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// Filter by query
	items, err = uc.List(context.Background(), "alpha", "")
	if err != nil {
		t.Fatalf("List with query failed: %v", err)
	}
	if len(items) != 1 || items[0].Name != "alpha" {
		t.Errorf("Expected 1 item (alpha), got %d", len(items))
	}
}

func TestList_FilterBySource(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	_, _ = uc.Create(context.Background(), &models.PipelineComponent{
		Name: "sys-one", Type: "container", Image: "busybox", Tag: "1", Source: "system",
	})
	_, _ = uc.Create(context.Background(), &models.PipelineComponent{
		Name: "custom-one", Type: "container", Image: "nginx", Tag: "1", Source: "custom",
	})

	items, err := uc.List(context.Background(), "", "system")
	if err != nil {
		t.Fatalf("List by source failed: %v", err)
	}
	if len(items) != 1 || items[0].Source != "system" {
		t.Errorf("Expected 1 system component, got %d", len(items))
	}

	items, err = uc.List(context.Background(), "", "custom")
	if err != nil {
		t.Fatalf("List by custom source failed: %v", err)
	}
	if len(items) != 1 || items[0].Source != "custom" {
		t.Errorf("Expected 1 custom component, got %d", len(items))
	}
}

func TestGet_NonExistent(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	pc, err := uc.Get(context.Background(), "non-existent-id")
	if err != nil {
		t.Fatalf("Get non-existent failed: %v", err)
	}
	if pc != nil {
		t.Error("Expected nil for non-existent component")
	}
}

func TestGet_Existing(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	created, _ := uc.Create(context.Background(), &models.PipelineComponent{
		Name: "fetch-me", Type: "container", Image: "busybox", Tag: "1",
	})

	fetched, err := uc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("Get returned nil for existing component")
	}
	if fetched.Name != "fetch-me" {
		t.Errorf("Got name %q, want %q", fetched.Name, "fetch-me")
	}
	if fetched.Image != "busybox" {
		t.Errorf("Got image %q, want %q", fetched.Image, "busybox")
	}
}

func TestUpdate_Existing(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	created, _ := uc.Create(context.Background(), &models.PipelineComponent{
		Name: "old-name", Type: "container", Image: "nginx:1.0", Tag: "1.0",
	})

	created.Name = "new-name"
	created.Image = "nginx:2.0"
	created.Tag = "2.0"

	err := uc.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	fetched, _ := uc.Get(context.Background(), created.ID)
	if fetched == nil {
		t.Fatal("Component disappeared after update")
	}
	if fetched.Name != "new-name" {
		t.Errorf("Name after update: got %q, want %q", fetched.Name, "new-name")
	}
	if fetched.Image != "nginx:2.0" {
		t.Errorf("Image after update: got %q, want %q", fetched.Image, "nginx:2.0")
	}
	if fetched.Tag != "2.0" {
		t.Errorf("Tag after update: got %q, want %q", fetched.Tag, "2.0")
	}
}

func TestUpdate_NonExistent(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	pc := &models.PipelineComponent{
		ID:    "non-existent",
		Name:  "ghost",
		Type:  "container",
		Image: "busybox",
	}

	err := uc.Update(context.Background(), pc)
	if err == nil {
		t.Fatal("Expected error when updating non-existent component")
	}
}

func TestDelete_Existing(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	created, _ := uc.Create(context.Background(), &models.PipelineComponent{
		Name: "delete-me", Type: "container", Image: "busybox", Tag: "1",
	})

	err := uc.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	fetched, _ := uc.Get(context.Background(), created.ID)
	if fetched != nil {
		t.Error("Component still exists after Delete")
	}
}

func TestDelete_NonExistent(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	err := uc.Delete(context.Background(), "non-existent")
	if err == nil {
		t.Fatal("Delete non-existent should error")
	}
}

func TestSeedSystemComponents_Empty(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	err := uc.SeedSystemComponents(context.Background())
	if err != nil {
		t.Fatalf("SeedSystemComponents failed: %v", err)
	}

	// Should have created 'sys-pass-through'
	pc, err := uc.Get(context.Background(), "sys-pass-through")
	if err != nil {
		t.Fatalf("Get after seed failed: %v", err)
	}
	if pc == nil {
		t.Fatal("sys-pass-through not found after seed")
	}
	if pc.Source != "system" {
		t.Errorf("Expected source 'system', got %q", pc.Source)
	}
	if pc.Type != "container" {
		t.Errorf("Expected type 'container', got %q", pc.Type)
	}
	if pc.Name != "Pass Through" {
		t.Errorf("Expected name 'Pass Through', got %q", pc.Name)
	}
}

func TestSeedSystemComponents_Idempotent(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	// Seed once
	if err := uc.SeedSystemComponents(context.Background()); err != nil {
		t.Fatalf("First seed failed: %v", err)
	}

	// Seed again — should be idempotent
	if err := uc.SeedSystemComponents(context.Background()); err != nil {
		t.Fatalf("Second seed failed: %v", err)
	}

	// Should still have exactly one sys-pass-through in the list
	items, err := uc.List(context.Background(), "", "system")
	if err != nil {
		t.Fatalf("List system components failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected exactly 1 system component, got %d", len(items))
	}
}

func TestCreate_WithExplicitSource(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	pc := &models.PipelineComponent{
		Name:   "explicit-source",
		Type:   "container",
		Image:  "python:3.12",
		Tag:    "3.12",
		Source: "dockerhub",
	}

	created, err := uc.Create(context.Background(), pc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Source != "dockerhub" {
		t.Errorf("Source should be 'dockerhub', got %q", created.Source)
	}
}

func TestCreate_WithPorts(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	pc := &models.PipelineComponent{
		Name:  "with-ports",
		Type:  "container",
		Image: "custom:latest",
		Tag:   "latest",
		InputPorts: []models.PortDef{
			{Name: "in-data", Type: "asset", Desc: "Input data asset"},
		},
		OutputPorts: []models.PortDef{
			{Name: "out-result", Type: "asset", Desc: "Output result asset"},
		},
	}

	created, err := uc.Create(context.Background(), pc)
	if err != nil {
		t.Fatalf("Create with ports failed: %v", err)
	}
	if len(created.InputPorts) != 1 || created.InputPorts[0].Name != "in-data" {
		t.Errorf("InputPorts not preserved: got %+v", created.InputPorts)
	}
	if len(created.OutputPorts) != 1 || created.OutputPorts[0].Name != "out-result" {
		t.Errorf("OutputPorts not preserved: got %+v", created.OutputPorts)
	}
}

func TestUpdate_PreservesCreatedAt(t *testing.T) {
	repo := newMockComponentRepo()
	uc := New(repo)

	created, _ := uc.Create(context.Background(), &models.PipelineComponent{
		Name: "preserve-time", Type: "container", Image: "busybox", Tag: "1",
	})
	originalCreatedAt := created.CreatedAt

	created.Name = "updated-name"
	_ = uc.Update(context.Background(), created)

	fetched, _ := uc.Get(context.Background(), created.ID)
	if fetched == nil {
		t.Fatal("Component not found after update")
	}
	if !fetched.CreatedAt.Equal(originalCreatedAt) {
		t.Error("Update should preserve CreatedAt")
	}
}
