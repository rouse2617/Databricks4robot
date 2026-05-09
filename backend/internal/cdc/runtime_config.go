package cdc

import "github.com/CyberOrigin2077/cyber-databrew/internal/config"

func BuildRuntimeConfig(cfg *config.Config) RuntimeConfig {
	enabled := cfg.CDCEnabled == "true"
	return RuntimeConfig{
		Enabled:      enabled,
		SourceDriver: cfg.CDCSourceDriver,
		Mappings: []TopicMapping{
			{Table: "asset_events", Topic: cfg.CDCAssetEventsTopic, Consumer: ConsumerKindBronzeEvents},
			{Table: "assets", Topic: cfg.CDCAssetsTopic, Consumer: ConsumerKindSearchProjection},
			{Table: "asset_tags", Topic: cfg.CDCAssetTagsTopic, Consumer: ConsumerKindSearchProjection},
			{Table: "asset_algo_latest", Topic: cfg.CDCAssetAlgoLatestTopic, Consumer: ConsumerKindSearchProjection},
			{Table: "mcap_files", Topic: cfg.CDCMcapFilesTopic, Consumer: ConsumerKindSearchProjection},
			{Table: "actions", Topic: cfg.CDCActionsTopic, Consumer: ConsumerKindSearchProjection},
		},
	}
}
