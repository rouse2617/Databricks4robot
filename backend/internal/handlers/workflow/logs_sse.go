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

func writeWorkflowSSEEvent(writer io.Writer, event string, payload any) bool {
	raw, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event, raw); err != nil {
		return false
	}
	return true
}

func streamWorkflowLogs(
	c *gin.Context,
	writer io.Writer,
	stream io.Reader,
	podName string,
	container string,
	limitBytes int64,
) bool {
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), int(maxWorkflowLogLimitBytes))

	if !writeWorkflowSSEEvent(writer, "heartbeat", gin.H{}) {
		return false
	}

	var emittedBytes int64
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
			lineBytes := int64(len(line))
			if limitBytes >= 0 && emittedBytes+lineBytes > limitBytes {
				remaining := limitBytes - emittedBytes
				if remaining > 0 {
					line = line[:remaining]
					if !writeWorkflowSSEEvent(writer, "log", gin.H{
						"podName":    podName,
						"container":  container,
						"line":       line,
						"truncated":  true,
						"limitBytes": limitBytes,
					}) {
						return false
					}
				}
				_ = writeWorkflowSSEEvent(writer, "end", gin.H{"reason": "limit-bytes"})
				return false
			}
			emittedBytes += lineBytes
			if !writeWorkflowSSEEvent(writer, "log", gin.H{
				"podName":   podName,
				"container": container,
				"line":      line,
			}) {
				return false
			}
		}
	}
	if scanner.Err() != nil {
		_ = writeWorkflowSSEEvent(writer, "end", gin.H{"reason": "stream-error"})
		return false
	}
	_ = writeWorkflowSSEEvent(writer, "end", gin.H{"reason": "stream-complete"})
	return false
}

// StreamWorkflowLogs handles GET /api/v1/workflows/:name/logs/stream?nodeId=xxx
// This is an SSE endpoint that streams Argo workflow pod logs.
func (h *Handler) StreamWorkflowLogs(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeID := strings.TrimSpace(c.Query("nodeId"))
	if name == "" || nodeID == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow name and nodeId are required", nil)
		return
	}

	namespace := h.namespaceForWorkflow(c.Request.Context(), c, name)

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

	opts, _, ok := parseWorkflowLogOptions(c)
	if !ok {
		return
	}

	stream, err := h.wfClient.GetWorkflowLogStream(
		c.Request.Context(),
		name,
		podName,
		namespace,
		opts,
	)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	defer stream.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Stream(func(writer io.Writer) bool {
		return streamWorkflowLogs(c, writer, stream, podName, opts.Container, *opts.LimitBytes)
	})
}
