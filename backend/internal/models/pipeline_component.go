package models

import "time"

// PipelineComponent represents a registered pipeline component (Docker image).
type PipelineComponent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Image       string                 `json:"image"`
	Tag         string                 `json:"tag"`
	Source      string                 `json:"source"`
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
