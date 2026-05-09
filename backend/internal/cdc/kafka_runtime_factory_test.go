package cdc

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

func TestSplitCSV(t *testing.T) {
	got := splitCSV("a:1, b:2 ,,c:3")
	if len(got) != 3 || got[0] != "a:1" || got[1] != "b:2" || got[2] != "c:3" {
		t.Fatalf("unexpected split result: %v", got)
	}
}

func TestNewKafkaSourceFromConfig(t *testing.T) {
	cfg := &config.Config{
		CDCKafkaBrokers:       "localhost:19092,localhost:29092",
		CDCKafkaGroupID:       "group-1",
		CDCKafkaPollTimeoutMs: "500",
		CDCKafkaMaxBatch:      "10",
	}
	runtimeCfg := RuntimeConfig{
		Mappings: []TopicMapping{
			{Topic: "asset_events", Consumer: ConsumerKindBronzeEvents},
			{Topic: "assets", Consumer: ConsumerKindSearchProjection},
		},
	}

	source := NewKafkaSourceFromConfig(cfg, runtimeCfg)
	if source == nil || source.Poller == nil {
		t.Fatal("expected kafka source and poller")
	}
}
