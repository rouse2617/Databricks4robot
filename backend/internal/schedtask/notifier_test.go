package schedtask

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFeishuNotifier_NoWebhookIsNoOp(t *testing.T) {
	n := &FeishuNotifier{http: &http.Client{}} // no webhook
	// If send panics or blocks, this test fails; correctness = "does nothing".
	n.NotifyRuleFailure(context.Background(), Rule{ID: "r"}, errors.New("boom"))
	n.NotifyRuleStuck(context.Background(), Rule{ID: "r"}, time.Hour)
	n.NotifyRuleEmpty(context.Background(), Rule{ID: "r"})
}

func TestFeishuNotifier_PostsMessageAndSurvivesNon200(t *testing.T) {
	var mu sync.Mutex
	var payloads []string
	// First call: succeed. Second: 500.
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		payloads = append(payloads, string(buf))
		call++
		if call >= 2 {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	n := &FeishuNotifier{webhook: srv.URL, http: &http.Client{Timeout: 2 * time.Second}, nowFn: func() time.Time { return time.Date(2026, 7, 21, 6, 0, 0, 0, time.UTC) }}
	n.NotifyRuleFailure(context.Background(), Rule{Name: "grace-sync-rule", TemplateID: "tpl", TargetID: "target", TriggerMode: TriggerIncremental}, errors.New("source fetch: 500"))
	n.NotifyRuleEmpty(context.Background(), Rule{Name: "grace-sync-rule", TemplateID: "tpl"})

	mu.Lock()
	defer mu.Unlock()
	if len(payloads) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(payloads))
	}
	if !strings.Contains(payloads[0], "定时任务失败") || !strings.Contains(payloads[0], "grace-sync-rule") {
		t.Fatalf("failure payload wrong: %s", payloads[0])
	}
	if !strings.Contains(payloads[1], "无数据") {
		t.Fatalf("empty payload wrong: %s", payloads[1])
	}
	// Second call was 500 — must not panic or affect a later call.
	n.NotifyRuleStuck(context.Background(), Rule{Name: "stuck"}, 3*time.Hour)
}
