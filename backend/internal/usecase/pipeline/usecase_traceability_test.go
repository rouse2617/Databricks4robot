package pipeline

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestInferRunTriggerSource(t *testing.T) {
	templateID := "tmpl-1"
	tests := []struct {
		name string
		run  *models.PipelineRun
		want string
	}{
		{
			name: "batch takes precedence",
			run: &models.PipelineRun{
				BatchRunID: "batch-1",
				AssetIDs:   []string{"asset-a"},
			},
			want: RunTriggerSourceBatch,
		},
		{
			name: "asset run with assets",
			run: &models.PipelineRun{
				AssetIDs:   []string{"asset-a"},
				AssetCount: 1,
			},
			want: RunTriggerSourceAssetRun,
		},
		{
			name: "manual template run without assets",
			run: &models.PipelineRun{
				TemplateID: &templateID,
			},
			want: RunTriggerSourceManual,
		},
		{
			name: "api fallback without template or assets",
			run:  &models.PipelineRun{},
			want: RunTriggerSourceAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferRunTriggerSource(tt.run)
			if got != tt.want {
				t.Fatalf("inferRunTriggerSource() = %q, want %q", got, tt.want)
			}
		})
	}
}
