package cdc

import (
	"strconv"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
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
	maxFailures, err := strconv.Atoi(cfg.CDCMaxConsecutiveFailures)
	if err != nil || maxFailures <= 0 {
		maxFailures = 20
	}
	continueOnCommitFailure := strings.EqualFold(strings.TrimSpace(cfg.CDCContinueOnCommitFailure), "true")
	dlqEnabled := strings.EqualFold(strings.TrimSpace(cfg.CDCDLQEnabled), "true")
	errorSink := NewFileErrorSink(FileErrorSinkConfig{
		Enabled: dlqEnabled,
		Dir:     cfg.CDCDLQDir,
	})

	return &KafkaSource{
		Poller:                  poller,
		ErrorSink:               errorSink,
		MaxConsecutiveFailures:  maxFailures,
		ContinueOnCommitFailure: continueOnCommitFailure,
	}
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
