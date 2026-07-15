package models

import "time"

// Cluster is a K8s execution destination that DataBrew can dispatch pipelines
// into. Historically the backend assumed a single cluster via env vars
// (K8S_API_ENDPOINT / K8S_AUDIENCE / ARGO_SERVER_URL, etc); this row-based
// model is CYB-3425's Phase B storage layer that lets one deployment manage
// many clusters at once.
//
// Blank connection fields fall back to env at runtime (default cluster
// compatibility). Non-blank fields take precedence so each cluster can carry
// its own WIF audience / CA / Argo URL.
type Cluster struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	DisplayName    string     `json:"displayName"`
	Description    string     `json:"description,omitempty"`
	IsDefault      bool       `json:"isDefault"`
	Status         string     `json:"status"`
	K8sAPIEndpoint string     `json:"k8sApiEndpoint,omitempty"`
	K8sAudience    string     `json:"k8sAudience,omitempty"`
	K8sCAData      string     `json:"k8sCaData,omitempty"`
	ArgoServerURL  string     `json:"argoServerUrl,omitempty"`
	ArgoNamespace  string     `json:"argoNamespace,omitempty"`
	KoordInstalled bool       `json:"koordInstalled"`
	CreatedAt      time.Time  `json:"createdAt,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt,omitempty"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}
