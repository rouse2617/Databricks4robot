package workflow

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
)

// ArchiveLogResult holds logs retrieved from an archive source.
type ArchiveLogResult struct {
	Logs      string
	LineCount int
	Source    string // "archive-loki", "archive-s3", "archive-gcs", etc.
}

// ArchiveLogStore defines the interface for fetching archived workflow logs.
// Implementations can connect to Loki, S3, GCS, ClickHouse, etc.
type ArchiveLogStore interface {
	// GetLogs retrieves logs from the archive. Returns nil, nil if unavailable.
	GetLogs(ctx context.Context, workflowName, nodeID, podName string, opts argo.WorkflowLogOptions) (*ArchiveLogResult, error)
}

// noopArchiveStore returns nil, nil for all requests — archive unavailable.
type noopArchiveStore struct{}

func (s *noopArchiveStore) GetLogs(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (*ArchiveLogResult, error) {
	return nil, nil
}

// NewNoopArchiveStore returns an archive store that always reports "unavailable".
// Replace with a real implementation (Loki, S3, GCS) when archive storage is configured.
func NewNoopArchiveStore() ArchiveLogStore {
	return &noopArchiveStore{}
}
