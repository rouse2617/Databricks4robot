package main

import "fmt"

func genCRUDConsistency(suite *TestSuite) {
	cat := "3.CRUD一致性"

	// 写后读 (50)
	for i := 0; i < 50; i++ {
		id, c := createAsset(map[string]string{"priority": "high"})
		if c != 201 {
			suite.record(TestResult{cat, fmt.Sprintf("创建#%d", i), false, c, 201, 0, ""})
			continue
		}
		c2, resp, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		pass := c2 == 200 && jsonGet(resp, "asset_id") == id && jsonGet(resp, "status") == "approved" && jsonGet(resp, "reviewer") == "mega-tester"
		suite.record(TestResult{cat, fmt.Sprintf("写后读#%d", i), pass, c2, 200, lat, ""})
		deleteAsset(id)
	}

	// 部分更新不覆盖 (30)
	for i := 0; i < 30; i++ {
		id, _ := createAsset(map[string]string{"priority": "high", "quality": "good"})
		if id == "" {
			continue
		}
		doReq("PATCH", "/api/v1/assets/"+id, map[string]interface{}{"reviewer": "patched"}, nil)
		c, resp, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		pass := c == 200 && jsonGet(resp, "reviewer") == "patched" && jsonGet(resp, "owner") == "mega-team"
		suite.record(TestResult{cat, fmt.Sprintf("部分更新#%d", i), pass, c, 200, lat, ""})
		deleteAsset(id)
	}

	// 版本递增 (20 × 5 patches)
	for i := 0; i < 20; i++ {
		id, _ := createAsset(nil)
		if id == "" {
			continue
		}
		for j := 0; j < 5; j++ {
			doReq("PATCH", "/api/v1/assets/"+id, map[string]interface{}{"reviewer": fmt.Sprintf("v%d", j+2)}, nil)
		}
		_, resp, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("版本递增#%d expect=6 got=%s", i, jsonGet(resp, "version")), jsonGet(resp, "version") == "6", 200, 200, lat, ""})
		deleteAsset(id)
	}

	// 时间精度 (10)
	for i := 0; i < 10; i++ {
		id, _ := createAsset(nil)
		if id == "" {
			continue
		}
		_, resp, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		ca := jsonGet(resp, "created_at")
		suite.record(TestResult{cat, fmt.Sprintf("时间精度#%d", i), ca != "" && len(ca) > 10, 200, 200, lat, ""})
		deleteAsset(id)
	}

	// 软删除后 GET 仍可访问 (10)
	for i := 0; i < 10; i++ {
		id, _ := createAsset(nil)
		if id == "" {
			continue
		}
		doReq("DELETE", "/api/v1/assets/"+id, nil, nil)
		c, resp, lat := doReq("GET", "/api/v1/assets/"+id, nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("删后读#%d status=%s", i, jsonGet(resp, "status")), c == 200 && jsonGet(resp, "status") == "archived", c, 200, lat, ""})
	}

	// 重复删除幂等 (10)
	for i := 0; i < 10; i++ {
		id, _ := createAsset(nil)
		if id == "" {
			continue
		}
		doReq("DELETE", "/api/v1/assets/"+id, nil, nil)
		c, _, lat := doReq("DELETE", "/api/v1/assets/"+id, nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("重复删除#%d", i), c == 200, c, 200, lat, ""})
	}

	// 不存在的资产 GET (10)
	for i := 0; i < 10; i++ {
		c, _, lat := doReq("GET", fmt.Sprintf("/api/v1/assets/nonexistent-%d", i), nil, nil)
		suite.record(TestResult{cat, fmt.Sprintf("不存在GET#%d", i), c == 404, c, 404, lat, ""})
	}
}
