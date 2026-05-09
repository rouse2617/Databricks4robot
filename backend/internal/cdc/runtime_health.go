package cdc

import (
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

// ValidateRuntimeHealth validates deploy-time wiring for CDC to fail fast with
// actionable diagnostics before starting the long-running source loop.
func ValidateRuntimeHealth(cfg *config.Config, runtimeCfg RuntimeConfig) error {
	if !runtimeCfg.Enabled {
		return nil
	}

	var issues []string
	if len(runtimeCfg.Mappings) == 0 {
		issues = append(issues, "runtime has no topic mappings")
	}
	for _, mapping := range runtimeCfg.Mappings {
		if strings.TrimSpace(mapping.Table) == "" {
			issues = append(issues, "found mapping with empty table")
		}
		if strings.TrimSpace(mapping.Topic) == "" {
			issues = append(issues, fmt.Sprintf("table %q has empty topic", mapping.Table))
		}
	}

	switch cfg.CDCSourceDriver {
	case "debezium-kafka":
		if len(splitCSV(cfg.CDCKafkaBrokers)) == 0 {
			issues = append(issues, "CDC_KAFKA_BROKERS is empty")
		}
		if strings.TrimSpace(cfg.CDCKafkaGroupID) == "" {
			issues = append(issues, "CDC_KAFKA_GROUP_ID is empty")
		}
	case "in-memory":
		// no-op
	default:
		issues = append(issues, fmt.Sprintf("unsupported CDC_SOURCE_DRIVER=%q", cfg.CDCSourceDriver))
	}

	if len(runtimeCfg.TopicsFor(ConsumerKindBronzeEvents)) == 0 {
		issues = append(issues, "no bronze_events topics configured")
	}
	if len(runtimeCfg.TopicsFor(ConsumerKindSearchProjection)) == 0 {
		issues = append(issues, "no search_projection topics configured")
	}

	if len(issues) > 0 {
		return fmt.Errorf("cdc startup validation failed: %s", strings.Join(issues, "; "))
	}
	return nil
}
