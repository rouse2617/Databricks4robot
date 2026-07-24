package models

// PipelinePromotionDependency is a portable component reference captured by a
// promotion plan. RuntimeImage must be digest-pinned before Ready can be true.
type PipelinePromotionDependency struct {
	NodeName              string `json:"nodeName"`
	ComponentID           string `json:"componentId,omitempty"`
	ReleaseID             string `json:"releaseId,omitempty"`
	ComponentVersionLabel string `json:"componentVersionLabel,omitempty"`
	RuntimeImage          string `json:"runtimeImage"`
}

// PipelinePromotionMappingRequirement describes an environment-bound resource
// that prod must resolve.
type PipelinePromotionMappingRequirement struct {
	Kind       string `json:"kind"`
	SourceID   string `json:"sourceId"`
	TargetID   string `json:"targetId,omitempty"`
	NodeName   string `json:"nodeName,omitempty"`
	MountPath  string `json:"mountPath,omitempty"`
	Required   bool   `json:"required"`
	Resolution string `json:"resolution,omitempty"`
}

// PipelinePromotionBundle is the content-addressed snapshot of a dev template
// and its dependencies captured during plan generation.
type PipelinePromotionBundle struct {
	SourceEnvironment string                        `json:"sourceEnvironment"`
	SourceTemplateID  string                        `json:"sourceTemplateId"`
	SourceVersion     int                           `json:"sourceVersion"`
	Name              string                        `json:"name"`
	Owner             string                        `json:"owner,omitempty"`
	Pipeline          map[string]interface{}        `json:"pipeline"`
	Dependencies      []PipelinePromotionDependency `json:"dependencies"`
	BundleDigest      string                        `json:"bundleDigest"`
}

// PipelinePromotionPlan is the read-only preview returned before execution.
type PipelinePromotionPlan struct {
	TargetEnvironment string                                `json:"targetEnvironment"`
	PlanDigest        string                                `json:"planDigest"`
	Bundle            PipelinePromotionBundle               `json:"bundle"`
	RequiredMappings  []PipelinePromotionMappingRequirement `json:"requiredMappings"`
	Warnings          []string                              `json:"warnings"`
	Blockers          []string                              `json:"blockers"`
	Ready             bool                                  `json:"ready"`
}
