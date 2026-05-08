//go:build integration

package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	espkg "data-platform/internal/elasticsearch"

	"github.com/testcontainers/testcontainers-go"
	tcElasticsearch "github.com/testcontainers/testcontainers-go/modules/elasticsearch"
)

func TestRuntime_Run_WithElasticsearchContainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	esContainer, err := tcElasticsearch.Run(
		ctx,
		"docker.elastic.co/elasticsearch/elasticsearch:8.11.1",
		testcontainers.WithEnv(map[string]string{
			"xpack.security.enabled": "false",
			"discovery.type":         "single-node",
			"ES_JAVA_OPTS":           "-Xms256m -Xmx256m",
		}),
	)
	if err != nil {
		t.Fatalf("start elasticsearch container: %v", err)
	}
	defer func() {
		_ = esContainer.Terminate(context.Background())
	}()

	host, err := esContainer.Host(ctx)
	if err != nil {
		t.Fatalf("resolve elasticsearch host: %v", err)
	}
	port, err := esContainer.MappedPort(ctx, "9200/tcp")
	if err != nil {
		t.Fatalf("resolve elasticsearch port: %v", err)
	}
	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	es := espkg.New(baseURL, "assets")
	if err := waitForESPing(ctx, es); err != nil {
		t.Fatalf("wait for elasticsearch readiness: %v", err)
	}

	consumer := &ESConsumer{
		Assets: &fakeAssetRepo{},
		Builder: &fakeBuilder{
			docs: map[string]map[string]any{
				"a-integration": {"asset_id": "a-integration", "owner": "integration"},
			},
			ok: map[string]bool{
				"a-integration": true,
			},
			errs: map[string]error{},
		},
		ES: es,
	}

	topic := "dbserver1.public.assets"
	runtime := &Runtime{
		Config: RuntimeConfig{
			Enabled: true,
			Mappings: []TopicMapping{
				{Table: "assets", Topic: topic, Consumer: ConsumerKindSearchProjection},
			},
		},
		Source: &InMemorySource{
			Batches: []TopicBatch{
				{
					Topic: topic,
					Events: []ChangeEvent{
						{
							Table: "assets",
							Op:    OperationCreate,
							After: map[string]any{"asset_id": "a-integration"},
						},
					},
				},
			},
		},
		Handlers: map[string]BatchHandler{
			topic: consumer,
		},
	}

	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("run cdc runtime: %v", err)
	}
	if err := refreshESIndex(ctx, baseURL, "assets"); err != nil {
		t.Fatalf("refresh elasticsearch index: %v", err)
	}

	count, err := es.Count(ctx)
	if err != nil {
		t.Fatalf("count indexed docs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 indexed document, got %d", count)
	}

	doc, err := fetchESDocument(ctx, baseURL, "assets", "a-integration")
	if err != nil {
		t.Fatalf("fetch indexed document: %v", err)
	}
	if got := fmt.Sprint(doc["asset_id"]); got != "a-integration" {
		t.Fatalf("expected indexed asset_id a-integration, got %q", got)
	}
}

func waitForESPing(ctx context.Context, es *espkg.Client) error {
	deadline := time.Now().Add(90 * time.Second)
	for {
		if err := es.Ping(ctx); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for elasticsearch ping")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func fetchESDocument(ctx context.Context, baseURL, index, id string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s/_doc/%s", baseURL, index, id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Found  bool           `json:"found"`
		Source map[string]any `json:"_source"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if !parsed.Found {
		return nil, fmt.Errorf("document %s not found", id)
	}
	return parsed.Source, nil
}

func refreshESIndex(ctx context.Context, baseURL, index string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/%s/_refresh", baseURL, index), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
