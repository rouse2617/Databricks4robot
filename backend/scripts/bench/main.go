// Bigtable API 压测工具
// 用法: go run ./scripts/bench -url http://localhost:8080 -c 10 -n 200
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	baseURL   = flag.String("url", "http://localhost:8080", "后端地址")
	token     = flag.String("token", "dev-token", "认证 token")
	conc      = flag.Int("c", 10, "并发数")
	total     = flag.Int("n", 200, "总请求数")
	scenario  = flag.String("s", "all", "场景: create|get|algo|lifecycle|all")
	keepAsset = flag.Bool("keep", false, "保留创建的资产 (不清理)")
)

// ── 统计 ────────────────────────────────────────────────────────────────────

type stats struct {
	mu        sync.Mutex
	latencies []time.Duration
	codes     map[int]int
	errors    int64
}

func newStats() *stats {
	return &stats{codes: map[int]int{}}
}

func (s *stats) record(d time.Duration, code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latencies = append(s.latencies, d)
	s.codes[code]++
}

func (s *stats) recordErr() {
	atomic.AddInt64(&s.errors, 1)
}

func (s *stats) report(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := len(s.latencies)
	if n == 0 {
		fmt.Printf("  %-25s  no data\n", label)
		return
	}
	sort.Slice(s.latencies, func(i, j int) bool { return s.latencies[i] < s.latencies[j] })

	sum := time.Duration(0)
	for _, d := range s.latencies {
		sum += d
	}
	avg := sum / time.Duration(n)
	p50 := s.latencies[n*50/100]
	p95 := s.latencies[n*95/100]
	p99 := s.latencies[n*99/100]
	maxL := s.latencies[n-1]

	errs := atomic.LoadInt64(&s.errors)
	okCount := 0
	for code, cnt := range s.codes {
		if code >= 200 && code < 300 {
			okCount += cnt
		}
	}

	fmt.Printf("  %-25s  n=%-5d  avg=%-8s  p50=%-8s  p95=%-8s  p99=%-8s  max=%-8s  ok=%d  err=%d\n",
		label, n, avg.Round(time.Millisecond), p50.Round(time.Millisecond),
		p95.Round(time.Millisecond), p99.Round(time.Millisecond),
		maxL.Round(time.Millisecond), okCount, errs)

	// 状态码分布
	codes := make([]int, 0, len(s.codes))
	for c := range s.codes {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	parts := make([]string, 0, len(codes))
	for _, c := range codes {
		parts = append(parts, fmt.Sprintf("%d×%d", c, s.codes[c]))
	}
	fmt.Printf("  %-25s  codes: %s\n", "", strings.Join(parts, "  "))
}

// ── HTTP 工具 ───────────────────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 30 * time.Second}

func doReq(method, path string, body interface{}) (int, []byte, time.Duration, error) {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, *baseURL+path, reader)
	if err != nil {
		return 0, nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Grace-Token", *token)

	start := time.Now()
	resp, err := httpClient.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return 0, nil, elapsed, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody, elapsed, nil
}

// ── 场景 ────────────────────────────────────────────────────────────────────

// benchCreate: POST /api/v1/assets
func benchCreate(st *stats, wg *sync.WaitGroup, ids chan<- string, count int) {
	defer wg.Done()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	envs := []string{"indoor", "outdoor", "warehouse", "office", "factory"}
	tasks := []string{"pick_and_place", "navigation", "inspection", "assembly"}

	for i := 0; i < count; i++ {
		ts := time.Now().UnixNano()
		body := map[string]interface{}{
			"mcap_file_id":       fmt.Sprintf("bench-mcap-%d-%d", ts, rng.Intn(100000)),
			"start_timestamp_ns": ts,
			"end_timestamp_ns":   ts + int64(rng.Intn(60)+1)*1e9,
			"reviewer":           fmt.Sprintf("bench-user-%d", rng.Intn(10)),
			"owner":              fmt.Sprintf("team-%d", rng.Intn(5)),
			"type":               "task_demo",
			"env":                envs[rng.Intn(len(envs))],
			"task":               tasks[rng.Intn(len(tasks))],
		}
		code, respBody, elapsed, err := doReq("POST", "/api/v1/assets", body)
		if err != nil {
			st.recordErr()
			continue
		}
		st.record(elapsed, code)
		if code == 201 {
			var m map[string]interface{}
			json.Unmarshal(respBody, &m)
			if id, ok := m["asset_id"].(string); ok {
				ids <- id
			}
		}
	}
}

// benchGet: GET /api/v1/assets/:id
func benchGet(st *stats, wg *sync.WaitGroup, ids []string, count int) {
	defer wg.Done()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < count; i++ {
		id := ids[rng.Intn(len(ids))]
		code, _, elapsed, err := doReq("GET", "/api/v1/assets/"+id, nil)
		if err != nil {
			st.recordErr()
			continue
		}
		st.record(elapsed, code)
	}
}

// benchAlgoLifecycle: start → finish → reset 完整流程
func benchAlgoLifecycle(st *stats, wg *sync.WaitGroup, ids []string, count int) {
	defer wg.Done()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	algoKey := "env_analysis@1.0.0"

	for i := 0; i < count; i++ {
		id := ids[rng.Intn(len(ids))]

		// start
		code, _, elapsed, err := doReq("POST",
			fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, algoKey),
			map[string]interface{}{"method": "bench", "run_id": fmt.Sprintf("bench-run-%d", i)})
		if err != nil {
			st.recordErr()
			continue
		}
		st.record(elapsed, code)
		if code != 200 {
			continue // 可能已经 running
		}

		// finish
		code, _, elapsed, err = doReq("POST",
			fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, algoKey),
			map[string]interface{}{"status": "ok"})
		if err != nil {
			st.recordErr()
			continue
		}
		st.record(elapsed, code)

		// reset
		code, _, elapsed, err = doReq("POST",
			fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, algoKey), nil)
		if err != nil {
			st.recordErr()
			continue
		}
		st.record(elapsed, code)
	}
}

// benchGetEvents: GET /api/v1/assets/:id/events
func benchGetEvents(st *stats, wg *sync.WaitGroup, ids []string, count int) {
	defer wg.Done()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < count; i++ {
		id := ids[rng.Intn(len(ids))]
		code, _, elapsed, err := doReq("GET", "/api/v1/assets/"+id+"/events", nil)
		if err != nil {
			st.recordErr()
			continue
		}
		st.record(elapsed, code)
	}
}

// ── 主流程 ──────────────────────────────────────────────────────────────────

func main() {
	flag.Parse()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Bigtable API 压测  url=%s  concurrency=%d  total=%d  scenario=%s\n",
		*baseURL, *conc, *total, *scenario)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 健康检查
	code, _, _, err := doReq("GET", "/healthz", nil)
	if err != nil || code != 200 {
		fmt.Fprintf(os.Stderr, "健康检查失败: code=%d err=%v\n", code, err)
		os.Exit(1)
	}
	fmt.Println("  ✓ 健康检查通过")
	fmt.Println()

	runAll := *scenario == "all"

	// ── Phase 1: Create ─────────────────────────────────────────────────
	var createdIDs []string

	if runAll || *scenario == "create" || *scenario == "lifecycle" {
		createN := *total
		if runAll {
			createN = *total / 4
			if createN < 20 {
				createN = 20
			}
		}

		fmt.Printf("▶ 创建资产 (n=%d, c=%d)\n", createN, *conc)
		createStats := newStats()
		idCh := make(chan string, createN)
		var wg sync.WaitGroup

		perWorker := createN / *conc
		remainder := createN % *conc
		start := time.Now()

		for w := 0; w < *conc; w++ {
			n := perWorker
			if w < remainder {
				n++
			}
			wg.Add(1)
			go benchCreate(createStats, &wg, idCh, n)
		}
		wg.Wait()
		close(idCh)
		wallTime := time.Since(start)

		for id := range idCh {
			createdIDs = append(createdIDs, id)
		}
		fmt.Printf("  创建了 %d 个资产, 耗时 %s (%.1f req/s)\n",
			len(createdIDs), wallTime.Round(time.Millisecond),
			float64(createN)/wallTime.Seconds())
		createStats.report("POST /assets")
		fmt.Println()
	}

	// 如果没有创建资产，先创建几个用于后续测试
	if len(createdIDs) == 0 {
		fmt.Println("▶ 预创建 10 个资产用于读取测试...")
		idCh := make(chan string, 10)
		var wg sync.WaitGroup
		wg.Add(1)
		go benchCreate(newStats(), &wg, idCh, 10)
		wg.Wait()
		close(idCh)
		for id := range idCh {
			createdIDs = append(createdIDs, id)
		}
		fmt.Printf("  预创建了 %d 个资产\n\n", len(createdIDs))
	}

	if len(createdIDs) == 0 {
		fmt.Fprintln(os.Stderr, "没有可用的资产 ID，终止")
		os.Exit(1)
	}

	// ── Phase 2: Get ────────────────────────────────────────────────────
	if runAll || *scenario == "get" {
		getN := *total
		if runAll {
			getN = *total / 4
		}

		fmt.Printf("▶ 读取资产 (n=%d, c=%d)\n", getN, *conc)
		getStats := newStats()
		var wg sync.WaitGroup

		perWorker := getN / *conc
		remainder := getN % *conc
		start := time.Now()

		for w := 0; w < *conc; w++ {
			n := perWorker
			if w < remainder {
				n++
			}
			wg.Add(1)
			go benchGet(getStats, &wg, createdIDs, n)
		}
		wg.Wait()
		wallTime := time.Since(start)

		fmt.Printf("  耗时 %s (%.1f req/s)\n", wallTime.Round(time.Millisecond), float64(getN)/wallTime.Seconds())
		getStats.report("GET /assets/:id")
		fmt.Println()
	}

	// ── Phase 3: Algo lifecycle ─────────────────────────────────────────
	if runAll || *scenario == "algo" || *scenario == "lifecycle" {
		algoN := *total
		if runAll {
			algoN = *total / 4
			if algoN < 10 {
				algoN = 10
			}
		}

		fmt.Printf("▶ 算法生命周期 start→finish→reset (n=%d, c=%d)\n", algoN, *conc)
		algoStats := newStats()
		var wg sync.WaitGroup

		perWorker := algoN / *conc
		remainder := algoN % *conc
		start := time.Now()

		for w := 0; w < *conc; w++ {
			n := perWorker
			if w < remainder {
				n++
			}
			wg.Add(1)
			go benchAlgoLifecycle(algoStats, &wg, createdIDs, n)
		}
		wg.Wait()
		wallTime := time.Since(start)

		fmt.Printf("  耗时 %s (%.1f ops/s, 每 op 含 3 个请求)\n",
			wallTime.Round(time.Millisecond), float64(algoN)/wallTime.Seconds())
		algoStats.report("algo lifecycle")
		fmt.Println()
	}

	// ── Phase 4: Get events ─────────────────────────────────────────────
	if runAll || *scenario == "algo" {
		evN := *total
		if runAll {
			evN = *total / 4
		}

		fmt.Printf("▶ 查询算法事件 (n=%d, c=%d)\n", evN, *conc)
		evStats := newStats()
		var wg sync.WaitGroup

		perWorker := evN / *conc
		remainder := evN % *conc
		start := time.Now()

		for w := 0; w < *conc; w++ {
			n := perWorker
			if w < remainder {
				n++
			}
			wg.Add(1)
			go benchGetEvents(evStats, &wg, createdIDs, n)
		}
		wg.Wait()
		wallTime := time.Since(start)

		fmt.Printf("  耗时 %s (%.1f req/s)\n", wallTime.Round(time.Millisecond), float64(evN)/wallTime.Seconds())
		evStats.report("GET /events")
		fmt.Println()
	}

	// ── 清理 ────────────────────────────────────────────────────────────
	if !*keepAsset && len(createdIDs) > 0 {
		fmt.Printf("▶ 清理 %d 个测试资产...\n", len(createdIDs))
		for _, id := range createdIDs {
			doReq("DELETE", "/api/v1/assets/"+id, nil)
		}
		fmt.Println("  ✓ 清理完成")
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  压测完成")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
