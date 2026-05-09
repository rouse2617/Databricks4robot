package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/CyberOrigin2077/cyber-databrew/internal/cdc"
	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

type loggingHandler struct {
	name string
}

func (h *loggingHandler) HandleBatch(_ context.Context, events []cdc.ChangeEvent) error {
	for _, event := range events {
		slog.Info("cdc-demo event",
			"handler", h.name,
			"table", event.Table,
			"op", event.Op,
			"key", event.Key,
		)
	}
	return nil
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	runtimeCfg := cdc.BuildRuntimeConfig(cfg)
	runtimeCfg.Enabled = true

	var source cdc.Source
	switch cfg.CDCSourceDriver {
	case "", "in-memory":
		source = &cdc.InMemorySource{
			Batches: []cdc.TopicBatch{
				{
					Topic: cfg.CDCAssetEventsTopic,
					Events: []cdc.ChangeEvent{
						{
							Table: "asset_events",
							Op:    cdc.OperationCreate,
							After: map[string]any{
								"event_id":               "demo-1",
								"event_seq":              int64(1),
								"event_type":             "asset_created",
								"aggregate_type":         "asset",
								"payload_schema_version": "v1",
								"event_source":           "backend",
								"publish_state":          "pending",
								"event_payload":          `{"asset_id":"demo-asset"}`,
								"occurred_at":            "2026-04-30T00:00:00Z",
								"created_at":             "2026-04-30T00:00:00Z",
							},
						},
					},
				},
				{
					Topic: cfg.CDCAssetsTopic,
					Events: []cdc.ChangeEvent{
						{
							Table: "assets",
							Op:    cdc.OperationUpdate,
							After: map[string]any{
								"asset_id": "demo-asset",
							},
						},
					},
				},
			},
		}
	case "debezium-kafka":
		source = cdc.NewKafkaSourceFromConfig(cfg, runtimeCfg)
	default:
		slog.Error("unsupported CDC_SOURCE_DRIVER for cdc-demo", "driver", cfg.CDCSourceDriver)
		os.Exit(1)
	}

	bronzeSink := cdc.NewBronzeSink(cdc.BronzeSinkConfig{
		Enabled:    true,
		StagingDir: cfg.CDCBronzeStagingDir,
	})
	bronzeHandler := cdc.BatchHandler(&loggingHandler{name: "bronze"})
	if bronzeSink != nil {
		bronzeHandler = &cdc.BronzeConsumer{
			Sink:                 bronzeSink,
			IncludeSnapshotReads: true,
		}
	}

	handlers := map[string]cdc.BatchHandler{}
	for _, topic := range runtimeCfg.TopicsFor(cdc.ConsumerKindBronzeEvents) {
		handlers[topic] = bronzeHandler
	}
	for _, topic := range runtimeCfg.TopicsFor(cdc.ConsumerKindSearchProjection) {
		handlers[topic] = &loggingHandler{name: "search_projection"}
	}

	runtime := &cdc.Runtime{
		Config:   runtimeCfg,
		Source:   source,
		Handlers: handlers,
	}
	if err := cdc.ValidateRuntimeHealth(cfg, runtimeCfg); err != nil {
		slog.Error("cdc-demo startup validation failed", "err", err)
		os.Exit(1)
	}
	if err := runtime.Validate(); err != nil {
		slog.Error("cdc-demo invalid runtime", "err", err)
		os.Exit(1)
	}

	slog.Info("cdc-demo starting",
		"driver", cfg.CDCSourceDriver,
		"bronze_topic", cfg.CDCAssetEventsTopic,
		"search_topics", append(
			runtimeCfg.TopicsFor(cdc.ConsumerKindSearchProjection),
			runtimeCfg.TopicsFor(cdc.ConsumerKindBronzeEvents)...,
		),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runtime.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("cdc-demo exited with error", "err", err)
		os.Exit(1)
	}
	fmt.Println("cdc-demo stopped")
}
