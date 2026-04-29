package main

import (
	"fmt"
	"strings"
	"time"
)

func genCreateValidation(suite *TestSuite) {
	cat := "1.输入校验-Create"
	ts := time.Now().UnixNano()
	end := ts + 60_000_000_000

	// 1.1 必填字段缺失组合 (2^4 = 16)
	for mask := 0; mask < 16; mask++ {
		body := map[string]interface{}{}
		missing := []string{}
		if mask&1 != 0 {
			body["mcap_file_id"] = "test"
		} else {
			missing = append(missing, "mcap_file_id")
		}
		if mask&2 != 0 {
			body["start_timestamp_ns"] = ts
		} else {
			missing = append(missing, "start_timestamp_ns")
		}
		if mask&4 != 0 {
			body["end_timestamp_ns"] = end
		} else {
			missing = append(missing, "end_timestamp_ns")
		}
		if mask&8 != 0 {
			body["reviewer"] = "tester"
		} else {
			missing = append(missing, "reviewer")
		}
		expected := 400
		if mask == 15 {
			expected = 201
		}
		code, resp, lat := doReq("POST", "/api/v1/assets", body, nil)
		suite.record(TestResult{cat, fmt.Sprintf("必填组合 mask=%d miss=[%s]", mask, strings.Join(missing, ",")), code == expected, code, expected, lat, ""})
		if code == 201 {
			deleteAsset(jsonGet(resp, "asset_id"))
		}
	}

	// 1.2 timestamp 边界 (10)
	tsCases := []struct {
		n    string
		s, e interface{}
		exp  int
	}{
		{"start==end", ts, ts, 422}, {"start>end", end, ts, 422},
		{"start=1,end=2", 1, 2, 201}, {"负数start", -1000, 1000, 201},
		{"极大值", int64(9223372036854775000), int64(9223372036854775807), 201},
		{"差值=1ns", ts, ts + 1, 201}, {"差值=1s", ts, ts + 1e9, 201},
		{"差值=1h", ts, ts + 3600e9, 201}, {"差值=24h", ts, ts + 86400e9, 201},
		{"负极大", int64(-9223372036854775807), int64(-9223372036854775000), 201},
	}
	for _, tc := range tsCases {
		body := map[string]interface{}{"mcap_file_id": fmt.Sprintf("ts-%s", tc.n), "start_timestamp_ns": tc.s, "end_timestamp_ns": tc.e, "reviewer": "t"}
		code, resp, lat := doReq("POST", "/api/v1/assets", body, nil)
		suite.record(TestResult{cat, fmt.Sprintf("ts: %s", tc.n), code == tc.exp, code, tc.exp, lat, ""})
		if code == 201 {
			deleteAsset(jsonGet(resp, "asset_id"))
		}
	}

	// 1.3 字段长度 (5 fields × 6 lengths = 30)
	for _, field := range []string{"reviewer", "owner", "type", "env", "task"} {
		for _, l := range []int{0, 1, 10, 100, 500, 5000} {
			body := map[string]interface{}{"mcap_file_id": fmt.Sprintf("len-%s-%d", field, l), "start_timestamp_ns": ts, "end_timestamp_ns": end, "reviewer": "t"}
			body[field] = strings.Repeat("A", l)
			code, resp, lat := doReq("POST", "/api/v1/assets", body, nil)
			suite.record(TestResult{cat, fmt.Sprintf("长度 %s=%d", field, l), code == 201 || code == 400, code, 0, lat, ""})
			if code == 201 {
				deleteAsset(jsonGet(resp, "asset_id"))
			}
		}
	}

	// 1.4 畸形 body (10)
	for _, bb := range []struct{ n, b string }{
		{"空", ""}, {"文本", "hello"}, {"XML", "<x/>"}, {"数组", "[1]"}, {"null", "null"},
		{"数字", "42"}, {"布尔", "true"}, {"深嵌套", `{"a":{"b":{"c":"d"}}}`},
		{"大JSON", "{" + strings.Repeat(`"k":"v",`, 5000) + `"z":"z"}`},
		{"控制字符", "{\"\x00\":1}"},
	} {
		code, _, lat := doReq("POST", "/api/v1/assets", bb.b, nil)
		suite.record(TestResult{cat, fmt.Sprintf("畸形: %s", bb.n), code == 400 || code == 201, code, 400, lat, ""})
	}
}
