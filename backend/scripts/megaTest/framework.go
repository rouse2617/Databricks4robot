package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type TestResult struct {
	Category string
	Name     string
	Pass     bool
	Code     int
	Expected int
	Latency  time.Duration
	Detail   string
}

type TestSuite struct {
	mu      sync.Mutex
	results []TestResult
	pass    int64
	fail    int64
}

func (s *TestSuite) record(r TestResult) {
	s.mu.Lock()
	s.results = append(s.results, r)
	s.mu.Unlock()
	if r.Pass {
		atomic.AddInt64(&s.pass, 1)
	} else {
		atomic.AddInt64(&s.fail, 1)
	}
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func doReq(method, path string, body interface{}, headers map[string]string) (int, []byte, time.Duration) {
	var reader io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			reader = strings.NewReader(v)
		case []byte:
			reader = bytes.NewReader(v)
		default:
			b, _ := json.Marshal(body)
			reader = bytes.NewReader(b)
		}
	}
	req, err := http.NewRequest(method, *baseURL+path, reader)
	if err != nil {
		return 0, nil, 0
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Grace-Token", *token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	start := time.Now()
	resp, err := httpClient.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return 0, nil, elapsed
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, elapsed
}

func jsonGet(data []byte, key string) string {
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func jsonGetNested(data []byte, keys ...string) string {
	var current interface{}
	json.Unmarshal(data, &current)
	for _, k := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = m[k]
	}
	if current == nil {
		return ""
	}
	return fmt.Sprintf("%v", current)
}

func jsonMapLen(data []byte, key string) int {
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]interface{}); ok {
			return len(mm)
		}
	}
	return 0
}

func jsonArrayLen(data []byte, key string) int {
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			return len(arr)
		}
	}
	return 0
}

var assetCounter int64

func createAsset(tags map[string]string) (string, int) {
	n := atomic.AddInt64(&assetCounter, 1)
	ts := time.Now().UnixNano()
	body := map[string]interface{}{
		"mcap_file_id":       fmt.Sprintf("mega-mcap-%d-%d", ts, n),
		"start_timestamp_ns": ts,
		"end_timestamp_ns":   ts + 60_000_000_000,
		"reviewer":           "mega-tester",
		"owner":              "mega-team",
	}
	if tags != nil {
		body["tags"] = tags
	}
	code, resp, _ := doReq("POST", "/api/v1/assets", body, nil)
	if code == 201 {
		return jsonGet(resp, "asset_id"), code
	}
	return "", code
}

func deleteAsset(id string) {
	doReq("DELETE", "/api/v1/assets/"+id, nil, nil)
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func buildFinishBody(algoKey, status string) map[string]interface{} {
	body := map[string]interface{}{"status": status}
	if status == "failed" {
		body["reason"] = "test failure"
		return body
	}
	switch {
	case strings.HasPrefix(algoKey, "hand_tracking"), strings.HasPrefix(algoKey, "head_tracking"), strings.HasPrefix(algoKey, "body_tracking"):
		body["output_uri"] = "gs://test/output.mcap"
		body["result_size_bytes"] = 100
		body["extra_fields"] = map[string]interface{}{"type": "tracking_v1"}
	case strings.HasPrefix(algoKey, "deface"):
		body["output_uri"] = "gs://test/deface.mp4"
		body["result_size_bytes"] = 200
		body["extra_fields"] = map[string]interface{}{"width": 1920, "height": 1080, "fps": 30, "source_stream": "left", "eye": "left"}
	case strings.HasPrefix(algoKey, "action_annotation"):
		body["output_uri"] = "gs://test/aa.json"
		body["result_size_bytes"] = 50
	}
	return body
}
