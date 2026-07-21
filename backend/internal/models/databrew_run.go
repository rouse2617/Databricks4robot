package models

import "time"

const (
	RunTypePipeline       = "pipeline"
	RunTypeComponentBuild = "component_build"
	RunTypeRAGBuild       = "rag_build"
)

// DatabrewRun is the unified execution record for all Argo-backed runs.
type DatabrewRun struct {
	ID                  string      `json:"id"`
	Type                string      `json:"type"`
	Name                string      `json:"name"`
	Status              string      `json:"status"`
	StatusLabel         string      `json:"statusLabel,omitempty"`
	Runtime             string      `json:"runtime"`
	RuntimeNamespace    string      `json:"runtimeNamespace"`
	RuntimeResourceName string      `json:"runtimeResourceName"`
	RuntimeUID          string      `json:"runtimeUid,omitempty"`
	Owner               string      `json:"owner,omitempty"`
	CreatedBy           string      `json:"createdBy,omitempty"`
	Message             string      `json:"message,omitempty"`
	Summary             any         `json:"summary,omitempty"`
	Actions             *RunActions `json:"actions,omitempty"`
	CreatedAt           time.Time   `json:"createdAt"`
	StartedAt           *time.Time  `json:"startedAt,omitempty"`
	FinishedAt          *time.Time  `json:"finishedAt,omitempty"`
	UpdatedAt           time.Time   `json:"updatedAt"`
}

// RunActions describes which lifecycle operations are available for a run.
type RunActions struct {
	CanRetry     bool `json:"canRetry"`
	CanTerminate bool `json:"canTerminate"`
	CanSuspend   bool `json:"canSuspend"`
	CanResume    bool `json:"canResume"`
	CanResubmit  bool `json:"canResubmit"`
	CanStop      bool `json:"canStop"`
}

// DatabrewRunListFilter scopes paginated run list queries.
type DatabrewRunListFilter struct {
	Type     string
	Status   string
	Page     int
	PageSize int
}

// DatabrewRunListResult is a paginated list of runs.
type DatabrewRunListResult struct {
	Items []DatabrewRun `json:"items"`
	Total int           `json:"total"`
}

// ComponentBuildRun stores component build extension fields.
type ComponentBuildRun struct {
	RunID           string `json:"runId"`
	ComponentID     string `json:"componentId,omitempty"`
	RepoURL         string `json:"repoUrl,omitempty"`
	GitRef          string `json:"gitRef,omitempty"`
	CommitSHA       string `json:"commitSha,omitempty"`
	Dockerfile      string `json:"dockerfile,omitempty"`
	BuildContext    string `json:"buildContext,omitempty"`
	ImageRepository string `json:"imageRepository,omitempty"`
	ImageTag        string `json:"imageTag,omitempty"`
	ImageDigest     string `json:"imageDigest,omitempty"`
}

// RAGBuildRun stores RAG build extension fields.
type RAGBuildRun struct {
	RunID              string                 `json:"runId"`
	KnowledgeBaseID    string                 `json:"knowledgeBaseId,omitempty"`
	DatasourceSnapshot map[string]interface{} `json:"datasourceSnapshot,omitempty"`
	EmbeddingModel     string                 `json:"embeddingModel,omitempty"`
	VectorIndexName    string                 `json:"vectorIndexName,omitempty"`
	ReleaseVersion     string                 `json:"releaseVersion,omitempty"`
}

// ComponentRelease records a built component image version.
type ComponentRelease struct {
	ID           string    `json:"id"`
	ComponentID  string    `json:"componentId"`
	SourceCommit string    `json:"sourceCommit,omitempty"`
	Image        string    `json:"image,omitempty"`
	ImageTag     string    `json:"imageTag,omitempty"`
	ImageDigest  string    `json:"imageDigest,omitempty"`
	ReleaseLabel string    `json:"releaseLabel,omitempty"`
	BuildRunID   *string   `json:"buildRunId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
