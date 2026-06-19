package models

import "time"

// PipelineConfig is a standalone user-owned config file record.
type PipelineConfig struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	Owner          string                  `json:"owner"`
	Scope          string                  `json:"scope,omitempty"`
	Tags           []string                `json:"tags"`
	FileType       string                  `json:"fileType"`
	Lifecycle      string                  `json:"lifecycle"`
	CurrentVersion int                     `json:"currentVersion"`
	VersionCount   int                     `json:"versionCount,omitempty"`
	Versions       []PipelineConfigVersion `json:"versions,omitempty"`
	CreatedAt      time.Time               `json:"createdAt"`
	UpdatedAt      time.Time               `json:"updatedAt"`
}

// PipelineConfigVersion is an immutable snapshot of a config file.
type PipelineConfigVersion struct {
	ID               string    `json:"id"`
	ConfigID         string    `json:"configId"`
	Version          int       `json:"version"`
	Status           string    `json:"status"`
	Content          string    `json:"content,omitempty"`
	ContentSHA256    string    `json:"contentSha256"`
	ContentSizeBytes int       `json:"contentSizeBytes"`
	Summary          string    `json:"summary"`
	Author           string    `json:"author"`
	CreatedAt        time.Time `json:"createdAt"`
}
