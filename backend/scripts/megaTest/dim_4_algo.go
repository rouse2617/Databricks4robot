package main

import (
	"fmt"
	"strings"
)

func genAlgoStateMachine(suite *TestSuite) {
	cat := "4.算法状态机"
	algoKeys := []string{"env_analysis@1.0.0", "hand_tracking@1.0.0", "hand_tracking@1.2.0", "head_tracking@1.0.0", "body_tracking@1.0.0", "deface@2.0.0"}

	// 完整生命周期 (6×3=18)
	for _, ak := range algoKeys {
		id, _ := createAsset(nil)
		if id == "" { continue }
		c, _, l := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "t"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s start", ak), c == 200, c, 200, l, ""})
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), buildFinishBody(ak, "ok"), nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s finish", ak), c == 200, c, 200, l, ""})
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s reset", ak), c == 200, c, 200, l, ""})
		deleteAsset(id)
	}

	// 非法转换 (2 algos × 7 = 14)
	for _, ak := range []string{"env_analysis@1.0.0", "hand_tracking@1.2.0"} {
		id, _ := createAsset(nil)
		if id == "" { continue }
		c, _, l := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "ok"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s pending→finish", ak), c == 409, c, 409, l, ""})
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s pending→reset", ak), c == 409, c, 409, l, ""})
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "t"}, nil)
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "t"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s running→start", ak), c == 409, c, 409, l, ""})
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s running→reset", ak), c == 409, c, 409, l, ""})
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), buildFinishBody(ak, "ok"), nil)
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "t"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s ok→start", ak), c == 409, c, 409, l, ""})
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "ok"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s ok→finish", ak), c == 409, c, 409, l, ""})
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
		deleteAsset(id)
	}

	// failed 场景 (6×2=12)
	for _, ak := range algoKeys {
		id, _ := createAsset(nil)
		if id == "" { continue }
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, ak), map[string]interface{}{"method": "t"}, nil)
		c, _, l := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "failed"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s failed无reason", ak), c == 422, c, 422, l, ""})
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, ak), map[string]interface{}{"status": "failed", "reason": "err"}, nil)
		suite.record(TestResult{cat, fmt.Sprintf("%s failed有reason", ak), c == 200, c, 200, l, ""})
		doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, ak), nil, nil)
		deleteAsset(id)
	}

	// 非法 key (20)
	id, _ := createAsset(nil)
	if id != "" {
		for _, bk := range []string{"fake@1.0.0", "env_analysis@9.9.9", "hand_tracking@0.0.0", "", "no_ver", "@1.0.0", "env_analysis@", "env_analysis", "hand_tracking@1.0.0.0", "HAND_TRACKING@1.0.0", "env-analysis@1.0.0", "env analysis@1.0.0", "../../../etc/passwd", "<script>", strings.Repeat("a", 200) + "@1.0.0", "DROP TABLE", "null", "undefined", "true", "0"} {
			c, _, l := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, bk), map[string]interface{}{"method": "t"}, nil)
			suite.record(TestResult{cat, fmt.Sprintf("非法key %q", truncStr(bk, 25)), c == 400 || c == 404, c, 400, l, ""})
		}
		deleteAsset(id)
	}

	// blocked 算法 (3)
	id, _ = createAsset(nil)
	if id != "" {
		aa := "action_annotation@1.0.0"
		c, _, l := doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/start", id, aa), map[string]interface{}{"method": "t"}, nil)
		suite.record(TestResult{cat, "blocked start", c == 409, c, 409, l, ""})
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/finish", id, aa), map[string]interface{}{"status": "ok"}, nil)
		suite.record(TestResult{cat, "blocked finish", c == 409, c, 409, l, ""})
		c, _, l = doReq("POST", fmt.Sprintf("/api/v1/assets/%s/algo/%s/reset", id, aa), nil, nil)
		suite.record(TestResult{cat, "blocked reset", c == 409, c, 409, l, ""})
		deleteAsset(id)
	}
}
