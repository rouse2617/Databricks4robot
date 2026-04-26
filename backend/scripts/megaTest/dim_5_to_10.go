package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ── 维度 5: 依赖链 unblock ──────────────────────────────────────────────────

func genDependencyChain(suite *TestSuite) {
	cat := "5.依赖链"
	// 完成 3 个依赖 → action_annotation unblock (重复 10 次)
	deps := []string{"hand_tracking@1.2.0", "head_tracking@1.0.0", "body_tracking@1.0.0"}
	for i := 0; i < 5; i++ {
		id, _ := createAsset(nil)
		if id == "" { continue }
		for _, dk := range deps {
			doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, dk), map[string]interface{}{"method": "t"}, nil)
			doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, dk), buildFinishBody(dk, "ok"), nil)
		}
		_, resp, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		st := jsonGetNested(resp, "algo_results", "action_annotation@1.0.0:status")
		suite.record(TestResult{cat, fmt.Sprintf("unblock#%d status=%s", i, st), st == "pending", 200, 200, lat, ""})
		deleteAsset(id)
	}

	// 部分依赖完成 → 仍 blocked (5 次, 只完成 2/3)
	for i := 0; i < 5; i++ {
		id, _ := createAsset(nil)
		if id == "" { continue }
		for _, dk := range deps[:2] { // 只完成前 2 个
			doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, dk), map[string]interface{}{"method": "t"}, nil)
			doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, dk), buildFinishBody(dk, "ok"), nil)
		}
		_, resp, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		st := jsonGetNested(resp, "algo_results", "action_annotation@1.0.0:status")
		suite.record(TestResult{cat, fmt.Sprintf("部分依赖#%d status=%s", i, st), st == "blocked", 200, 200, lat, ""})
		deleteAsset(id)
	}
}

// ── 维度 6: 幂等性 ──────────────────────────────────────────────────────────

func genIdempotency(suite *TestSuite) {
	cat := "6.幂等性"

	// finish 幂等 (相同 run_id) (10)
	for i := 0; i < 10; i++ {
		id, _ := createAsset(nil)
		if id == "" { continue }
		ak := "env_analysis@1.0.0"
		runID := fmt.Sprintf("idem-run-%d", i)
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "t", "run_id": runID}, nil)
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "ok", "run_id": runID}, nil)
		c, _, lat := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "ok", "run_id": runID}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("finish幂等#%d", i), c == 200, c, 200, lat, ""})
		// 不同 run_id → 409
		c, _, lat = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "ok", "run_id": "different"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("finish不同runid#%d", i), c == 409, c, 409, lat, ""})
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
		deleteAsset(id)
	}

	// delivery 幂等 (10)
	for i := 0; i < 10; i++ {
		id, _ := createAsset(nil)
		if id == "" { continue }
		idemKey := fmt.Sprintf("idem-del-%d-%d", time.Now().UnixNano(), i)
		body := map[string]interface{}{"asset_ids": []string{id}, "customer_id": "c", "owner": "o"}
		c1, r1, _ := doReq("POST", "/api/v1/deliveries", body, map[string]string{"Idempotency-Key": idemKey})
		c2, r2, lat := doReq("POST", "/api/v1/deliveries", body, map[string]string{"Idempotency-Key": idemKey})
		pass := c1 == 201 && c2 == 201 && jsonGet(r1, "delivery_id") == jsonGet(r2, "delivery_id")
		suite.record(TestResult{cat, fmt.Sprintf("delivery幂等#%d", i), pass, c2, 201, lat, ""})
		// 不同 body 同 key → 409
		body2 := map[string]interface{}{"asset_ids": []string{id}, "customer_id": "DIFFERENT", "owner": "o"}
		c3, _, lat := doReq("POST", "/api/v1/deliveries", body2, map[string]string{"Idempotency-Key": idemKey})
		suite.record(TestResult{cat, fmt.Sprintf("delivery冲突#%d", i), c3 == 409, c3, 409, lat, ""})
		deleteAsset(id)
	}
}

// ── 维度 7: 快速循环 ────────────────────────────────────────────────────────

func genRapidCycle(suite *TestSuite) {
	cat := "7.快速循环"
	ak := "env_analysis@1.0.0"

	// 5 个资产各循环 3 次 = 15 个循环
	for i := 0; i < 5; i++ {
		id, _ := createAsset(nil)
		if id == "" { continue }
		ok := 0
		for j := 0; j < 3; j++ {
			c1, _, _ := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": fmt.Sprintf("c%d", j)}, nil)
			c2, _, _ := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "ok"}, nil)
			c3, _, _ := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
			if c1 == 200 && c2 == 200 && c3 == 200 { ok++ }
		}
		suite.record(TestResult{cat, fmt.Sprintf("资产#%d 循环3次 ok=%d", i, ok), ok == 3, 200, 200, 0, ""})
		deleteAsset(id)
	}
}

// ── 维度 8: 并发 ────────────────────────────────────────────────────────────

func genConcurrency(suite *TestSuite) {
	cat := "8.并发"

	// 并发 start 同一算法 (5 轮 × 5 并发)
	for i := 0; i < 5; i++ {
		id, _ := createAsset(nil)
		if id == "" { continue }
		ak := "env_analysis@1.0.0"
		var wg sync.WaitGroup
		codes := make([]int, 5)
		for j := 0; j < 5; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				c, _, _ := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "conc"}, nil)
				codes[idx] = c
			}(j)
		}
		wg.Wait()
		ok200 := 0
		for _, c := range codes { if c == 200 { ok200++ } }
		suite.record(TestResult{cat, fmt.Sprintf("并发start#%d 200×%d", i, ok200), ok200 >= 1, 200, 200, 0, ""})
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "ok"}, nil)
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
		deleteAsset(id)
	}

	// 并发 PATCH 同一资产 (5 轮 × 5 并发)
	for i := 0; i < 5; i++ {
		id, _ := createAsset(nil)
		if id == "" { continue }
		var wg sync.WaitGroup
		for j := 0; j < 5; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				doReq("PATCH", "/api/v1/assets/"+id, map[string]interface{}{"reviewer": fmt.Sprintf("conc-%d", idx)}, nil)
			}(j)
		}
		wg.Wait()
		c, _, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("并发PATCH#%d", i), c == 200, c, 200, lat, ""})
		deleteAsset(id)
	}

	// 并发创建 (50 个并发创建)
	var wg sync.WaitGroup
	ids := make([]string, 50)
	codes := make([]int, 50)
	for j := 0; j < 50; j++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id, c := createAsset(nil)
			ids[idx] = id
			codes[idx] = c
		}(j)
	}
	wg.Wait()
	ok := 0
	for j := 0; j < 50; j++ {
		if codes[j] == 201 { ok++ }
	}
	suite.record(TestResult{cat, fmt.Sprintf("并发创建50个 ok=%d", ok), ok == 50, 200, 200, 0, ""})
	for _, id := range ids { if id != "" { deleteAsset(id) } }
}

// ── 维度 9: Unicode & 特殊字符 ──────────────────────────────────────────────

func genUnicode(suite *TestSuite) {
	cat := "9.Unicode特殊字符"
	ts := time.Now().UnixNano()
	end := ts + 60_000_000_000

	// 各种 Unicode 字符串作为 reviewer/owner (20)
	unicodeVals := []string{
		"张三", "田中太郎", "김철수", "Müller", "Ñoño", "Ωmega",
		"emoji🤖🚀💻", "混合abc中文123", "tab\there", "newline\nhere",
		"引号\"here\"", "反斜杠\\here", "斜杠/here", "尖括号<>here",
		"&amp;entity", "零宽\u200b字符", "RTL\u200ftext", "BOM\ufeffhere",
		strings.Repeat("中", 500), strings.Repeat("🤖", 100),
	}
	for _, v := range unicodeVals {
		body := map[string]interface{}{"mcap_file_id": fmt.Sprintf("uni-%d-%d", ts, rand.Intn(1e6)), "start_timestamp_ns": ts, "end_timestamp_ns": end, "reviewer": v, "owner": v}
		c, resp, lat := doReq("POST", "/api/v1/assets", body, nil)
		suite.record(TestResult{cat, fmt.Sprintf("创建 %q", truncStr(v, 15)), c == 201, c, 201, lat, ""})
		if c == 201 {
			id := jsonGet(resp, "asset_id")
			// 读回验证
			c2, resp2, lat2 := doReq("GET", "/api/v1/assets/"+id, nil, nil)
			pass := c2 == 200 && jsonGet(resp2, "owner") == v
			suite.record(TestResult{cat, fmt.Sprintf("往返 %q", truncStr(v, 15)), pass, c2, 200, lat2, ""})
			deleteAsset(id)
		}
	}

	// 特殊字符 mcap_file_id (10)
	specialIDs := []string{
		"file with spaces", "file#hash", "file@at", "file/slash",
		"file?query=1", "file%20encoded", "file+plus", "file=equals",
		"file;semicolon", "file&ampersand",
	}
	for _, mid := range specialIDs {
		body := map[string]interface{}{"mcap_file_id": mid, "start_timestamp_ns": ts, "end_timestamp_ns": end, "reviewer": "t"}
		c, resp, lat := doReq("POST", "/api/v1/assets", body, nil)
		suite.record(TestResult{cat, fmt.Sprintf("mcap_id %q", mid), c == 201, c, 201, lat, ""})
		if c == 201 { deleteAsset(jsonGet(resp, "asset_id")) }
	}
}

// ── 维度 10: 分页边界 ───────────────────────────────────────────────────────

func genPagination(suite *TestSuite) {
	cat := "10.分页边界"
	id, _ := createAsset(nil)
	if id == "" { return }
	defer deleteAsset(id)

	// deliveries 分页参数 (20)
	pageCases := []struct{ n, p, ps string; exp int }{
		{"正常", "1", "10", 200}, {"page=0", "0", "10", 200}, {"page=-1", "-1", "10", 200},
		{"page=999999", "999999", "10", 200}, {"ps=0", "1", "0", 200}, {"ps=-1", "1", "-1", 200},
		{"ps=999", "1", "999", 200}, {"ps=abc", "1", "abc", 200}, {"page=abc", "abc", "10", 200},
		{"both=abc", "abc", "xyz", 200}, {"empty", "", "", 200}, {"page=1.5", "1.5", "10", 200},
		{"ps=1e9", "1", "1000000000", 200}, {"page=0x10", "0x10", "10", 200},
		{"负大数", "-999999", "10", 200}, {"ps=MAX", "1", "2147483647", 200},
		{"page=MAX", "2147483647", "10", 200}, {"unicode", "一", "二", 200},
		{"空格", " 1 ", " 10 ", 200}, {"特殊", "1;DROP", "10", 200},
	}
	for _, tc := range pageCases {
		path := fmt.Sprintf("/api/v1/assets/%s/deliveries?page=%s&page_size=%s", id, tc.p, tc.ps)
		c, _, lat := doReq("GET", path, nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("分页 %s", tc.n), c == tc.exp, c, tc.exp, lat, ""})
	}
}
