package models

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"
)

// PipelineComponent represents a registered pipeline component (Docker image).
type PipelineComponent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Image       string                 `json:"image"`
	Tag         string                 `json:"tag"`
	Source      string                 `json:"source"`
	Scope       string                 `json:"scope,omitempty"`
	Owner       string                 `json:"owner,omitempty"`
	Command     []string               `json:"command,omitempty"`
	Args        []string               `json:"args,omitempty"`
	Env         map[string]string      `json:"env,omitempty"`
	InputPorts  []PortDef              `json:"inputPorts"`
	OutputPorts []PortDef              `json:"outputPorts"`
	Resources   map[string]interface{} `json:"resources,omitempty"`
	EnvVars     []EnvVarDef            `json:"envVars,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// PortDef defines an input or output port for a component.
type PortDef struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Desc         string `json:"desc,omitempty"`
	DefaultValue string `json:"default_value,omitempty"` // default value for input ports (F2.10)
}

// EnvVarDef defines an environment variable for a component.
type EnvVarDef struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// ComponentReleaseRuntimeSnapshot captures the immutable runtime contract used
// when a pipeline node selects a generated component release.
type ComponentReleaseRuntimeSnapshot struct {
	Image       string                 `json:"image"`
	Command     []string               `json:"command,omitempty"`
	Args        []string               `json:"args,omitempty"`
	Env         map[string]string      `json:"env,omitempty"`
	InputPorts  []PortDef              `json:"inputPorts"`
	OutputPorts []PortDef              `json:"outputPorts"`
	Resources   map[string]interface{} `json:"resources,omitempty"`
}

// ComponentReleaseIngestSource captures CI/build context shared by a release
// manifest. Individual release items can still override these fields.
type ComponentReleaseIngestSource struct {
	Provider string `json:"provider,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Ref      string `json:"ref,omitempty"`
	RefType  string `json:"refType,omitempty"`
	Commit   string `json:"commit,omitempty"`
	BuildID  string `json:"buildId,omitempty"`
	Trigger  string `json:"trigger,omitempty"`
}

// ComponentReleaseIngestManifest is the CI-facing contract for publishing one
// or more generated component releases into DataBrew.
type ComponentReleaseIngestManifest struct {
	Source ComponentReleaseIngestSource `json:"source,omitempty"`
	Items  []PipelineComponentRelease   `json:"items"`
}

// PipelineComponentRelease is a generated, validated version of an algorithm
// task. Normal users select releases rather than authoring image/digest fields.
type PipelineComponentRelease struct {
	ID                string                          `json:"id"`
	ImageUID          string                          `json:"imageUid,omitempty"`
	ComponentID       string                          `json:"componentId"`
	TaskName          string                          `json:"taskName"`
	TaskPath          string                          `json:"taskPath,omitempty"`
	DisplayName       string                          `json:"displayName,omitempty"`
	Owner             string                          `json:"owner,omitempty"`
	ReleaseLabel      string                          `json:"releaseLabel"`
	Channel           string                          `json:"channel"`
	SourceRepo        string                          `json:"sourceRepo,omitempty"`
	SourceRef         string                          `json:"sourceRef,omitempty"`
	SourceRefType     string                          `json:"sourceRefType,omitempty"`
	SourceCommit      string                          `json:"sourceCommit,omitempty"`
	BuildID           string                          `json:"buildId,omitempty"`
	ImageRepo         string                          `json:"imageRepo,omitempty"`
	ImageTag          string                          `json:"imageTag,omitempty"`
	ImageDigest       string                          `json:"imageDigest,omitempty"`
	RuntimeImage      string                          `json:"runtimeImage"`
	Status            string                          `json:"status"`
	Selectable        bool                            `json:"selectable"`
	ValidationStatus  string                          `json:"validationStatus"`
	ValidationErrors  []string                        `json:"validationErrors,omitempty"`
	RuntimeSnapshot   ComponentReleaseRuntimeSnapshot `json:"runtimeSnapshot"`
	TechnicalMetadata map[string]interface{}          `json:"technicalMetadata,omitempty"`
	CreatedAt         time.Time                       `json:"createdAt"`
	UpdatedAt         time.Time                       `json:"updatedAt"`
	LastSyncedAt      *time.Time                      `json:"lastSyncedAt,omitempty"`
}

// ShortImageUID returns a stable 8-character display ID for a released image.
func ShortImageUID(identity string) string {
	identity = strings.ToLower(strings.TrimSpace(identity))
	if identity == "" {
		return ""
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(identity))
	return fmt.Sprintf("%08x", h.Sum32())
}
