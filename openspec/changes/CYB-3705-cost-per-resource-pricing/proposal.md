# Proposal — CYB-3705

## Why

流水线成本估算在 delivery 集群(`cyber-clust`)上系统性虚高约 **10×**。样本 run `35961808-1ac6-482b-8175-6ef7a25f471e`(单 CPU 步骤 `sea-sky-delivery`,83m53s)显示 **$6.76**,真实应 ~$0.17–$0.63。

两个独立 bug 叠乘(均为历史修复的回归):

- **Bug A — `cpu` core-seconds 当 wall-clock。** `resourcesDurationToCost`(`cost.go:118`)取 `rd["cpu"]` 作 `totalSec` 再 `× 整机时价 / 3600`。但 Argo 的 `resourcesDuration.cpu` = `核数 × wall秒`(core-seconds),不是 wall-clock。4 核 pod 虚高 4×,16 核虚高 16×。CYB-1849(PR #222)曾改用 wall-clock `runtime_seconds` 修掉,但被 CYB-3073(PR #297,`a78e4340`)覆盖回 core-seconds。
- **Bug B — CPU-only 步骤 fallback 到 GPU 时价。** 机型解析不出时默认 `g2-standard-16`/`nvidia-l4` = **$1.20/hr**(`cost.go:91`)。CYB-3073 加的 `NodeInstanceResolver` 只有单个 clientset(`node_resolver.go:55`),指向默认 dev 集群;`cyber-clust` 的节点查不到 → 一律 fallback GPU 价,`gcp_pricing.yaml` 里的 CPU 档(c3d/t2d/e2)成了死配置。

**证据(对样本 run 反推)**:`$6.76 = coreSec × 1.20/3600` → `coreSec = 20,280`;wall = 5,033s → `cores ≈ 4.03`。一个 4 核 delivery pod,同时踩两 bug:`4.0 (core-seconds) × 2.7 (GPU vs c3d 价) ≈ 10.7×`。

## What Changes

把成本模型从「主导资源秒数 × 整机时价」改为 **按资源分项单价累加**——因为 Argo 的 `resourcesDuration` 本来就是「资源量 × 秒」的基准单位:

```
cost = cpu_core_seconds     × ($/vCPU-hour)   / 3600
     + gpu_seconds          × ($/GPU-hour)    / 3600
     + memory_100Mi_seconds × ($/100Mi-hour)  / 3600
```

### Modified Capabilities

1. **`backend/config/gcp_pricing.yaml`** — 结构从 `prices[instance][accel][prov] = 整机时价` 改为 `units[resource][provisioning] = 单位时价`。CPU 用一个跨机型的混合 vCPU 单价(各机型 per-vCPU 仅差 ~1.6×:E2 $0.0218 – C3 $0.0347),GPU(L4)单独按 per-GPU-hour 计价,memory 给一个很小的 per-100Mi-hour 价。保留 `calibration_factor`。
2. **`backend/internal/usecase/pipeline/cost.go`** — `resourcesDurationToCost` 改为分项累加;`provisioning`(spot/standard,3× 差异)仍从解析到的节点标签取,取不到时安全默认 `standard`(宁可对 spot 略高估,不会再 10× 爆)。删除已死的 `lookupHourlyRate` / `maxResourceDurationSeconds` / instance-fallback。`ComputeRunCost` 仍只累加叶子 Pod(CYB-3073 Bug 1 保持已修)。

### 关键收益

- **Bug A 消失**:core-seconds 正是要乘 per-vCPU-hour 的正确量纲,无需换算 wall-clock,不依赖核数。
- **Bug B 优雅降级**:GPU 成本仅在 `nvidia.com/gpu > 0` 时计入,CPU-only 步骤即使机型解析失败也永远不会继承 GPU 价;跨集群 node-resolver 的 RBAC 不再是 load-bearing。

## Impact

- **Affected code**: `backend/config/gcp_pricing.yaml`, `backend/internal/usecase/pipeline/cost.go`, 相关单测(`cost_test.go`, `usecase_cost_snapshot_test.go`)
- **New APIs**: 无(响应 shape 不变,仍是 `estimatedCostUsd` / `totalEstimatedCostUsd`,只是数值更准)
- **Dependencies**: 无新增;实际上**移除**了对 per-cluster node RBAC 的强依赖
- **数据**: 无 migration。历史 run 的 `estimated_cost_usd` 快照不会自动回填(除非该 run 再次 refresh);新 run 立即生效。

## Scope

- **In scope**: 成本模型重写 + pricing YAML + 单测;dev 部署验证样本 run 成本回落
- **Out of scope**: CUD/SUD、网络/存储 egress、prod 价格对账(CYB-1570);node-resolver 多集群化(本方案让它不再必需)

## Success Criteria

- [ ] 样本 run `35961808-…` 重新 refresh 后成本从 $6.76 回落到 ~$0.2–$0.6 量级
- [ ] CPU-only 步骤不再出现 GPU 单价;GPU 步骤成本与旧值同量级(回归保护)
- [ ] `go test ./internal/usecase/pipeline/...` 全绿(含改写后的成本单测)
