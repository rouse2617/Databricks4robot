package subtask

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type FeishuNotifier struct {
	webhook string
	http    *http.Client
	nowFn   func() time.Time
}

func NewFeishuNotifierFromEnv() *FeishuNotifier {
	webhook := os.Getenv("FEISHU_BOT_WEBHOOK")
	return &FeishuNotifier{
		webhook: webhook,
		http:    &http.Client{Timeout: 10 * time.Second},
		nowFn:   time.Now,
	}
}

func (n *FeishuNotifier) NotifyTaskFailure(ctx context.Context, task Task, err error) {
	if n == nil {
		return
	}
	msg := fmt.Sprintf("❌ 订阅任务失败\n\n"+
		"📌 任务: %s\n"+
		"🎬 流水线: %s\n"+
		"🎯 资源池: %s\n"+
		"📡 订阅: %s/%s\n"+
		"⏰ 时间: %s\n"+
		"⚠️  错误: %s",
		task.Name, task.TemplateID, task.TargetID,
		task.ProjectID, task.SubscriptionID,
		n.now().UTC().Format(time.RFC3339), truncateErr(err))
	n.send(ctx, msg)
}

func (n *FeishuNotifier) NotifyTaskEmpty(ctx context.Context, task Task) {
	if n == nil {
		return
	}
	msg := fmt.Sprintf("⚠️ 订阅任务 — 无消息\n\n"+
		"📌 任务: %s\n"+
		"📡 订阅: %s/%s\n"+
		"⏰ 时间: %s",
		task.Name, task.ProjectID, task.SubscriptionID,
		n.now().UTC().Format(time.RFC3339))
	n.send(ctx, msg)
}

func (n *FeishuNotifier) send(ctx context.Context, message string) {
	if n.webhook == "" {
		slog.Debug("subtask.notify: FEISHU_BOT_WEBHOOK unset, skipping")
		return
	}
	payload := map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": message},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhook, bytes.NewReader(body))
	if err != nil {
		slog.Warn("subtask.notify: build request failed", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.http.Do(req)
	if err != nil {
		slog.Warn("subtask.notify: send failed", "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("subtask.notify: webhook non-200", "status", resp.StatusCode)
	}
}

func (n *FeishuNotifier) now() time.Time {
	if n.nowFn != nil {
		return n.nowFn()
	}
	return time.Now()
}

func truncateErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	const max = 400
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
