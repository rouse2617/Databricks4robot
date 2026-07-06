package workflow

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestLookupWorkflowNodeRuntimeInfo_DualFormat(t *testing.T) {
	run := &models.PipelineRun{
		PipelineJSON: map[string]any{
			"nodes": []any{
				map[string]any{
					"id":        "node-aaaaaaaa-0000-0000-0000-000000000001",
					"component": map[string]any{"name": "head-track"},
					"image":     "repo/head-track:v1",
					"commit":    "abc123",
				},
			},
		},
	}

	// New readable template name resolves.
	if info := lookupWorkflowNodeRuntimeInfo(run, "step-head-track"); info == nil || info.Image != "repo/head-track:v1" {
		t.Fatalf("new-format lookup = %+v", info)
	}
	// New readable name with uuid8 suffix (duplicate-component case) resolves.
	if info := lookupWorkflowNodeRuntimeInfo(run, "step-head-track-aaaaaaaa"); info == nil || info.SourceCommit != "abc123" {
		t.Fatalf("new-format uuid8 lookup = %+v", info)
	}
	// Legacy template name (historical runs) still resolves.
	if info := lookupWorkflowNodeRuntimeInfo(run, "step-node-aaaaaaaa-0000-0000-0000-000000000001"); info == nil || info.Image != "repo/head-track:v1" {
		t.Fatalf("legacy-format lookup = %+v", info)
	}
	// Unrelated template name does not resolve.
	if info := lookupWorkflowNodeRuntimeInfo(run, "step-something-else"); info != nil {
		t.Fatalf("unexpected match = %+v", info)
	}
}
