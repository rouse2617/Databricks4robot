package postgres

import (
	"context"
	"testing"

	"data-platform/internal/models"
)

func TestSavedQueryRepo_ListGetCreateUpdateDelete(t *testing.T) {
	ctx := context.Background()
	repo := &SavedQueryRepo{c: &Client{db: &fakeDB{execRowsAffected: 1}}}

	createdTime := mustTime(t, "2026-05-07T12:00:00Z")
	updatedTime := mustTime(t, "2026-05-07T12:30:00Z")

	repo.c.db.(*fakeDB).queryRow = &fakeRow{
		values: []any{
			"sq-1", createdTime, updatedTime,
		},
	}
	created, err := repo.Create(ctx, savedQueryFixture())
	if err != nil {
		t.Fatalf("Create() err = %v", err)
	}
	if created.SavedQueryID != "sq-1" {
		t.Fatalf("unexpected saved_query_id: %q", created.SavedQueryID)
	}

	repo.c.db.(*fakeDB).queryRow = &fakeRow{
		values: []any{
			"sq-1",
			"Recent Ready Assets",
			"",
			"assets",
			"v1",
			[]byte(`{"schema_version":"v1","scope":{"resource":"assets"}}`),
			"alice",
			createdTime,
			updatedTime,
		},
	}
	got, err := repo.Get(ctx, "sq-1")
	if err != nil {
		t.Fatalf("Get() err = %v", err)
	}
	if got == nil || got.Name != "Recent Ready Assets" {
		t.Fatalf("unexpected Get() result: %+v", got)
	}

	repo.c.db.(*fakeDB).rows = &fakeRows{
		data: [][]any{{
			"sq-1",
			"Recent Ready Assets",
			"",
			"assets",
			"v1",
			[]byte(`{"schema_version":"v1","scope":{"resource":"assets"}}`),
			"alice",
			createdTime,
			updatedTime,
		}},
	}
	items, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() err = %v", err)
	}
	if len(items) != 1 || items[0].SavedQueryID != "sq-1" {
		t.Fatalf("unexpected List() result: %+v", items)
	}

	repo.c.db.(*fakeDB).queryRow = &fakeRow{
		values: []any{
			createdTime, updatedTime,
		},
	}
	item := savedQueryFixture()
	item.SavedQueryID = "sq-1"
	item.Name = "Updated Query"
	updated, err := repo.Update(ctx, item)
	if err != nil {
		t.Fatalf("Update() err = %v", err)
	}
	if updated == nil || updated.Name != "Updated Query" {
		t.Fatalf("unexpected Update() result: %+v", updated)
	}

	if err := repo.Delete(ctx, "sq-1"); err != nil {
		t.Fatalf("Delete() err = %v", err)
	}
}

func savedQueryFixture() *models.SavedQuery {
	return &models.SavedQuery{
		Name:          "Recent Ready Assets",
		Resource:      "assets",
		SchemaVersion: "v1",
		QueryIRJSON: map[string]interface{}{
			"schema_version": "v1",
			"scope": map[string]interface{}{
				"resource": "assets",
			},
		},
		Owner: "alice",
	}
}
