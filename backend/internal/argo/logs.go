package argo

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type logStreamEntry struct {
	Result struct {
		Content string `json:"content"`
		PodName string `json:"podName"`
	} `json:"result"`
}

func parseLogStream(r io.Reader) (string, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", nil
	}

	if logs, ok, err := parseLogJSONValues(raw); ok || err != nil {
		return logs, err
	}
	return parseLogLines(raw)
}

func parseLogJSONValues(raw []byte) (string, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var builder strings.Builder
	parsed := false
	for {
		var entry logStreamEntry
		err := dec.Decode(&entry)
		if err == io.EOF {
			return builder.String(), parsed, nil
		}
		if err != nil {
			return "", parsed, nil
		}
		parsed = true
		appendLogEntry(&builder, entry)
	}
}

func parseLogLines(raw []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var builder strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry logStreamEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return "", fmt.Errorf("decode log entry: %w", err)
		}
		appendLogEntry(&builder, entry)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func appendLogEntry(builder *strings.Builder, entry logStreamEntry) {
	content := entry.Result.Content
	if content == "" {
		return
	}
	if entry.Result.PodName != "" {
		builder.WriteString(entry.Result.PodName)
		builder.WriteString(" ")
	}
	builder.WriteString(content)
}
