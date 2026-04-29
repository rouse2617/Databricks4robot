package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func genTagValidation(suite *TestSuite) {
	cat := "2.Tag校验"
	ts := time.Now().UnixNano()
	end := ts + 60_000_000_000
	mk := func(tags map[string]string) map[string]interface{} {
		return map[string]interface{}{"mcap_file_id": fmt.Sprintf("tag-%d-%d", ts, rand.Intn(1e6)), "start_timestamp_ns": ts, "end_timestamp_ns": end, "reviewer": "t", "tags": tags}
	}

	// 合法 enum 值 (14)
	for tag, vals := range map[string][]string{"priority": {"critical", "high", "medium", "low"}, "quality": {"excellent", "good", "acceptable", "poor", "unusable"}, "scene": {"indoor", "outdoor", "warehouse", "office", "factory"}} {
		for _, v := range vals {
			code, resp, lat := doReq("POST", "/api/v1/assets", mk(map[string]string{tag: v}), nil)
			suite.record(TestResult{cat, fmt.Sprintf("enum %s=%s", tag, v), code == 201, code, 201, lat, ""})
			if code == 201 {
				deleteAsset(jsonGet(resp, "asset_id"))
			}
		}
	}

	// 非法 enum 值 (3×5=15)
	for _, tag := range []string{"priority", "quality", "scene"} {
		for _, v := range []string{"", "INVALID", "123", "true", "null"} {
			code, _, lat := doReq("POST", "/api/v1/assets", mk(map[string]string{tag: v}), nil)
			suite.record(TestResult{cat, fmt.Sprintf("非法 %s=%q", tag, v), code == 422, code, 422, lat, ""})
		}
	}

	// string tags 各种值 (3×8=24)
	for _, tag := range []string{"task", "batch", "notes"} {
		for _, v := range []string{"normal", "", "中文", "emoji🤖", strings.Repeat("x", 100), strings.Repeat("长", 200), "a b c", "k=v&f=b"} {
			code, resp, lat := doReq("POST", "/api/v1/assets", mk(map[string]string{tag: v}), nil)
			suite.record(TestResult{cat, fmt.Sprintf("str %s=%q", tag, truncStr(v, 15)), code == 201 || code == 422, code, 0, lat, ""})
			if code == 201 {
				deleteAsset(jsonGet(resp, "asset_id"))
			}
		}
	}

	// 未注册 key (20)
	for _, k := range []string{"fake", "source", "label", "category", "region", "team", "version", "status", "id", "created_at", "updated_at", "mcap_file_id", "asset_id", "is_deleted", "password", "secret", "token", "admin", "root", "DROP TABLE"} {
		code, _, lat := doReq("POST", "/api/v1/assets", mk(map[string]string{k: "v"}), nil)
		suite.record(TestResult{cat, fmt.Sprintf("未注册 %q", k), code == 422, code, 422, lat, ""})
	}

	// 组合 (5)
	for i, tags := range []map[string]string{
		{"priority": "high", "quality": "good"},
		{"priority": "low", "scene": "indoor", "task": "t"},
		{"priority": "critical", "quality": "excellent", "scene": "outdoor", "task": "n", "batch": "B1", "notes": "all"},
		{"priority": "high", "notes": strings.Repeat("备注", 100)},
		{"task": "", "batch": ""},
	} {
		code, resp, lat := doReq("POST", "/api/v1/assets", mk(tags), nil)
		suite.record(TestResult{cat, fmt.Sprintf("组合#%d", i+1), code == 201, code, 201, lat, ""})
		if code == 201 {
			deleteAsset(jsonGet(resp, "asset_id"))
		}
	}
}
