package schedtask

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

// FeishuNotifier posts scheduled-task alerts to a Feishu bot webhook. It is a
// short-term copy of services/grace-sync/main.go:notifyFeishu (decisions.md
// CYB-3744): we deliberately do NOT abstract a shared notifier package until a
// second consumer needs one — otherwise we grow a bespoke framework for one
// caller.
//
// Wire it by env: FEISHU_BOT_WEBHOOK. Empty webhook = notifier no-ops
// (matches grace-sync behaviour, so dev environments without the secret don't
// error out).
type FeishuNotifier struct {
	webhook string
	http    *http.Client
	nowFn   func() time.Time // injectable for tests
}

// NewFeishuNotifierFromEnv reads FEISHU_BOT_WEBHOOK. Returns a nil notifier
// when unset so callers can wire it unconditionally.
func NewFeishuNotifierFromEnv() *FeishuNotifier {
	webhook := os.Getenv("FEISHU_BOT_WEBHOOK")
	return &FeishuNotifier{
		webhook: webhook,
		http:    &http.Client{Timeout: 10 * time.Second},
		nowFn:   time.Now,
	}
}

// NotifyRuleFailure is called when a rule's source fetch or batch creation
// errored. The rule is already marked failed in DB; this is best-effort UX.
func (n *FeishuNotifier) NotifyRuleFailure(ctx context.Context, rule Rule, err error) {
	if n == nil {
		return
	}
	msg := fmt.Sprintf("❌ 定时任务失败\n\n"+
		"📌 规则: %s\n"+
		"🎬 流水线: %s\n"+
		"🎯 资源池: %s\n"+
		"🧭 触发模式: %s\n"+
		"⏰ 时间: %s\n"+
		"⚠️  错误: %s",
		rule.Name, rule.TemplateID, rule.TargetID, rule.TriggerMode,
		n.now().UTC().Format(time.RFC3339), truncateErr(err))
	n.send(ctx, msg)
}

// NotifyRuleStuck fires when a rule hasn't successfully run for > threshold.
// It is a bounded "static rot" alarm, not a per-cycle notification.
func (n *FeishuNotifier) NotifyRuleStuck(ctx context.Context, rule Rule, since time.Duration) {
	if n == nil {
		return
	}
	msg := fmt.Sprintf("🟠 定时任务陈旧\n\n"+
		"📌 规则: %s\n"+
		"⏰ 距上次成功: %s\n"+
		"🎬 流水线: %s\n"+
		"⚠️  说明: 规则未在预期间隔内成功运行,可能同步链路已停",
		rule.Name, since.Round(time.Minute), rule.TemplateID)
	n.send(ctx, msg)
}

// NotifyRuleEmpty is called only for rules that opted into empty notifications
// (trigger_config.notifyOnEmpty). Mirrors grace-sync's "no videos" message.
func (n *FeishuNotifier) NotifyRuleEmpty(ctx context.Context, rule Rule) {
	if n == nil {
		return
	}
	msg := fmt.Sprintf("⚠️ 定时任务执行 — 无数据\n\n"+
		"📌 规则: %s\n"+
		"🎬 流水线: %s\n"+
		"⏰ 时间: %s\n"+
		"📌 状态: 本次拉取到 0 条,未创建批量",
		rule.Name, rule.TemplateID, n.now().UTC().Format(time.RFC3339))
	n.send(ctx, msg)
}

// send posts a text message to the Feishu webhook. Errors are logged and
// swallowed — a notification failing must NEVER block or fail the scheduler.
func (n *FeishuNotifier) send(ctx context.Context, message string) {
	if n.webhook == "" {
		slog.Debug("schedtask.notify: FEISHU_BOT_WEBHOOK unset, skipping", "message", firstLine(message))
		return
	}
	payload := map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": message},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhook, bytes.NewReader(body))
	if err != nil {
		slog.Warn("schedtask.notify: build request failed", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.http.Do(req)
	if err != nil {
		slog.Warn("schedtask.notify: send failed", "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("schedtask.notify: webhook non-200", "status", resp.StatusCode)
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

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
