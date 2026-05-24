package openlineage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Emitter struct {
	Endpoint   string
	HTTPClient *http.Client
}

func NewEmitter(endpoint string, timeout time.Duration) (*Emitter, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("openlineage emitter: endpoint is required")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Emitter{
		Endpoint:   endpoint,
		HTTPClient: &http.Client{Timeout: timeout},
	}, nil
}

func (e *Emitter) Emit(ctx context.Context, event *Event) error {
	if e == nil || strings.TrimSpace(e.Endpoint) == "" {
		return fmt.Errorf("openlineage emitter: incomplete wiring")
	}
	if event == nil {
		return nil
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("openlineage emitter: marshal event: %w", err)
	}
	client := e.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("openlineage emitter: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("openlineage emitter: post event: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("openlineage emitter: endpoint returned status %d", resp.StatusCode)
	}
	return nil
}
