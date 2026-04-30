package cdc

import (
	"strconv"
	"strings"
	"time"

	"data-platform/internal/config"
)

func NewKafkaSourceFromConfig(cfg *config.Config, runtimeCfg RuntimeConfig) *KafkaSource {
	brokers := splitCSV(cfg.CDCKafkaBrokers)
	pollTimeoutMs, _ := strconv.Atoi(cfg.CDCKafkaPollTimeoutMs)
	maxBatch, _ := strconv.Atoi(cfg.CDCKafkaMaxBatch)
	topics := append(
		runtimeCfg.TopicsFor(ConsumerKindBronzeEvents),
		runtimeCfg.TopicsFor(ConsumerKindSearchProjection)...,
	)

	poller := NewKafkaGoPoller(
		brokers,
		cfg.CDCKafkaGroupID,
		topics,
		time.Duration(pollTimeoutMs)*time.Millisecond,
		maxBatch,
	)
	return &KafkaSource{Poller: poller}
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
