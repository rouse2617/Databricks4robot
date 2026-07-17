package pipeline

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// TestListRuns_NoAssetNodeWriteOnRead is a performance regression guard.
//
// enrichRun used to unconditionally call refreshAssetNodes, which is a WRITE
// (assetNodeRepo.ReplaceByRunID = DELETE+INSERT). ListRuns fans enrichRun over
// every row, so a plain list read turned into an O(N) write storm + row-lock
// churn — amplified by the frontend polling the list every few seconds.
//
// The fix threads a refreshAssets flag through enrichRun; the list path passes
// false. This test locks that in: ListRuns must not touch the asset_node repo.
func TestListRuns_NoAssetNodeWriteOnRead(t *testing.T) {
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{
		"r1": {ID: "r1", Status: "Succeeded"},
		"r2": {ID: "r2", Status: "Failed"},
	}}
	assetNodes := &mockAssetNodeRepo{}
	uc := &Usecase{runRepo: runRepo}
	uc.SetObservabilityRepositories(assetNodes, nil, nil)

	runs, err := uc.ListRuns(context.Background(), false)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	if len(assetNodes.byRun) != 0 {
		t.Errorf(
			"write-on-read regression: ListRuns wrote asset_nodes for %d run(s); the list read path must not write",
			len(assetNodes.byRun),
		)
	}
}

// TestGetRun-style single-run paths still refresh asset_nodes (refreshAssets=true)
// — that lazy write is low-frequency drill-in, not a per-list-row storm. We keep
// it by leaving the single-run enrichRun callers on true; no assertion here
// beyond documenting the intent, since GetRun pulls in the Argo client.
