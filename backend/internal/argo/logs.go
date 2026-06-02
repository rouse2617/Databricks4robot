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

func parseLogStreamBounded(r io.Reader, limitBytes *int64) (string, int, bool, error) {
	var reader io.Reader = r
	var limit int64
	if limitBytes != nil && *limitBytes >= 0 {
		limit = *limitBytes
		reader = io.LimitReader(r, limit+1)
	}

	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", 0, false, err
	}

	truncated := false
	if limitBytes != nil && int64(len(raw)) > limit {
		raw = raw[:limit]
		truncated = true
	}

	logs, lineCount, err := parseLogBytes(raw, truncated)
	return logs, lineCount, truncated, err
}

func parseLogBytes(raw []byte, allowPartial bool) (string, int, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", 0, nil
	}

	if logs, ok, err := parseLogJSONValues(raw, allowPartial); ok || err != nil {
		return logs, countLogLines(logs), err
	}
	logs, err := parseLogLines(raw, allowPartial)
	return logs, countLogLines(logs), err
}

func parseLogJSONValues(raw []byte, allowPartial bool) (string, bool, error) {
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
			if allowPartial && parsed {
				return builder.String(), parsed, nil
			}
			return "", parsed, nil
		}
		parsed = true
		appendLogEntry(&builder, entry)
	}
}

func parseLogLines(raw []byte, allowPartial bool) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var builder strings.Builder
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry logStreamEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			if allowPartial && scanner.Err() == nil && lineNumber > 0 {
				continue
			}
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

func countLogLines(logs string) int {
	logs = strings.TrimSuffix(logs, "\n")
	if strings.TrimSpace(logs) == "" {
		return 0
	}
	return len(strings.Split(logs, "\n"))
}
