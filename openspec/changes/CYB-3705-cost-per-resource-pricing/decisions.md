# Decisions — CYB-3705

## 2026-07-20 — 成本模型:按资源分项单价 vs wall-clock × 整机价

- **Context**: 两个 bug 需同时修:(A) `cpu` core-seconds 被当 wall-clock;(B) 机型解析失败时 fallback GPU 价。三种候选:
  1. **CYB-1849 路线**:用 wall-clock `runtime_seconds`(StartedAt/FinishedAt)× 整机时价。修 A,但仍需解析机型定整机价——在 `cyber-clust` 上解析失败 → 仍走 GPU $1.20(2.7× 高)。B 未解决。
  2. **本方案**:按资源分项单价累加(cpu core-sec × per-vCPU-hr + gpu-sec × per-GPU-hr + mem)。
  3. 多集群化 node-resolver(每个 cluster 一个 clientset + RBAC)。
- **Decision**: 选方案 2。CPU 用跨机型混合 vCPU 单价 + `provisioning`(spot/standard)轴;GPU 仅在 `nvidia.com/gpu > 0` 时单独计价;机型解析失败时默认 `standard`。
- **Alternatives**: 方案 1 不解决跨集群 B;方案 3 是新基础设施 + off-limits(RBAC/Secret),违反「偏好最小化」且 per-vCPU 机型差仅 ~1.6×,不值当。
- **Rationale**:
  - Argo `resourcesDuration` 本就是「资源量 × 秒」,per-resource 单价是量纲正确的乘法,A 自然消失。
  - GPU 分项计价使 CPU-only 步骤永不继承 GPU 价,B 优雅降级,node-resolver 不再 load-bearing(跨集群无需 RBAC)。
  - per-vCPU 机型差 ~1.6× 远小于整机 GPU/CPU 的 2.7×,混合单价对「best-effort 估算」足够;spot 3× 差异用 provisioning 轴保留,取不到时默认 standard 只会略高估、不会再爆。
- **Trade-off accepted**: 混合 vCPU 单价对具体机型有 ±30% 偏差;spot 步骤在解析失败的集群上会按 standard 高估(≤3×,远好于旧的 10×)。历史 run 快照不自动回填。均属 best-effort 估算可接受范围。

## 2026-07-20 — OpenSpec checkpoint 批准

- 用户确认「可以,预估大概就行吧」,批准 per-resource 混合 vCPU 单价方案(非每机型单列),进入编码。
- 用户将提供 GCP 账单用于校准:L4 GPU 单价 + CUD/SUD 折扣 → 折进 unit rates / `calibration_factor`。listing 价先落地,账单到后替数字。

## 2026-07-20 — 在线配置化:跟进做,不入本 PR

- **Context**: 用户问 pricing 是否要进配置中心以便新机型即时改。
- **Decision**: 本 PR 只做 YAML fix;在线配置另开 CYB。
- **Rationale**: (1) 新分项混合单价模型下,**新机型自动继承混合 vCPU 单价,无需新增 pricing 条目**——"新机型要马上改"的诉求基本消失;真正要改的只剩混合单价/新 GPU 类型/校准系数,皆低频(季度级)。(2) 新机型引入本就是集群 onboarding(deploy-gated),同 PR 加一行 YAML 自然可追溯。(3) 在线配置是独立 feature:需 migration(off-limits 批准)或塞现有 JSONB + admin API 全套契约同步 + 注册中心 UI + 进程级 pricing 缓存热更。保持 `LoadPricing → PricingConfig` 为唯一加载点,以后接 DB/admin 加载器时成本函数零改动(前向兼容)。

## 2026-07-20 — 验证 Tier

- **Decision**: Tier M(`fmt`+`vet`+`go test ./internal/usecase/pipeline/...`)。仅 2 个非测试文件、无 router/handler/OpenAPI/shared-types 改动、响应 shape 不变。

## 2026-07-20 — 推迟部署验证到合并后(用户书面批准)

- **Context**: deploy-before-commit 要求提交前滚 dev 验证。但样本 run `35961808-…` 是冻结快照(Failed、Argo workflow 已 GC),部署新代码也不会改变其显示成本——只有新 delivery-clust run 才能 live 验证。且滚动共享 dev 会打断用户压测。
- **Decision**: 用户在 chat 明确选择「先 commit+PR,合并后自然验证」。跳过提交前的共享 dev 部署,依赖离线真实数据回放证据。
- **Evidence(离线,真实数据)**: 从 live dev API 拉到样本 run 的真实 `resourcesDuration = {cpu:20270, memory:814970}`(无 gpu key,host=`gke-delivery-clust-t2d-pool-…`)。经发布的 `gcp_pricing.yaml` + 新 `resourcesDurationToCost` 计算 = **$0.2595**(cpu $0.169 + mem $0.091,standard 默认),对比旧 **$6.7567**,降 26×。已固化为 `TestShippedPricingYAMLPricesDeliveryNodeSanely`。
- **Rationale**: 纯定价数学改动,无 schema/API/迁移;单测 + 真实数据端到端回放覆盖充分;合并后 dev 自动部署,新 delivery run 自然按新价。Rule precedence #1(用户显式指令)> #4(deploy gate)。
- **Follow-up**: 合并部署后,抽查一条新 delivery-clust run 的成本落在 CPU 量级即可确认 live 生效。
