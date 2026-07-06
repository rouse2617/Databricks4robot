package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendText_EmptyWebhookURLIsNoOp(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(Config{WebhookURL: ""})
	if err := c.SendText(context.Background(), "hello"); err != nil {
		t.Fatalf("SendText with empty webhook URL should be a no-op, got err: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("expected no HTTP calls, got %d", calls.Load())
	}
}

func TestSendText_PostsExpectedPayload(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(Config{WebhookURL: srv.URL})
	if err := c.SendText(context.Background(), "批量任务完成"); err != nil {
		t.Fatalf("SendText: %v", err)
	}

	if gotBody["msg_type"] != "text" {
		t.Errorf("msg_type = %v, want text", gotBody["msg_type"])
	}
	content, ok := gotBody["content"].(map[string]any)
	if !ok {
		t.Fatalf("content field missing or wrong shape: %v", gotBody["content"])
	}
	if content["text"] != "批量任务完成" {
		t.Errorf("content.text = %v, want 批量任务完成", content["text"])
	}
}

func TestSendText_RetriesOnFailureThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	c := NewClient(Config{WebhookURL: srv.URL, MaxRetries: 2})
	if err := c.SendText(context.Background(), "retry me"); err != nil {
		t.Fatalf("SendText should succeed after retry, got err: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected exactly 2 attempts, got %d", calls.Load())
	}
}

func TestSendText_GivesUpAfterMaxRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(Config{WebhookURL: srv.URL, MaxRetries: 2})
	if err := c.SendText(context.Background(), "always fails"); err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
	if calls.Load() != 3 { // 1 initial attempt + 2 retries
		t.Fatalf("expected 3 attempts (1 + MaxRetries), got %d", calls.Load())
	}
}

func TestSendText_FeishuApplicationErrorIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":19021,"msg":"webhook revoked"}`))
	}))
	defer srv.Close()

	c := NewClient(Config{WebhookURL: srv.URL, MaxRetries: 0})
	if err := c.SendText(context.Background(), "hi"); err == nil {
		t.Fatal("expected an error for non-zero Feishu response code, got nil")
	}
}

func TestSendText_RespectsContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewClient(Config{WebhookURL: srv.URL, MaxRetries: 3})
	start := time.Now()
	err := c.SendText(ctx, "cancelled")
	if err == nil {
		t.Fatal("expected an error for cancelled context")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("SendText took %v after cancellation, expected it to return quickly", elapsed)
	}
}
