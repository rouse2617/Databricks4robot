package cdc

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

func TestValidateRuntimeHealth_OK(t *testing.T) {
	cfg := &config.Config{
		CDCSourceDriver: "debezium-kafka",
		CDCKafkaBrokers: "localhost:19092",
		CDCKafkaGroupID: "g1",
	}
	runtimeCfg := RuntimeConfig{
		Enabled: true,
		Mappings: []TopicMapping{
			{Table: "asset_events", Topic: "topic.asset_events", Consumer: ConsumerKindBronzeEvents},
			{Table: "assets", Topic: "topic.assets", Consumer: ConsumerKindSearchProjection},
		},
	}
	if err := ValidateRuntimeHealth(cfg, runtimeCfg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateRuntimeHealth_FailsOnMissingBroker(t *testing.T) {
	cfg := &config.Config{
		CDCSourceDriver: "debezium-kafka",
		CDCKafkaBrokers: "",
		CDCKafkaGroupID: "g1",
	}
	runtimeCfg := RuntimeConfig{
		Enabled: true,
		Mappings: []TopicMapping{
			{Table: "asset_events", Topic: "topic.asset_events", Consumer: ConsumerKindBronzeEvents},
			{Table: "assets", Topic: "topic.assets", Consumer: ConsumerKindSearchProjection},
		},
	}
	if err := ValidateRuntimeHealth(cfg, runtimeCfg); err == nil {
		t.Fatalf("expected validation error")
	}
}
