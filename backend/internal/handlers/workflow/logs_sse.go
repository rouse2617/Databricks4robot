package workflow

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

const defaultWorkflowLogContainer = "main"

const (
	sseRingBufferSize    = 1000
	sseRingBufferMaxAge  = 5 * time.Minute
	sseRingBufferCleanup = 1 * time.Minute
)

type sseEvent struct {
	id   int64
	data string // full SSE frame: "event: log\ndata: {...}\n\n"
}

type logRingBuffer struct {
	mu        sync.Mutex
	buffer    []sseEvent
	nextID    int64
	lastWrite time.Time
	maxSize   int
}

func newLogRingBuffer(maxSize int) *logRingBuffer {
	return &logRingBuffer{
		buffer:    make([]sseEvent, 0, maxSize),
		maxSize:   maxSize,
		lastWrite: time.Now(),
	}
}

func (rb *logRingBuffer) push(event sseEvent) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	event.id = rb.nextID
	rb.nextID++
	rb.lastWrite = time.Now()
	if len(rb.buffer) >= rb.maxSize {
		rb.buffer = rb.buffer[1:]
	}
	rb.buffer = append(rb.buffer, event)
}

func (rb *logRingBuffer) replayAfter(lastID int64) []sseEvent {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	var result []sseEvent
	for _, ev := range rb.buffer {
		if ev.id > lastID {
			result = append(result, ev)
		}
	}
	return result
}

func (rb *logRingBuffer) age() time.Duration {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return time.Since(rb.lastWrite)
}

// ringBufferStore holds ring buffers for active log streams.
type ringBufferStore struct {
	mu     sync.Mutex
	buffers map[string]*logRingBuffer
}

func newRingBufferStore() *ringBufferStore {
	s := &ringBufferStore{
		buffers: make(map[string]*logRingBuffer),
	}
	go s.cleanupLoop()
	return s
}

func (s *ringBufferStore) getOrCreate(key string) *logRingBuffer {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rb, ok := s.buffers[key]; ok {
		return rb
	}
	rb := newLogRingBuffer(sseRingBufferSize)
	s.buffers[key] = rb
	return rb
}

func (s *ringBufferStore) remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buffers, key)
}

func (s *ringBufferStore) cleanupLoop() {
	for {
		time.Sleep(sseRingBufferCleanup)
		s.mu.Lock()
		for key, rb := range s.buffers {
			if rb.age() > sseRingBufferMaxAge {
				delete(s.buffers, key)
			}
		}
		s.mu.Unlock()
	}
}

func writeWorkflowSSEEventRaw(writer io.Writer, id int64, event string, payload any) bool {
	raw, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(writer, "id: %d\nevent: %s\ndata: %s\n\n", id, event, raw); err != nil {
		return false
	}
	return true
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

type sseLineEmitter struct {
	writer  io.Writer
	podName string
	ring    *logRingBuffer
	scanID  int64
}

func (e *sseLineEmitter) emit(container string, line string, truncated bool, limitBytes int64) bool {
	e.scanID++
	payload := gin.H{
		"podName":   e.podName,
		"container": container,
		"line":      line,
	}
	if truncated {
		payload["truncated"] = true
		payload["limitBytes"] = limitBytes
	}
	frame := formatSSEFrame(e.scanID, "log", payload)
	if e.ring != nil {
		e.ring.push(sseEvent{data: frame})
	}
	if _, err := fmt.Fprint(e.writer, frame); err != nil {
		return false
	}
	return true
}

func formatSSEFrame(id int64, event string, payload any) string {
	raw, _ := json.Marshal(payload)
	return fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", id, event, raw)
}

func streamWorkflowLogs(
	c *gin.Context,
	writer io.Writer,
	stream io.Reader,
	podName string,
	container string,
	limitBytes int64,
	ring *logRingBuffer,
) bool {
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), int(maxWorkflowLogLimitBytes))

	emitter := &sseLineEmitter{
		writer:  writer,
		podName: podName,
		ring:    ring,
	}

	// Initial heartbeat
	frame := formatSSEFrame(emitter.scanID, "heartbeat", gin.H{})
	emitter.scanID++
	if ring != nil {
		ring.push(sseEvent{data: frame})
	}
	if _, err := fmt.Fprint(writer, frame); err != nil {
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
					if !emitter.emit(container, line, true, limitBytes) {
						return false
					}
				}
				frame := formatSSEFrame(emitter.scanID+1, "end", gin.H{"reason": "limit-bytes"})
				fmt.Fprint(writer, frame)
				return false
			}
			emittedBytes += lineBytes
			if !emitter.emit(container, line, false, 0) {
				return false
			}
		}
	}
	if scanner.Err() != nil {
		frame := formatSSEFrame(emitter.scanID+1, "end", gin.H{"reason": "stream-error"})
		fmt.Fprint(writer, frame)
		return false
	}
	frame  = formatSSEFrame(emitter.scanID+1, "end", gin.H{"reason": "stream-complete"})
	fmt.Fprint(writer, frame)
	return false
}

// writeGracefulLogStreamEnd opens the SSE response and emits a single terminal
// "end" event carrying reason, then returns. Used when a node has no live logs
// to stream — the workflow was TTL'd/GC'd, its pod was recycled, or the pod has
// not been created yet. The browser's EventSource treats an "end" event as
// normal completion and closes without reconnecting; a 4xx/5xx instead fires
// onerror and reconnects with exponential backoff, hammering this endpoint in a
// loop (the reconnect storm the non-stream /logs handler already avoids via
// CYB-3568/3575). This mirrors that handler's graceful branches for the SSE
// variant (CYB-3579).
func writeGracefulLogStreamEnd(c *gin.Context, reason string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	fmt.Fprint(c.Writer, formatSSEFrame(1, "end", gin.H{"reason": reason}))
}

// StreamWorkflowLogs handles GET /api/v1/workflows/:name/logs/stream?nodeId=xxx
// This is an SSE endpoint that streams Argo workflow pod logs with id-based
// sequencing for reconnection support.
func (h *Handler) StreamWorkflowLogs(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeID := strings.TrimSpace(c.Query("nodeId"))
	if name == "" || nodeID == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow name and nodeId are required", nil)
		return
	}

	client, namespace := h.resolveWorkflowRouting(c.Request.Context(), c, name)

	workflow, err := client.GetWorkflow(c.Request.Context(), name, namespace)
	if err != nil {
		// A TTL'd/GC'd workflow is genuinely gone — end the SSE stream cleanly
		// so EventSource stops instead of reconnecting on a 4xx. Genuine faults
		// (argo-server down, RBAC) still 500 so the client retries and #462
		// logs them server-side.
		if errors.Is(err, argo.ErrNotFound) {
			writeGracefulLogStreamEnd(c, "workflow-gone")
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	podName, ok := resolveCachedWorkflowPodName(workflow, nodeID)
	if !ok {
		// A Pending / not-yet-scheduled node has no pod yet (Status.Nodes still
		// empty, or the node is known but its pod hasn't materialized). End the
		// stream cleanly so the viewer shows "已结束" and EventSource stops,
		// rather than a 4xx reconnect loop (mirrors GetWorkflowLogs, CYB-3575).
		// A genuinely unknown node id on a live workflow still 400s.
		phase := string(workflow.Status.Phase)
		_, nodeKnown := workflow.Status.Nodes[nodeID]
		if phase == "" || phase == "Pending" || nodeKnown {
			writeGracefulLogStreamEnd(c, "pod-not-created")
			return
		}
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow pod node not found", nil)
		return
	}

	opts, _, ok := parseWorkflowLogOptions(c)
	if !ok {
		return
	}

	// Only live-tail (follow) a node that is still running. Following a finished
	// node never receives new lines and never EOFs through Argo's follow API, so
	// the SSE request would hang until the Cloud Run request timeout (~60s) → 504,
	// and the client auto-reconnects, starving the browser's per-host connection
	// pool. For a fulfilled node, fetch once (Follow=false): Argo returns the
	// existing logs then EOF, and the stream emits its "end" event and closes in
	// ~1-2s. (CYB-3483)
	node, hasNode := workflow.Status.Nodes[nodeID]
	opts.Follow = hasNode && !node.Fulfilled()

	stream, err := client.GetWorkflowLogStream(
		c.Request.Context(),
		name,
		podName,
		namespace,
		opts,
	)
	if err != nil {
		// Pod recycled after the workflow object survived (CRD mode: k8s GetLogs
		// → IsNotFound → argo.ErrNotFound). End the stream cleanly instead of a
		// 500 that makes EventSource reconnect in a loop — the reconnect storm
		// CYB-3579 fixes. Genuine faults (RBAC, API connectivity) still 500.
		if errors.Is(err, argo.ErrNotFound) {
			writeGracefulLogStreamEnd(c, "pod-recycled")
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	defer stream.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Get or create ring buffer for reconnection
	ringKey := name + "/" + nodeID + "/" + opts.Container
	ring := h.sseRingBuffers.getOrCreate(ringKey)

	// Handle Last-Event-ID for reconnection
	lastEventID := c.Query("lastEventId")
	if lastEventID == "" {
		lastEventID = c.GetHeader("Last-Event-ID")
	}
	if lastEventID != "" {
		if sinceID, err := strconv.ParseInt(lastEventID, 10, 64); err == nil {
			events := ring.replayAfter(sinceID)
			for _, ev := range events {
				if _, writeErr := fmt.Fprint(c.Writer, ev.data); writeErr != nil {
					return
				}
			}
		}
	}

	c.Stream(func(writer io.Writer) bool {
		return streamWorkflowLogs(c, writer, stream, podName, opts.Container, *opts.LimitBytes, ring)
	})

	// Stream ended — clean up ring buffer after a grace period
	go func() {
		time.Sleep(sseRingBufferMaxAge)
		h.sseRingBuffers.remove(ringKey)
	}()
}
