package resourcevalidation

import "testing"

func TestValidateResourceStrings(t *testing.T) {
	tests := []struct {
		name       string
		cpu        string
		memory     string
		disk       string
		gpu        string
		wantErrors int
	}{
		{name: "valid common quantities", cpu: "500m", memory: "512Mi", disk: "20Gi", gpu: "1"},
		{name: "valid cpu cores", cpu: "1", memory: "1Gi", disk: "1Gi", gpu: "0"},
		{name: "bare memory rejected", cpu: "1", memory: "1", disk: "1Gi", gpu: "0", wantErrors: 1},
		{name: "bare disk rejected", cpu: "1", memory: "512Mi", disk: "1", gpu: "0", wantErrors: 1},
		{name: "zero cpu rejected", cpu: "0", memory: "512Mi", disk: "1Gi", gpu: "0", wantErrors: 1},
		{name: "zero memory rejected", cpu: "1", memory: "0Gi", disk: "1Gi", gpu: "0", wantErrors: 1},
		{name: "zero disk rejected", cpu: "1", memory: "512Mi", disk: "0Gi", gpu: "0", wantErrors: 1},
		{name: "negative cpu rejected", cpu: "-1", memory: "512Mi", disk: "1Gi", gpu: "0", wantErrors: 1},
		{name: "fractional gpu rejected", cpu: "1", memory: "512Mi", disk: "1Gi", gpu: "0.5", wantErrors: 1},
		{name: "invalid cpu rejected", cpu: "abc", memory: "512Mi", disk: "1Gi", gpu: "0", wantErrors: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateResourceStrings("node step-a", tt.cpu, tt.memory, tt.disk, tt.gpu)
			if len(got) != tt.wantErrors {
				t.Fatalf("got %d errors %v, want %d", len(got), got, tt.wantErrors)
			}
		})
	}
}
