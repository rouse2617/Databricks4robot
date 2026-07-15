# CYB-3422 P3.1a — Koordinator ElasticQuota 只读展示(资源池 UX)

## Why

CYB-3420 已在 GKE 生产集群交付 3 个 ElasticQuota(`cyberorigin-delivery-high` / `-mid` / `-low`)+ Koordinator 精简档 + Cloud Monitoring dashboard,提供了「多业务不同紧急程度、大额度、弹性借用」的资源池能力。

但当前 UX 层**看不到这些资源池** —— 用户和运维只能通过 `kubectl get elasticquota` 才知道池的 min / max / used。同时,DataBrew 现有 Registry 页面(`/registry` → pools tab)只有 `ExecutionTarget`(1:1 K8s namespace)的概念,没有集群级 ElasticQuota 的展示。

### 为什么先只做只读展示(P3.1a),而不是一步到位接入 Deploy(P3.1b)?

生产接入 Koordinator 是渐进过程:

- **P3.1a(本次)**:只加只读 API + 前端展示。**不改 transpiler**,不改 Deploy 逻辑,不改 ExecutionTarget schema。上线后集群调度行为完全不变,零业务风险 —— 但用户可以在 UI 上看到 3 个资源池的实时状态。
- **P3.1b(下一次)**:ExecutionTarget schema 加 `elastic_quota_name` nullable 字段,transpiler 生成 pod spec 时注入 3 要素(`schedulerName: koord-scheduler` + `priorityClassName: koord-*` + `quota.scheduling.koordinator.sh/name` label)。这才是真正的「Deploy 接入 quota」,涉及调度器路径切换,风险等级更高。

P3.1a 是 P3.1b 的前置观察窗口:上线后先在 registry 页面观察 quota 状态是否真实、UX 是否符合预期,再决定是否推进 P3.1b。

## What Changes

**Backend — 新增只读 API `GET /api/v1/elastic-quotas`**

- 新文件 `backend/internal/handlers/workflow/elastic_quota.go`,和现有 `resource_quota.go` 同模式(K8s client 现查,不缓存)
- 通过 in-cluster kubeconfig list `scheduling.sigs.k8s.io/v1alpha1` `ElasticQuota` 对象(所有 namespace)
- 返回 JSON:
  ```json
  {
    "items": [
      {
        "name": "cyberorigin-delivery-high",
        "namespace": "cyber-databrew-dev",
        "min": {"cpu": "4", "memory": "8Gi"},
        "max": {"cpu": "24", "memory": "48Gi"},
        "used": {"cpu": "0", "memory": "0"},
        "utilizationPercent": {"cpu": 0, "memory": 0}
      }
    ]
  }
  ```
- 路由注册在 `backend/routes/routes.go` 现有 `resource-quotas` 附近
- 无缓存(和 `resource-quotas` 一致);每次现查 K8s,latency ~200-500ms 可接受
- 错误处理:K8s API 失败 → 502;ElasticQuota CRD 不存在 → 200 + 空列表(未装 Koordinator 的集群兼容)

**Frontend — Registry pools tab 加 ElasticQuota 面板**

- `Frontend/src/api/pipelineApi.ts` 加 `listElasticQuotas()` 函数
- `Frontend/src/components/pipeline/PoolManager.tsx` 加一个新 section(现有 ExecutionTarget 表格下方),展示 ElasticQuota 列表
- 表格列:名字 · Namespace · CPU(min/max/used + usage bar)· Memory(min/max/used)· 最近刷新时间
- 面板顶部有说明:「资源池由 Koordinator 提供集群级弹性配额,平时空闲时可互相借用,上限硬约束」
- 15 秒轮询刷新(和现有 ExecutionTarget 面板一致)
- ElasticQuota CRD 不存在时(未装 Koordinator 的环境)不显示面板(空列表 = 隐藏 section)

**不做(明确排除,留给 P3.1b/P3.2)**

- ❌ 不改 `ExecutionTarget` 表结构或 model —— 老 target 完全保持不变
- ❌ 不改 `transpiler.Options` 或 `applyTemplateSchedulingDefaults` —— pod 生成路径不变
- ❌ 不改 `Deploy` usecase —— 部署行为不变
- ❌ 不加 ExecutionTarget → ElasticQuota 绑定关系(P3.1b 做)
- ❌ 不加编辑 quota 的能力 —— ElasticQuota 由 kubectl / GitOps 管理(chart 和运维路径,不通过 UI 建)

## 兼容性 & 回滚

- **零迁移**:不动任何现有数据
- **零调度改动**:所有 pipeline 继续走 default-scheduler
- **回滚**:revert PR 即可(前端 section 消失,后端 API 消失)
- **未装 Koordinator 的环境**:API 返回空列表,前端面板隐藏 —— 老 dev 环境不受影响

## 后续(P3.1b,不在本次)

- ExecutionTarget 加 `elastic_quota_name`(nullable text)
- Transpiler 生成 pod 时按 `elastic_quota_name` 注入 3 要素
- PoolManager 编辑弹窗支持选择绑定的 quota
