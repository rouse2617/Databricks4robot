package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"
)

var (
	baseURL    = flag.String("url", "http://localhost:8080", "后端地址")
	token      = flag.String("token", "dev-token", "认证 token")
	reportFile = flag.String("report", "test_report.md", "报告输出文件")
)

func main() {
	flag.Parse()
	suite := &TestSuite{}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Bigtable API 千级测试  url=%s\n", *baseURL)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 健康检查
	c, _, _ := doReq("GET", "/healthz", nil, nil)
	if c != 200 {
		fmt.Fprintf(os.Stderr, "健康检查失败: %d\n", c)
		os.Exit(1)
	}
	fmt.Println("  ✓ 健康检查通过")

	start := time.Now()

	run := func(name string, fn func(*TestSuite)) {
		before := suite.pass + suite.fail
		fmt.Printf("▶ %s ...\n", name)
		fn(suite)
		after := suite.pass + suite.fail
		fmt.Printf("  完成 +%d 用例 (累计 %d)\n\n", after-before, after)
	}

	run("维度1: 输入校验", genCreateValidation)
	run("维度2: Tag校验", genTagValidation)
	run("维度3: CRUD一致性", genCRUDConsistency)
	run("维度4: 算法状态机", genAlgoStateMachine)
	run("维度5: 依赖链", genDependencyChain)
	run("维度6: 幂等性", genIdempotency)
	run("维度7: 快速循环", genRapidCycle)
	run("维度8: 并发", genConcurrency)
	run("维度9: Unicode特殊字符", genUnicode)
	run("维度10: 分页边界", genPagination)
	run("维度11: Delivery", genDelivery)
	run("维度12: CommitSegments", genCommitSegments)
	run("维度13: 错误响应格式", genErrorFormat)
	run("维度14: HTTP方法路由", genHTTPMethods)
	run("维度15: 认证", genAuth)
	run("维度16: AlgoEvents", genAlgoEvents)
	run("维度17: 安全", genSecurity)
	run("维度18: 大数据量", genLargeData)
	run("维度19: 空值处理", genEmptyVsMissing)
	run("维度20: 随机Fuzz", func(s *TestSuite) { genFuzz(s, 1000) })

	elapsed := time.Since(start)

	// 生成报告
	generateReport(suite, elapsed)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  总计: %d pass / %d fail / %d total  耗时 %s\n", suite.pass, suite.fail, suite.pass+suite.fail, elapsed.Round(time.Second))
	fmt.Printf("  报告: %s\n", *reportFile)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if suite.fail > 0 {
		os.Exit(1)
	}
}

func generateReport(suite *TestSuite, elapsed time.Duration) {
	f, err := os.Create(*reportFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建报告失败: %v\n", err)
		return
	}
	defer f.Close()

	w := func(format string, args ...interface{}) { fmt.Fprintf(f, format, args...) }

	w("# Bigtable API 测试报告\n\n")
	w("- 日期: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	w("- 后端: %s\n", *baseURL)
	w("- 耗时: %s\n", elapsed.Round(time.Second))
	w("- 总计: **%d** pass / **%d** fail / **%d** total\n", suite.pass, suite.fail, suite.pass+suite.fail)
	w("- 通过率: **%.1f%%**\n\n", float64(suite.pass)/float64(suite.pass+suite.fail)*100)

	// 按维度汇总
	catStats := map[string][2]int{} // [pass, fail]
	catOrder := []string{}
	seen := map[string]bool{}
	for _, r := range suite.results {
		if !seen[r.Category] {
			catOrder = append(catOrder, r.Category)
			seen[r.Category] = true
		}
		s := catStats[r.Category]
		if r.Pass {
			s[0]++
		} else {
			s[1]++
		}
		catStats[r.Category] = s
	}

	w("## 维度汇总\n\n")
	w("| 维度 | Pass | Fail | Total | 通过率 |\n")
	w("|------|------|------|-------|--------|\n")
	for _, cat := range catOrder {
		s := catStats[cat]
		total := s[0] + s[1]
		rate := float64(s[0]) / float64(total) * 100
		status := "✅"
		if s[1] > 0 {
			status = "❌"
		}
		w("| %s %s | %d | %d | %d | %.0f%% |\n", status, cat, s[0], s[1], total, rate)
	}

	// 延迟分布
	w("\n## 延迟分布\n\n")
	var lats []time.Duration
	for _, r := range suite.results {
		if r.Latency > 0 {
			lats = append(lats, r.Latency)
		}
	}
	if len(lats) > 0 {
		sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
		n := len(lats)
		sum := time.Duration(0)
		for _, l := range lats {
			sum += l
		}
		w("| 指标 | 值 |\n")
		w("|------|----|\n")
		w("| 请求数 | %d |\n", n)
		w("| 平均 | %s |\n", (sum / time.Duration(n)).Round(time.Millisecond))
		w("| P50 | %s |\n", lats[n*50/100].Round(time.Millisecond))
		w("| P95 | %s |\n", lats[n*95/100].Round(time.Millisecond))
		w("| P99 | %s |\n", lats[n*99/100].Round(time.Millisecond))
		w("| Max | %s |\n", lats[n-1].Round(time.Millisecond))
	}

	// 失败详情
	var failures []TestResult
	for _, r := range suite.results {
		if !r.Pass {
			failures = append(failures, r)
		}
	}
	if len(failures) > 0 {
		w("\n## 失败用例详情\n\n")
		w("| # | 维度 | 用例 | 期望 | 实际 | 延迟 |\n")
		w("|---|------|------|------|------|------|\n")
		for i, r := range failures {
			w("| %d | %s | %s | %d | %d | %s |\n", i+1, r.Category, truncStr(r.Name, 50), r.Expected, r.Code, r.Latency.Round(time.Millisecond))
		}
	}

	// 全部用例明细 (折叠)
	w("\n## 全部用例明细\n\n")
	w("<details>\n<summary>展开查看全部 %d 个用例</summary>\n\n", len(suite.results))
	w("| # | 状态 | 维度 | 用例 | HTTP | 延迟 |\n")
	w("|---|------|------|------|------|------|\n")
	for i, r := range suite.results {
		status := "✅"
		if !r.Pass {
			status = "❌"
		}
		w("| %d | %s | %s | %s | %d | %s |\n", i+1, status, r.Category, truncStr(r.Name, 60), r.Code, r.Latency.Round(time.Millisecond))
	}
	w("\n</details>\n")
}

// json package needed for dim_11_to_20.go
var _ = json.Unmarshal
