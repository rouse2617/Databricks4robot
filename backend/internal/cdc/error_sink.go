package cdc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FailureStage string

const (
	FailureStageDecode FailureStage = "decode"
	FailureStageRoute  FailureStage = "route"
	FailureStageCommit FailureStage = "commit"
)

// FailedRecord stores enough data to inspect and replay poison messages.
type FailedRecord struct {
	Time         time.Time      `json:"time"`
	Stage        FailureStage   `json:"stage"`
	Topic        string         `json:"topic"`
	Key          map[string]any `json:"key,omitempty"`
	ValueBase64  string         `json:"value_base64,omitempty"`
	ErrorMessage string         `json:"error"`
}

func NewFailedRecord(stage FailureStage, record KafkaRecord, err error) FailedRecord {
	return FailedRecord{
		Time:         time.Now().UTC(),
		Stage:        stage,
		Topic:        record.Topic,
		Key:          record.Key,
		ValueBase64:  base64.StdEncoding.EncodeToString(record.Value),
		ErrorMessage: err.Error(),
	}
}

func (r FailedRecord) ValueBytes() ([]byte, error) {
	if r.ValueBase64 == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(r.ValueBase64)
	if err != nil {
		return nil, fmt.Errorf("decode value_base64: %w", err)
	}
	return b, nil
}

type ErrorSink interface {
	Write(ctx context.Context, record FailedRecord) error
}

type FileErrorSinkConfig struct {
	Enabled bool
	Dir     string
}

// FileErrorSink appends failed records as JSONL lines for operator replay.
type FileErrorSink struct {
	dir string
	mu  sync.Mutex
}

func NewFileErrorSink(cfg FileErrorSinkConfig) *FileErrorSink {
	if !cfg.Enabled {
		return nil
	}
	dir := cfg.Dir
	if dir == "" {
		dir = "/tmp/cdc-dlq"
	}
	return &FileErrorSink{dir: dir}
}

func (s *FileErrorSink) Write(_ context.Context, record FailedRecord) error {
	if s == nil {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("dlq mkdir: %w", err)
	}

	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("dlq marshal: %w", err)
	}

	path := filepath.Join(s.dir, "failed_records_"+time.Now().UTC().Format("20060102")+".jsonl")
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("dlq open: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("dlq append: %w", err)
	}
	return nil
}
