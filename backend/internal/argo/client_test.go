package argo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestClientGetWorkflowLogsUsesPodNameWithoutGrep(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		if r.URL.Path != "/api/v1/workflows/cyber-databrew-dev/wf-1/log" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("podName"); got != "wf-1-step-123" {
			t.Fatalf("podName query = %q, want wf-1-step-123", got)
		}
		if got := r.URL.Query().Get("grep"); got != "" {
			t.Fatalf("grep query = %q, want empty", got)
		}
		if got := r.URL.Query().Get("logOptions.tailLines"); got != "200" {
			t.Fatalf("tailLines query = %q, want 200", got)
		}
		if got := r.URL.Query().Get("logOptions.limitBytes"); got != "1024" {
			t.Fatalf("limitBytes query = %q, want 1024", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"podName":"wf-1-step-123","content":"hello\n"}}`))
	}))
	defer server.Close()

	client := NewClientFromConfig(&Config{ServerURL: server.URL})
	tailLines := int64(200)
	limitBytes := int64(1024)
	result, err := client.GetWorkflowLogs(context.Background(), "wf-1", "wf-1-step-123", "cyber-databrew-dev", WorkflowLogOptions{
		Container:  "main",
		TailLines:  &tailLines,
		LimitBytes: &limitBytes,
	})
	if err != nil {
		t.Fatalf("GetWorkflowLogs returned error: %v", err)
	}
	if result.Logs != "wf-1-step-123 hello\n" {
		t.Fatalf("logs = %q, want pod-prefixed log content", result.Logs)
	}
	if result.LineCount != 1 || result.Truncated {
		t.Fatalf("unexpected result metadata: %#v", result)
	}
	if gotQuery == "" {
		t.Fatal("server did not receive query string")
	}
}

func TestClientGetWorkflowLogsDetectsLimitBytesTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"result":{"podName":"pod","content":"first\n"}}` + "\n" +
				`{"result":{"podName":"pod","content":"second\n"}}` + "\n",
		))
	}))
	defer server.Close()

	client := NewClientFromConfig(&Config{ServerURL: server.URL})
	limitBytes := int64(55)
	result, err := client.GetWorkflowLogs(context.Background(), "wf-1", "pod", "cyber-databrew-dev", WorkflowLogOptions{
		LimitBytes: &limitBytes,
	})
	if err != nil {
		t.Fatalf("GetWorkflowLogs returned error: %v", err)
	}
	if !result.Truncated {
		t.Fatalf("expected truncated result, got %#v", result)
	}
}

func TestClientGetWorkflowLogsPreservesEntryBoundaries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"result":{"podName":"pod","content":"first"}}` +
				`{"result":{"podName":"pod","content":"second"}}`,
		))
	}))
	defer server.Close()

	client := NewClientFromConfig(&Config{ServerURL: server.URL})
	result, err := client.GetWorkflowLogs(context.Background(), "wf-1", "pod", "cyber-databrew-dev", WorkflowLogOptions{})
	if err != nil {
		t.Fatalf("GetWorkflowLogs returned error: %v", err)
	}
	if result.Logs != "pod first\npod second\n" {
		t.Fatalf("logs = %q, want separate pod-prefixed log lines", result.Logs)
	}
	if result.LineCount != 2 {
		t.Fatalf("lineCount = %d, want 2", result.LineCount)
	}
}

// TestCreateWorkflowReturnsErrAlreadyExists covers Phase 0 / Phase 1's
// idempotency hinge: when the Argo API rejects a duplicate create with
// HTTP 409 + "already exists" body, the client must surface a typed
// ErrAlreadyExists that callers can use to drive an adoption retry.
// Witnessed body shape against dev video-proc-dev argo-workflows-server:2746
// on 2026-07-05: {"code":6,"message":"... already exists"}.
func TestCreateWorkflowReturnsErrAlreadyExists(t *testing.T) {
	const argoBody = `{"code":6,"message":"workflows.argoproj.io \"idem-test\" already exists"}`

	var pathSeen string
	var methodSeen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSeen = r.URL.Path
		methodSeen = r.Method
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(argoBody))
	}))
	defer srv.Close()

	c := NewClientFromConfig(&Config{ServerURL: srv.URL})
	wf := &wfv1.Workflow{}
	err := c.CreateWorkflow(context.Background(), wf, "video-proc-dev")
	if err == nil {
		t.Fatalf("CreateWorkflow returned nil error, want ErrAlreadyExists")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("CreateWorkflow err = %v, want errors.Is(_, ErrAlreadyExists)=true", err)
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error message lost source signal: %q (must include \"already exists\")", err.Error())
	}
	if methodSeen != http.MethodPost || pathSeen != "/api/v1/workflows/video-proc-dev" {
		t.Fatalf("unexpected request: %s %s", methodSeen, pathSeen)
	}
	if !IsArgo409Response(argoBody) {
		t.Fatalf("IsArgo409Response should accept the real Argo body")
	}
	if IsTerminatingAlreadyExists(argoBody) {
		t.Fatalf("plain already-exists body must NOT be classified as terminating")
	}
}

// TestIsTerminatingAlreadyExists catches the deletion-in-flight variant
// a Finalizer-cleanup or slow-deletion case will emit. Phase 1 callers
// must use this signal to reset state rather than adopt a dying object.
func TestIsTerminatingAlreadyExists(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{`{"code":6,"message":"... already exists"}`, false},
		{`{"code":6,"message":"... already exists: object is being deleted"}`, true},
		{`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"workflows.argoproj.io \"x\" already exists","reason":"AlreadyExists","details":{"causes":[{"reason":"Object is being deleted","message":"..."}]}}`, true},
		{``, false},
		{`<html>nope</html>`, false},
	}
	for i, c := range cases {
		got := IsTerminatingAlreadyExists(c.body)
		if got != c.want {
			t.Errorf("case %d: IsTerminatingAlreadyExists(%q)=%v, want %v", i, c.body, got, c.want)
		}
	}
}
