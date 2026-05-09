package cdc

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

type recordingHandler struct {
	batches [][]ChangeEvent
}

func (r *recordingHandler) HandleBatch(_ context.Context, events []ChangeEvent) error {
	copied := make([]ChangeEvent, len(events))
	copy(copied, events)
	r.batches = append(r.batches, copied)
	return nil
}

type noopSource struct{}

func (noopSource) Run(context.Context, Router) error { return nil }

func TestBuildRuntimeConfig_DefaultMappings(t *testing.T) {
	cfg := &config.Config{
		CDCEnabled:              "true",
		CDCSourceDriver:         "debezium-kafka",
		CDCAssetEventsTopic:     "topic.asset_events",
		CDCAssetsTopic:          "topic.assets",
		CDCAssetTagsTopic:       "topic.asset_tags",
		CDCAssetAlgoLatestTopic: "topic.asset_algo_latest",
		CDCMcapFilesTopic:       "topic.mcap_files",
		CDCActionsTopic:         "topic.actions",
	}

	runtimeCfg := BuildRuntimeConfig(cfg)
	if !runtimeCfg.Enabled {
		t.Fatal("expected cdc runtime to be enabled")
	}
	if runtimeCfg.SourceDriver != "debezium-kafka" {
		t.Fatalf("source driver = %q", runtimeCfg.SourceDriver)
	}

	bronzeTopics := runtimeCfg.TopicsFor(ConsumerKindBronzeEvents)
	if len(bronzeTopics) != 1 || bronzeTopics[0] != "topic.asset_events" {
		t.Fatalf("unexpected bronze topics: %v", bronzeTopics)
	}
	searchTopics := runtimeCfg.TopicsFor(ConsumerKindSearchProjection)
	if len(searchTopics) != 5 {
		t.Fatalf("expected 5 search topics, got %v", searchTopics)
	}
}

func TestRuntime_HandleTopicBatch_RoutesByTopic(t *testing.T) {
	handler := &recordingHandler{}
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "noop"},
		Source: noopSource{},
		Handlers: map[string]BatchHandler{
			"topic.assets": handler,
		},
	}

	err := runtime.HandleTopicBatch(context.Background(), "topic.assets", []ChangeEvent{
		{Table: "assets", Op: OperationUpdate},
	})
	if err != nil {
		t.Fatalf("HandleTopicBatch returned error: %v", err)
	}
	if len(handler.batches) != 1 || len(handler.batches[0]) != 1 {
		t.Fatalf("unexpected handler batches: %+v", handler.batches)
	}
}

func TestRuntime_Run_WithInMemorySource(t *testing.T) {
	handler := &recordingHandler{}
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "in-memory"},
		Source: &InMemorySource{
			Batches: []TopicBatch{
				{
					Topic: "topic.assets",
					Events: []ChangeEvent{
						{Table: "assets", Op: OperationUpdate, After: map[string]any{"asset_id": "a1"}},
					},
				},
			},
		},
		Handlers: map[string]BatchHandler{
			"topic.assets": handler,
		},
	}

	if err := runtime.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(handler.batches) != 1 || len(handler.batches[0]) != 1 {
		t.Fatalf("unexpected handler batches: %+v", handler.batches)
	}
}

func TestRuntime_HandleTopicBatch_UnknownTopicIsDroppedByDefault(t *testing.T) {
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "in-memory"},
		Source: noopSource{},
		Handlers: map[string]BatchHandler{
			"topic.assets": &recordingHandler{},
		},
	}
	if err := runtime.HandleTopicBatch(context.Background(), "topic.unknown", []ChangeEvent{{Table: "assets"}}); err != nil {
		t.Fatalf("expected unknown topic to be dropped, got %v", err)
	}
}

func TestRuntime_HandleTopicBatch_UnknownTopicFailsInStrictMode(t *testing.T) {
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "in-memory", StrictTopicRouting: true},
		Source: noopSource{},
		Handlers: map[string]BatchHandler{
			"topic.assets": &recordingHandler{},
		},
	}
	if err := runtime.HandleTopicBatch(context.Background(), "topic.unknown", []ChangeEvent{{Table: "assets"}}); err == nil {
		t.Fatalf("expected error in strict topic routing mode")
	}
}
