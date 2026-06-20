package argo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
