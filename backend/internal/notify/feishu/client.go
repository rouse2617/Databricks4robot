// Package feishu sends plain-text messages to a Feishu (Lark) custom bot
// incoming webhook. It knows nothing about any particular caller's domain —
// callers build their own message text and own their own webhook URL config.
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultTimeout    = 10 * time.Second
	defaultMaxRetries = 2
)

// Config configures a Client. All fields are optional; zero values fall back
// to sane defaults. WebhookURL empty makes SendText a no-op (feature disabled).
type Config struct {
	WebhookURL string
	Timeout    time.Duration
	MaxRetries int
	HTTPClient *http.Client
}

// Sender is the capability a Client provides. Consumers that want to depend
// on an interface rather than this concrete package are expected to declare
// their own local interface with this same method — *Client satisfies it
// structurally without the consumer importing this package's types.
type Sender interface {
	SendText(ctx context.Context, text string) error
}

// Client sends text messages to a single Feishu custom bot webhook.
type Client struct {
	cfg Config
}

// NewClient builds a Client, applying default Timeout/MaxRetries/HTTPClient
// when left zero.
func NewClient(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}
	return &Client{cfg: cfg}
}

type textPayload struct {
	MsgType string      `json:"msg_type"`
	Content textContent `json:"content"`
}

type textContent struct {
	Text string `json:"text"`
}

// feishuResponse captures Feishu's application-level status, which can
// indicate failure even on HTTP 200 (e.g. StatusCode != 0 for a revoked bot).
type feishuResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// SendText posts text to the configured webhook. A blank WebhookURL is
// treated as "notifications disabled" and returns nil without any network
// call. Transient failures are retried up to MaxRetries times within this
// call; callers do not need their own retry loop.
func (c *Client) SendText(ctx context.Context, text string) error {
	if c == nil || c.cfg.WebhookURL == "" {
		return nil
	}

	body, err := json.Marshal(textPayload{
		MsgType: "text",
		Content: textContent{Text: text},
	})
	if err != nil {
		return fmt.Errorf("feishu: marshal payload: %w", err)
	}

	var lastErr error
	attempts := c.cfg.MaxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
			}
		}
		lastErr = c.sendOnce(ctx, body)
		if lastErr == nil {
			return nil
		}
	}
	return fmt.Errorf("feishu: send failed after %d attempt(s): %w", attempts, lastErr)
}

func (c *Client) sendOnce(ctx context.Context, body []byte) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var parsed feishuResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		// Feishu's webhook response is normally small JSON; a decode failure
		// on an otherwise-2xx response is not worth failing the send over.
		return nil
	}
	if parsed.Code != 0 {
		return errors.New("feishu error: " + parsed.Msg)
	}
	return nil
}
