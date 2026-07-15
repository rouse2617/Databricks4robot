# CYB-3422 P3.1a Decisions

## D1. 只读展示先行,Deploy 接入后行

### 选择

先做 P3.1a(只加 GET API + 前端 panel),等生产验证 UX + 数据后再做 P3.1b(schema 改动 + transpiler 注入)。

### 备选

- **一步到位**:一次 PR 同时改 schema + transpiler + Deploy + 前端
- **只做后端**:不做前端,先让 API 存在

### 拒绝理由

- 一步到位:回滚粒度大,且 transpiler 改动比较重,难以拆开;先只读能提前验证 UX 假设
- 只做后端:用户无法在 UI 上验证,失去 P3.1a 的 UX 验证价值

## D2. 数据源:后端每次现查 K8s,不缓存

### 选择

后端 handler 每次 API 调用现查 K8s(list ElasticQuota + 读 status),不加缓存。

### 备选

- **缓存 30s**:降低 K8s API 压力
- **前端直连 K8s**:通过 kube-apiserver 代理

### 拒绝理由

- 缓存 30s:用户点开面板时 `used` 值可能滞后 30s,和实时 usage 诉求冲突;而且 K8s API 单次 list 只 3 个对象,开销可忽略
- 前端直连 K8s:引入新的鉴权/CORS 问题,和现有 `/api/v1/resource-quotas` 模式不一致

## D3. UI 位置:Registry pools tab 里加 section(不新增页面)

### 选择

在现有 `/registry` → pools tab 里加一个 ElasticQuota section(现有 ExecutionTarget 表格下方)。

### 备选

- **独立菜单项**"资源池"(顶层导航)
- **和 ExecutionTarget 合并成一个表格**(在同一个表格里显示两种资源池)

### 拒绝理由

- 独立菜单项:违反用户要求"只在 pipeline 创建表单里下拉旁边显示(最轻,不做独立页)",且 UX 侧发现资源池的路径已经明确在 Registry 里
- 合并表格:ExecutionTarget 是 namespace 概念,ElasticQuota 是集群级配额概念,二者语义不同,合并反而混乱 —— P3.1b 才做绑定关系

## D4. 未装 Koordinator 环境的兼容

### 选择

后端处理 CRD 不存在错误(`meta.IsNoMatchError`)→ 返回 200 + 空 items;前端拿到空列表隐藏整个 section。

### 备选

- 返回 404 / 502
- 前端一直显示面板(即使空)

### 拒绝理由

- 4xx/5xx:老 dev 环境或未装 Koordinator 的分支会看到红色报错,影响体验
- 前端一直显示:未装 Koordinator 时看到"资源池 (0)"面板混淆用户

## D5. 不加编辑能力

### 选择

面板纯只读。ElasticQuota 由 kubectl / GitOps 管理,不通过 UI 建。

### 备选

- 面板加"创建 quota"按钮,弹窗填 min/max
- 支持删除 quota

### 拒绝理由

- 创建/删除 ElasticQuota 是**集群级**配置(会影响所有走 koord-scheduler 的 pod),不适合暴露给普通 pipeline 开发者;应由 SRE 通过 IaC 或 kubectl 管理
- 未来若需要通过 UI 建,可以在 P3.2+ 加,由权限系统保护
