package workflow

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

const (
	defaultTerminalMaxSessionSeconds = 900
	defaultTerminalIdleSeconds       = 120
	defaultTerminalAttachTTLSeconds  = 60
)

type terminalPolicy struct {
	Enabled            bool
	MaxSessionSeconds  int
	IdleTimeoutSeconds int
	AllowedCommands    []string
	DeniedPatterns     []string
	AllowCompletedPods bool
}

type terminalSession struct {
	ID                string     `json:"id"`
	RunID             string     `json:"runId,omitempty"`
	WorkflowName      string     `json:"workflowName"`
	NodeID            string     `json:"nodeId"`
	PodName           string     `json:"podName"`
	ContainerName     string     `json:"containerName,omitempty"`
	ExecutionTargetID string     `json:"executionTargetId,omitempty"`
	Cluster           string     `json:"cluster,omitempty"`
	Namespace         string     `json:"namespace"`
	Command           string     `json:"command"`
	Status            string     `json:"status"`
	AttachURL         string     `json:"attachUrl,omitempty"`
	ExpiresAt         time.Time  `json:"expiresAt"`
	CreatedAt         time.Time  `json:"createdAt"`
	AttachedAt        *time.Time `json:"attachedAt,omitempty"`
	EndedAt           *time.Time `json:"endedAt,omitempty"`
	ErrorCode         string     `json:"errorCode,omitempty"`
	ErrorMessage      string     `json:"errorMessage,omitempty"`

	attachToken string
	attachUsed  bool
}

type terminalSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*terminalSession
}

func newTerminalSessionStore() *terminalSessionStore {
	store := &terminalSessionStore{sessions: map[string]*terminalSession{}}
	go store.janitor(5 * time.Minute)
	return store
}

func (s *terminalSessionStore) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, sess := range s.sessions {
			if sess.EndedAt != nil || now.After(sess.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func (s *terminalSessionStore) save(session *terminalSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *session
	s.sessions[session.ID] = &cp
}

func (s *terminalSessionStore) get(id string) (*terminalSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	cp := *session
	return &cp, true
}

func (s *terminalSessionStore) update(id string, fn func(*terminalSession)) (*terminalSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	fn(session)
	cp := *session
	return &cp, true
}

type createTerminalSessionRequest struct {
	ContainerName string `json:"containerName"`
	Command       string `json:"command"`
}

type terminalWebSocketWriter struct {
	mu sync.Mutex
	rw *bufio.ReadWriter
}

func (w *terminalWebSocketWriter) Write(payload []byte) (int, error) {
	w.writeFrame(map[string]any{
		"type": "stdout",
		"data": string(payload),
	})
	return len(payload), nil
}

func (w *terminalWebSocketWriter) WriteStderr(payload []byte) (int, error) {
	w.writeFrame(map[string]any{
		"type": "stderr",
		"data": string(payload),
	})
	return len(payload), nil
}

func (w *terminalWebSocketWriter) writeFrame(frame map[string]any) {
	data, err := json.Marshal(frame)
	if err != nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	writeWebSocketText(w.rw, string(data))
	_ = w.rw.Flush()
}

// CreateTerminalSession handles POST /api/v1/workflows/:name/nodes/:nodeId/terminal-sessions.
func (h *Handler) CreateTerminalSession(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeID := strings.TrimSpace(c.Param("nodeId"))
	if name == "" || nodeID == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow name and nodeId are required", nil)
		return
	}

	var req createTerminalSessionRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			httpresp.BadRequest(c, "INVALID_ARGUMENT", "invalid terminal session request body", map[string]any{
				"error": err.Error(),
			})
			return
		}
	}
	command := strings.TrimSpace(req.Command)
	if command == "" {
		command = "sh"
	}

	namespace := h.namespaceFor(c)
	wf, err := h.wfClient.GetWorkflow(c.Request.Context(), name, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			httpresp.NotFound(c, "WORKFLOW_NOT_FOUND", err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	node, ok := wf.Status.Nodes[nodeID]
	if !ok {
		httpresp.NotFound(c, "NODE_NOT_FOUND", "node "+nodeID+" was not found")
		return
	}
	podName, ok := resolveWorkflowPodName(wf, nodeID)
	if !ok {
		httpresp.NotFound(c, "POD_NOT_FOUND", "node "+nodeID+" does not resolve to a pod")
		return
	}

	run, _ := h.findPipelineRunByWorkflow(c.Request.Context(), name)
	policy := terminalPolicyFromRun(run)
	if !policy.Enabled {
		httpresp.Error(c, http.StatusForbidden, "POD_EXEC_FORBIDDEN", "Pod 终端未启用，请在执行目标配置中开启", nil)
		return
	}
	if !policy.AllowCompletedPods && !isWorkflowNodeRunning(node) {
		httpresp.Error(c, http.StatusConflict, "POD_EXEC_UNAVAILABLE", "Pod 已完成，终端仅支持运行中的 Pod", nil)
		return
	}
	if err := validateTerminalCommand(policy, command); err != nil {
		httpresp.Error(c, http.StatusForbidden, "POD_EXEC_FORBIDDEN", err.Error(), nil)
		h.appendTerminalRunEvent(c.Request.Context(), run, "pod_terminal_session_failed", "failed", nodeID, podName, command, err.Error())
		return
	}

	now := h.now()
	maxSeconds := policy.MaxSessionSeconds
	if maxSeconds <= 0 {
		maxSeconds = defaultTerminalMaxSessionSeconds
	}
	token, err := randomToken()
	if err != nil {
		httpresp.Internal(c, "create attach token: "+err.Error())
		return
	}
	session := &terminalSession{
		ID:                uuid.NewString(),
		WorkflowName:      name,
		NodeID:            nodeID,
		PodName:           podName,
		ContainerName:     strings.TrimSpace(req.ContainerName),
		Namespace:         namespace,
		Command:           command,
		Status:            "created",
		ExpiresAt:         now.Add(time.Duration(maxSeconds) * time.Second),
		CreatedAt:         now,
		attachToken:       token,
		ExecutionTargetID: executionTargetIDFromRun(run),
	}
	if run != nil {
		session.RunID = run.ID
		session.Cluster = targetString(run.TargetSnapshot, "cluster")
		if targetNS := targetString(run.TargetSnapshot, "namespace"); targetNS != "" {
			session.Namespace = targetNS
		}
	}
	session.AttachURL = fmt.Sprintf("/api/v1/pod-terminal/sessions/%s/attach?token=%s", session.ID, token)
	h.terminalStore.save(session)
	h.appendTerminalRunEvent(c.Request.Context(), run, "pod_terminal_session_created", "created", nodeID, podName, command, "")

	c.JSON(http.StatusCreated, sanitizeTerminalSession(session))
}

// GetTerminalSession handles GET /api/v1/pod-terminal/sessions/:id.
func (h *Handler) GetTerminalSession(c *gin.Context) {
	session, ok := h.terminalStore.get(strings.TrimSpace(c.Param("id")))
	if !ok {
		httpresp.NotFound(c, "POD_TERMINAL_SESSION_NOT_FOUND", "terminal session not found")
		return
	}
	c.JSON(http.StatusOK, sanitizeTerminalSession(session))
}

// TerminateTerminalSession handles POST /api/v1/pod-terminal/sessions/:id/terminate.
func (h *Handler) TerminateTerminalSession(c *gin.Context) {
	session, ok := h.terminalStore.update(strings.TrimSpace(c.Param("id")), func(s *terminalSession) {
		now := h.now()
		s.Status = "terminated"
		s.EndedAt = &now
	})
	if !ok {
		httpresp.NotFound(c, "POD_TERMINAL_SESSION_NOT_FOUND", "terminal session not found")
		return
	}
	if run, _ := h.findPipelineRunByWorkflow(c.Request.Context(), session.WorkflowName); run != nil {
		h.appendTerminalRunEvent(c.Request.Context(), run, "pod_terminal_session_ended", "terminated", session.NodeID, session.PodName, session.Command, "")
	}
	c.JSON(http.StatusOK, sanitizeTerminalSession(session))
}

// AttachTerminalSession handles GET /api/v1/pod-terminal/sessions/:id/attach.
//
// This first slice performs the backend-owned WebSocket/session handshake and
// returns a controlled unavailable frame until the Kubernetes exec stream is
// enabled for the target. The browser still never receives Kubernetes
// credentials, and attach tokens remain scoped and single-use.
func (h *Handler) AttachTerminalSession(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	token := strings.TrimSpace(c.Query("token"))
	var attachValid bool
	session, ok := h.terminalStore.update(id, func(s *terminalSession) {
		now := h.now()
		if s.attachToken == token && !s.attachUsed && now.Before(s.ExpiresAt) {
			s.attachUsed = true
			s.Status = "attached"
			s.AttachedAt = &now
			attachValid = true
		}
	})
	if !ok {
		httpresp.NotFound(c, "POD_TERMINAL_SESSION_NOT_FOUND", "terminal session not found")
		return
	}
	if token == "" || token != session.attachToken || !attachValid {
		httpresp.Error(c, http.StatusForbidden, "POD_EXEC_FORBIDDEN", "terminal attach token is invalid, expired, or already used", nil)
		return
	}

	conn, rw, ok := hijackWebSocket(c.Writer, c.Request)
	if !ok {
		return
	}
	defer conn.Close()

	writer := &terminalWebSocketWriter{rw: rw}
	writer.writeFrame(map[string]any{"type": "status", "status": "attached"})

	// Detect client disconnect: read from hijacked connection in a goroutine
	// and cancel the exec context if the client hangs up. Without this the
	// K8s exec stream would run until session.ExpiresAt.
	execCtx, execCancel := context.WithDeadline(context.Background(), session.ExpiresAt)
	defer execCancel()
	go func() {
		buf := make([]byte, 32)
		for {
			if _, err := conn.Read(buf); err != nil {
				execCancel()
				return
			}
		}
	}()

	if h.execClient == nil {
		session = h.failTerminalSession(context.Background(), id, "POD_EXEC_UNAVAILABLE", "Kubernetes exec client is not configured")
		writer.writeFrame(map[string]any{
			"type":    "error",
			"code":    session.ErrorCode,
			"message": session.ErrorMessage,
		})
		writer.writeFrame(map[string]any{"type": "exit", "exitCode": -1, "reason": "exec_unavailable"})
		return
	}

	execCtx, cancel := context.WithDeadline(context.Background(), session.ExpiresAt)
	defer cancel()
	err := h.execClient.ExecPod(execCtx, k8s.PodExecRequest{
		Namespace:     session.Namespace,
		PodName:       session.PodName,
		ContainerName: session.ContainerName,
		Command:       terminalCommandArgs(session.Command),
		TTY:           false,
	}, writer, stderrWriter{writer})
	if err != nil {
		session = h.failTerminalSession(context.Background(), id, "POD_EXEC_FAILED", err.Error())
		writer.writeFrame(map[string]any{
			"type":    "error",
			"code":    session.ErrorCode,
			"message": session.ErrorMessage,
		})
		writer.writeFrame(map[string]any{"type": "exit", "exitCode": 1, "reason": "exec_failed"})
		return
	}

	session, _ = h.terminalStore.update(id, func(s *terminalSession) {
		now := h.now()
		s.Status = "ended"
		s.EndedAt = &now
	})
	if run, _ := h.findPipelineRunByWorkflow(context.Background(), session.WorkflowName); run != nil {
		h.appendTerminalRunEvent(context.Background(), run, "pod_terminal_session_ended", "ended", session.NodeID, session.PodName, session.Command, "")
	}
	writer.writeFrame(map[string]any{"type": "exit", "exitCode": 0, "reason": "completed"})
}

type stderrWriter struct {
	writer *terminalWebSocketWriter
}

func (w stderrWriter) Write(payload []byte) (int, error) {
	return w.writer.WriteStderr(payload)
}

func (h *Handler) failTerminalSession(ctx context.Context, id, code, message string) *terminalSession {
	session, _ := h.terminalStore.update(id, func(s *terminalSession) {
		now := h.now()
		s.Status = "failed"
		s.EndedAt = &now
		s.ErrorCode = code
		s.ErrorMessage = message
	})
	if run, _ := h.findPipelineRunByWorkflow(ctx, session.WorkflowName); run != nil {
		h.appendTerminalRunEvent(ctx, run, "pod_terminal_session_failed", "failed", session.NodeID, session.PodName, session.Command, session.ErrorMessage)
	}
	return session
}

func (h *Handler) now() time.Time {
	if h.terminalNowFunc != nil {
		return h.terminalNowFunc().UTC()
	}
	return time.Now().UTC()
}

func (h *Handler) findPipelineRunByWorkflow(ctx context.Context, workflowName string) (*models.PipelineRun, error) {
	if h.runRepo == nil || strings.TrimSpace(workflowName) == "" {
		return nil, nil
	}
	return h.runRepo.FindByWorkflowName(ctx, workflowName)
}

func (h *Handler) appendTerminalRunEvent(ctx context.Context, run *models.PipelineRun, eventType, status, nodeID, podName, command, message string) {
	if h.runEventRepo == nil || run == nil {
		return
	}
	_ = h.runEventRepo.Append(ctx, &models.PipelineRunEvent{
		RunID:        run.ID,
		WorkflowName: run.WorkflowName,
		EventType:    eventType,
		SubjectType:  "pod",
		SubjectID:    podName,
		Status:       status,
		Message:      message,
		Payload: map[string]interface{}{
			"nodeId":  nodeID,
			"podName": podName,
			"command": command,
		},
		OccurredAt: h.now(),
		ObservedAt: h.now(),
	})
}

func (h *Handler) terminalCapabilityForNode(run *models.PipelineRun, _ *wfv1.Workflow, node wfv1.NodeStatus, podName string) map[string]interface{} {
	policy := terminalPolicyFromRun(run)
	enabled := policy.Enabled && podName != "" && (policy.AllowCompletedPods || isWorkflowNodeRunning(node))
	reason := "Pod 终端未启用，请在执行目标配置中开启"
	if policy.Enabled {
		reason = "Pod 终端可用，当前节点 Pod 正在运行"
	}
	if podName == "" {
		reason = "当前节点未解析到 Pod"
	} else if policy.Enabled && !policy.AllowCompletedPods && !isWorkflowNodeRunning(node) {
		reason = "Pod 已完成，终端仅支持运行中的 Pod"
	}
	return map[string]interface{}{
		"execEnabled":       enabled,
		"logStreamEnabled":  true,
		"metricsEnabled":    false,
		"costEnabled":       false,
		"reason":            reason,
		"allowedCommands":   policy.AllowedCommands,
		"maxSessionSeconds": policy.MaxSessionSeconds,
	}
}

func terminalPolicyFromRun(run *models.PipelineRun) terminalPolicy {
	policy := terminalPolicy{
		Enabled:            false,
		MaxSessionSeconds:  defaultTerminalMaxSessionSeconds,
		IdleTimeoutSeconds: defaultTerminalIdleSeconds,
		AllowedCommands:    []string{"sh", "pwd", "ls", "env"},
		DeniedPatterns:     []string{"kubectl", "docker", "crictl", "rm -rf /", "mkfs"},
	}
	if run == nil {
		return policy
	}
	raw, _ := run.TargetSnapshot["terminal"].(map[string]interface{})
	if raw == nil {
		raw, _ = run.TargetSnapshot["terminalPolicy"].(map[string]interface{})
	}
	if raw == nil {
		return policy
	}
	if enabled, ok := raw["enabled"].(bool); ok {
		policy.Enabled = enabled
	}
	if value := intFromAny(raw["maxSessionSeconds"]); value > 0 {
		policy.MaxSessionSeconds = value
	}
	if value := intFromAny(raw["idleTimeoutSeconds"]); value > 0 {
		policy.IdleTimeoutSeconds = value
	}
	if values := stringSliceFromAny(raw["allowedCommands"]); len(values) > 0 {
		policy.AllowedCommands = values
	}
	if values := stringSliceFromAny(raw["deniedPatterns"]); len(values) > 0 {
		policy.DeniedPatterns = values
	}
	if allow, ok := raw["allowCompletedPods"].(bool); ok {
		policy.AllowCompletedPods = allow
	}
	return policy
}

func validateTerminalCommand(policy terminalPolicy, command string) error {
	normalized := strings.TrimSpace(command)
	if normalized == "" {
		return errors.New("terminal command is required")
	}
	lower := strings.ToLower(normalized)
	for _, pattern := range policy.DeniedPatterns {
		if pattern != "" && strings.Contains(lower, strings.ToLower(pattern)) {
			return fmt.Errorf("terminal command %q is blocked by policy", normalized)
		}
	}
	for _, allowed := range policy.AllowedCommands {
		if normalized == allowed {
			return nil
		}
	}
	return fmt.Errorf("terminal command %q is not allowed for this execution target", normalized)
}

func terminalCommandArgs(command string) []string {
	normalized := strings.TrimSpace(command)
	if normalized == "" {
		return nil
	}
	if strings.ContainsAny(normalized, " \t\n\r") {
		return []string{"sh", "-lc", normalized}
	}
	return []string{normalized}
}

func isWorkflowNodeRunning(node wfv1.NodeStatus) bool {
	return strings.EqualFold(string(node.Phase), "Running")
}

func executionTargetIDFromRun(run *models.PipelineRun) string {
	if run == nil {
		return ""
	}
	return run.ExecutionTargetID
}

func targetString(snapshot map[string]interface{}, key string) string {
	if snapshot == nil {
		return ""
	}
	value, _ := snapshot[key].(string)
	return value
}

func intFromAny(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func stringSliceFromAny(value interface{}) []string {
	raw, ok := value.([]interface{})
	if !ok {
		if typed, ok := value.([]string); ok {
			return typed
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func sanitizeTerminalSession(session *terminalSession) gin.H {
	return gin.H{
		"id":                session.ID,
		"runId":             session.RunID,
		"workflowName":      session.WorkflowName,
		"nodeId":            session.NodeID,
		"podName":           session.PodName,
		"containerName":     session.ContainerName,
		"executionTargetId": session.ExecutionTargetID,
		"cluster":           session.Cluster,
		"namespace":         session.Namespace,
		"command":           session.Command,
		"status":            session.Status,
		"attachUrl":         session.AttachURL,
		"expiresAt":         session.ExpiresAt,
		"createdAt":         session.CreatedAt,
		"attachedAt":        session.AttachedAt,
		"endedAt":           session.EndedAt,
		"errorCode":         session.ErrorCode,
		"errorMessage":      session.ErrorMessage,
	}
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func hijackWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, bool) {
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return nil, nil, false
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket hijack unavailable", http.StatusInternalServerError)
		return nil, nil, false
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, nil, false
	}
	accept := websocketAcceptKey(key)
	_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	_, _ = rw.WriteString("Upgrade: websocket\r\n")
	_, _ = rw.WriteString("Connection: Upgrade\r\n")
	_, _ = rw.WriteString("Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
	_ = rw.Flush()
	return conn, rw, true
}

func websocketAcceptKey(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, key)
	_, _ = io.WriteString(h, "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func writeWebSocketText(rw *bufio.ReadWriter, payload string) {
	data := []byte(payload)
	_ = rw.WriteByte(0x81)
	switch {
	case len(data) < 126:
		_ = rw.WriteByte(byte(len(data)))
	case len(data) <= 65535:
		_ = rw.WriteByte(126)
		_ = rw.WriteByte(byte(len(data) >> 8))
		_ = rw.WriteByte(byte(len(data)))
	default:
		_ = rw.WriteByte(127)
		for i := 7; i >= 0; i-- {
			_ = rw.WriteByte(byte(uint64(len(data)) >> (8 * i)))
		}
	}
	_, _ = rw.Write(data)
}
