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
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"podName":"wf-1-step-123","content":"hello\n"}}`))
	}))
	defer server.Close()

	client := NewClientFromConfig(&Config{ServerURL: server.URL})
	logs, err := client.GetWorkflowLogs(context.Background(), "wf-1", "wf-1-step-123", "cyber-databrew-dev")
	if err != nil {
		t.Fatalf("GetWorkflowLogs returned error: %v", err)
	}
	if logs != "wf-1-step-123 hello\n" {
		t.Fatalf("logs = %q, want pod-prefixed log content", logs)
	}
	if gotQuery == "" {
		t.Fatal("server did not receive query string")
	}
}
