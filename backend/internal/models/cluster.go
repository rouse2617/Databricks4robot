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
	ID             string `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description,omitempty"`
	IsDefault      bool   `json:"isDefault"`
	Status         string `json:"status"`
	K8sAPIEndpoint string `json:"k8sApiEndpoint,omitempty"`
	K8sAudience    string `json:"k8sAudience,omitempty"`
	K8sCAData      string `json:"k8sCaData,omitempty"`
	// AuthType selects how to authenticate to the cluster's K8s API.
	// CYB-3486 auth.1: default "gke_wif" keeps every existing row on the
	// current GKE metadata-token path. Reserved future values (not yet
	// consumed by the factory): "bearer" (static token from Secret Manager),
	// "ack_wif" / "eks_wif" (equivalent WIF paths on Alibaba / AWS).
	AuthType string `json:"authType,omitempty"`
	// AuthSecretRef is the Secret Manager / external secret reference for
	// non-WIF auth. Empty for the default "gke_wif" path.
	AuthSecretRef  string `json:"authSecretRef,omitempty"`
	ArgoServerURL  string `json:"argoServerUrl,omitempty"`
	ArgoNamespace  string `json:"argoNamespace,omitempty"`
	KoordInstalled bool   `json:"koordInstalled"`
	// ClientQPS / ClientBurst cap the backend's K8s client request rate to this
	// cluster's API (rest.Config QPS/Burst). Editable online from the cluster
	// admin UI; the ClientFactory rebuilds this cluster's clients on update, so
	// changes apply within seconds without a restart. Zero → factory default
	// (50 / 100), which is a safe lift over client-go's throttling defaults
	// (5 / 10) that otherwise cap batch submission throughput. CYB-3486.
	ClientQPS   float32    `json:"clientQps"`
	ClientBurst int        `json:"clientBurst"`
	CreatedAt   time.Time  `json:"createdAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt,omitempty"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// DefaultClientQPS / DefaultClientBurst are the effective K8s client rate limits
// applied when a cluster leaves ClientQPS / ClientBurst unset (<=0). They lift
// client-go's throttling defaults (5 / 10) — the ceiling that otherwise caps
// batch submission throughput — to a batch-friendly level. CYB-3486.
const (
	DefaultClientQPS   float32 = 50
	DefaultClientBurst int     = 100
)

// ResolveClientLimits returns the effective K8s client QPS / Burst for this
// cluster, substituting the package defaults for unset (<=0) values. Both the
// postgres repo (persisted rows) and the k8s ClientFactory (rest.Config build)
// route through here so the two never drift.
func (c *Cluster) ResolveClientLimits() (float32, int) {
	qps := c.ClientQPS
	if qps <= 0 {
		qps = DefaultClientQPS
	}
	burst := c.ClientBurst
	if burst <= 0 {
		burst = DefaultClientBurst
	}
	return qps, burst
}
