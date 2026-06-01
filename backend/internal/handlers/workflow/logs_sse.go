package workflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

const defaultWorkflowLogContainer = "main"

type workflowLogStreamEntry struct {
	Result struct {
		Content string `json:"content"`
		PodName string `json:"podName"`
	} `json:"result"`
}

func extractLogLine(raw string) (string, bool) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "data:"))
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	var entry workflowLogStreamEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return raw, true
	}
	if entry.Result.Content == "" {
		return "", false
	}
	content := strings.TrimSuffix(entry.Result.Content, "\n")
	return strings.TrimSpace(content), true
}

func writeWorkflowLogSSE(writer io.Writer, line string) bool {
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", line); err != nil {
		return false
	}
	return true
}

func streamWorkflowLogs(
	c *gin.Context,
	writer io.Writer,
	stream io.Reader,
) bool {
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		if c.Request.Context().Err() != nil {
			return false
		}
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}

		content, ok := extractLogLine(raw)
		if !ok {
			continue
		}
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSuffix(line, "\r")
			if line == "" {
				continue
			}
			if !writeWorkflowLogSSE(writer, line) {
				return false
			}
		}
	}
	return scanner.Err() == nil
}

func streamWorkflowLogsText(writer io.Writer, logs string) bool {
	hasAny := false
	for _, line := range strings.Split(strings.TrimSuffix(logs, "\n"), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		hasAny = true
		if !writeWorkflowLogSSE(writer, line) {
			return false
		}
	}
	return hasAny
}

// StreamWorkflowLogs handles GET /api/v1/workflows/:name/log/stream?nodeId=xxx
// This is an SSE endpoint that streams Argo workflow pod logs.
func (h *Handler) StreamWorkflowLogs(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeID := strings.TrimSpace(c.Query("nodeId"))
	if name == "" || nodeID == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow name and nodeId are required", nil)
		return
	}

	namespace := h.namespaceFor(c)

	workflow, err := h.wfClient.GetWorkflow(c.Request.Context(), name, namespace)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	podName, ok := resolveWorkflowPodName(workflow, nodeID)
	if !ok {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow pod node not found", nil)
		return
	}

	stream, err := h.wfClient.GetWorkflowLogStream(
		c.Request.Context(),
		name,
		podName,
		defaultWorkflowLogContainer,
		namespace,
	)
	if err != nil {
		logs, fallbackErr := h.wfClient.GetWorkflowLogs(
			c.Request.Context(),
			name,
			podName,
			namespace,
		)
		if fallbackErr != nil {
			httpresp.Internal(c, fallbackErr.Error())
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		c.Stream(func(writer io.Writer) bool {
			return streamWorkflowLogsText(writer, logs)
		})
		return
	}
	defer stream.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Stream(func(writer io.Writer) bool {
		return streamWorkflowLogs(c, writer, stream)
	})
}
