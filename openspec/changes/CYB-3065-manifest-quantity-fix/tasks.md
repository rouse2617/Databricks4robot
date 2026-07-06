# Tasks — CYB-3065

## Context files
```
backend/internal/usecase/pipeline/usecase.go       # manifest 序列化点(~L3320)
backend/internal/usecase/pipeline/resource_usage.go # templateResourcesFromManifest,确认其 fallback 逻辑不受影响
backend/internal/handlers/workflow/reconstruct_from_db.go  # CYB-3063 的唯一消费者,验证修复后能真正走通降级路径
```

## Implementation

- [ ] [backend] `usecase.go`:序列化 manifest 那一行,`yaml.Marshal(wf)` 改为 `sigsyaml.Marshal(wf)`(需要引入 `sigs.k8s.io/yaml`,若该文件尚未导入)
- [ ] [backend] 确认改动后原有的 `yaml` import(`gopkg.in/yaml.v3`)是否在该文件其他地方还有使用,若无则移除,避免遗留未使用的 import

## Scenario coverage(测试)

- [ ] [backend] 单测:序列化一个带 CPU/内存 requests/limits 的 pipeline 定义,反序列化后数值与原始声明一致 —— 覆盖 *"终态 run 的 pipeline 声明了资源 requests/limits"*
- [ ] [backend] 回归确认:`internal/handlers/workflow` 包已有的 CYB-3063 三个测试(`TestGetWorkflow_TerminalRunUsesDBReconstruction` 等)仍然通过

## API contract sync
N/A —— 内部序列化实现细节改动,不涉及 HTTP 接口。

## 验证(Tier M)
- [ ] `make fmt && make vet`
- [ ] `go test ./internal/usecase/pipeline/... ./internal/handlers/workflow/...`
- [ ] `go test ./...`

## 部署验证
- [ ] 部署到 Cloud Run dev,独立核实 revision 健康
- [ ] 提交一个真实的、声明了资源 requests/limits 的 pipeline 任务(比如现有模板 `2693fc7e-aa54-410e-ac50-665888c43f4c`,先确认其组件是否声明了资源,若没有则换一个已知声明了资源的模板),等其终态完成
- [ ] 用这个新产生的 run 调 `GET /api/v1/workflows/:name`,直接从数据库取出它的 manifest 用 `sigsyaml.Unmarshal` 验证能正确解析(而不是只看接口响应是否 200——响应 200 不能区分"真的走了降级路径"还是"fallback 到了 Argo 但看起来一样",需要独立验证 manifest 内容本身)
