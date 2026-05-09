package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/CyberOrigin2077/cyber-databrew/internal/cdc"
)

func main() {
	file := flag.String("file", "", "Path to DLQ JSONL file")
	brokersRaw := flag.String("brokers", "localhost:19092", "Comma-separated Kafka brokers")
	overrideTopic := flag.String("topic", "", "Optional topic override for replay")
	dryRun := flag.Bool("dry-run", false, "Only parse and print summary; do not publish")
	flag.Parse()

	if strings.TrimSpace(*file) == "" {
		slog.Error("missing required flag: --file")
		os.Exit(1)
	}

	records, err := readDLQFile(*file)
	if err != nil {
		slog.Error("failed to read dlq file", "err", err)
		os.Exit(1)
	}
	if len(records) == 0 {
		slog.Info("no records to replay")
		return
	}

	if *dryRun {
		slog.Info("dlq replay dry-run summary", "file", *file, "records", len(records))
		return
	}

	brokers := splitCSV(*brokersRaw)
	if len(brokers) == 0 {
		slog.Error("no kafka brokers provided")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	published := 0
	for _, rec := range records {
		topic := rec.Topic
		if *overrideTopic != "" {
			topic = *overrideTopic
		}
		if strings.TrimSpace(topic) == "" {
			slog.Warn("skip record with empty topic")
			continue
		}
		payload, err := rec.ValueBytes()
		if err != nil {
			slog.Error("decode dlq payload failed", "topic", rec.Topic, "err", err)
			continue
		}
		keyBytes, err := json.Marshal(rec.Key)
		if err != nil {
			slog.Error("marshal key failed", "topic", rec.Topic, "err", err)
			continue
		}

		writer := &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
		if err := writer.WriteMessages(ctx, kafka.Message{
			Key:   keyBytes,
			Value: payload,
			Time:  time.Now(),
		}); err != nil {
			_ = writer.Close()
			slog.Error("replay publish failed", "topic", topic, "err", err)
			continue
		}
		_ = writer.Close()
		published++
	}

	slog.Info("dlq replay finished", "file", *file, "published", published, "total", len(records))
}

func readDLQFile(path string) ([]cdc.FailedRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dlq file: %w", err)
	}
	defer f.Close()

	var out []cdc.FailedRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec cdc.FailedRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("decode dlq line: %w", err)
		}
		out = append(out, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan dlq file: %w", err)
	}
	return out, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
