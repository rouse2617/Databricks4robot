package workflow

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
)

// GetNodePodDiagnostics handles GET /api/v1/workflows/:name/nodes/:nodeId/pod.
func (h *Handler) GetNodePodDiagnostics(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeID := strings.TrimSpace(c.Param("nodeId"))
	if name == "" || nodeID == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow name and nodeId are required", nil)
		return
	}

	namespace := h.namespaceForWorkflow(c.Request.Context(), c, name)
	wf, err := h.wfClient.GetWorkflow(c.Request.Context(), name, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			httpresp.NotFound(c, "WORKFLOW_NOT_FOUND", err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	podName, ok := resolveWorkflowPodName(wf, nodeID)
	if !ok {
		httpresp.NotFound(c, "NODE_NOT_FOUND", "node "+nodeID+" not found or does not resolve to a pod")
		return
	}

	if h.podClient == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "K8S_UNAVAILABLE", "Kubernetes Pod diagnostics are not configured", nil)
		return
	}

	diag, err := h.podClient.GetPodDiagnostics(c.Request.Context(), namespace, podName)
	if err != nil {
		switch {
		case apierrors.IsNotFound(err):
			httpresp.NotFound(c, "POD_NOT_FOUND", "pod "+podName+" was not found")
		case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
			httpresp.Error(c, http.StatusForbidden, "K8S_FORBIDDEN", "Kubernetes credentials cannot read pod diagnostics", nil)
		case errors.Is(err, k8s.ErrUnavailable):
			httpresp.Error(c, http.StatusServiceUnavailable, "K8S_UNAVAILABLE", err.Error(), nil)
		default:
			httpresp.Error(c, http.StatusServiceUnavailable, "K8S_UNAVAILABLE", err.Error(), nil)
		}
		return
	}

	c.JSON(200, diag)
}
