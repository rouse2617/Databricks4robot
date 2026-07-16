# 新增执行集群 onboarding checklist（多集群接入）

接入**一个全新的 K8s 集群**到 DataBrew 多集群执行体系（CYB-3486）时，需要一次性配置的每集群清单。接 `delivery-clust` 时全部踩过一遍，固化成清单避免漏项。

**范围区分：**

- 本文 = 接入**一个新集群**（新 K8s API、新 `clusters` 行）。
- 同一个已接入集群里**再加一个业务 namespace / execution target** → 见 [`AI-RULES.md`](AI-RULES.md) 的「Execution targets (Argo namespaces)」小节（`video-proc-dev` 复制模板）。本文的第 2 项即引用它。

**两种提交模式（决定配置重点）：**

| 模式 | 触发条件 | backend 如何提交 |
|------|----------|------------------|
| **argo-server (HTTP)** | `clusters.argo_server_url` 非空 | 走 argo-server HTTP API（`cyber-clust` 现状） |
| **CRD 直提** | `clusters.argo_server_url` **留空** | backend 用 K8s dynamic client 直接建 Workflow CRD（`delivery-clust` 现状） |

> ⚠️ **只有 `is_default` 集群**的空 `argo_server_url` 才会 fallback 到 env `ARGO_SERVER_URL`；非默认集群留空 = **明确走 CRD**，不吃 env（[#436 修复](../../backend/internal/argo/factory.go) `configForCluster`）。给新集群误填 `cyber-clust` 的 URL → workflow 打到 `cyber-clust` 的 argo-server → `namespaces xxx not found`。

---

## 清单总览

| # | 配置项 | 类型 | 可代码托底? |
|---|--------|------|-------------|
| 1 | backend GSA → 新集群 K8s API 的 WIF + RBAC | GCP IAM + K8s RBAC | 否（每集群一次性授权） |
| 2 | argo workflows 安装 + 每业务 ns 的 KSA/Role/RoleBinding | K8s RBAC | 否（引用 AI-RULES 复制模板） |
| 3 | 节点 SA 的 Artifact Registry 拉镜像权限 | GCP IAM | **是**（imagePullSecret 注入） |
| 4 | `databrew-run-webhook-token` secret 存在于业务 ns | K8s Secret | **是**（submit 时 ensure secret） |
| 5 | `workflow-runner` KSA ↔ 有 GCS 读写权限的 GSA（WI） | GCP IAM | 否（每集群一次性授权） |
| 6 | `clusters` 表新行 | DataBrew 配置 | 否（admin UI/CRUD） |
| 7 | `execution_targets` 建 target 指向新集群 | DataBrew 配置 | 否（admin UI/CRUD） |

---

## 1. backend GSA → 新集群 K8s API：WIF + RBAC 〔GCP IAM + K8s RBAC〕

**做什么**

- backend 的 GSA（dev: `cyber-databrew-dev@green-valley-442103.iam.gserviceaccount.com`；prod 对应 GSA）要能拿到新集群 K8s API 的 token。`auth_type=gke_wif`（默认）走 GKE metadata token（[`buildConfigGKEWIF`](../../backend/internal/k8s/factory.go)，要求 `k8s_audience` + `k8s_ca_data` 非空，见第 6 项）。
- 在**新集群**里给这个 GSA 建 RBAC：`workflows`（argoproj CRD）/ `pods` / `nodes` / `elasticquotas` 的 CRUD `ClusterRole` + `ClusterRoleBinding`。
- GKE 会把 GSA 解析成 **User subject（GSA 邮箱）**，所以 `ClusterRoleBinding.subjects` 要写 `kind: User, name: <gsa-email>`，**不是** `ServiceAccount`。

**为什么**

CRD 模式下 backend 直接用 K8s client 提交/读 Workflow CRD、列 pods/nodes、读 ElasticQuota。这条 RBAC 漏一项 → `Unauthorized` / `create runtime config projection: Unauthorized`。

---

## 2. argo workflows 安装 + 每业务 namespace 的 KSA/Role/RoleBinding 〔K8s RBAC〕

**做什么**

- 新集群装 `argoproj` CRD + `workflow-controller`（namespaced 模式，`--namespaced`）。
- **每个业务 namespace** 复制以下（模板见 [`AI-RULES.md` → Execution targets](AI-RULES.md)，参考 `video-proc-dev`）：
  1. SA `workflow-runner` + SA `argo-workflows-workflow-controller`
  2. Role `argo-workflows-workflow`（**含 `workflowtaskresults` 的 `create`/`patch`**）+ Role `argo-workflows-workflow-controller` + Role `cyber-databrew-backend-runtime-config`
  3. 每个 Role 对应的 RoleBinding → 对应 SA
  4. `workflow-controller` Deployment + `argo-workflows-workflow-controller-configmap`（从已有 ns 拷）

**为什么**

`workflowtaskresults create/patch` 最容易漏：漏了 → step 容器跑完但 controller 收不到结果，node 一直卡 `Running`。

---

## 3. 节点 SA 拉镜像权限：Artifact Registry reader（跨项目） 〔GCP IAM〕

**做什么**

- 新集群 node pool 的**节点 GSA** 需要镜像所在 Artifact Registry 的 `roles/artifactregistry.reader`。
- 镜像在**别的 GCP project** 时，binding 加在**源 project 的 repo 上**（授权对象是新集群节点 GSA），不是目标 project。

**为什么**

GKE + Artifact Registry 默认用**节点 GSA** 拉镜像。跨项目时源 project 不认新集群节点 GSA → `ImagePullBackOff`。这是 `delivery-clust` 的遗留跨项目镜像权限问题。

**可代码托底（方向）** ✅

transpiler 已支持 [`Options.ImagePullSecrets`](../../backend/internal/transpiler/transpiler.go)（注入 `wf.Spec.imagePullSecrets`）。未来可在 submit 时按 target 注入 `imagePullSecret`，用 pull secret 代替「节点 SA 跨项目授权」，把这项从「每集群手工 IAM」变成「代码自动」。

---

## 4. exit-hook secret：`databrew-run-webhook-token` 存在于业务 namespace 〔K8s Secret〕

**做什么**

- 在新集群的**每个业务 namespace** 建 Secret `databrew-run-webhook-token`（backend 配置 `ArgoRunWebhookTokenSecretName`/`Key`，见 [`core.go`](../../backend/cmd/server/core.go)）。
- transpiler 给每个 workflow 注入工作流级 exit hook（[`databrew-exit-notify`](../../backend/internal/transpiler/transpiler.go) 模板）：exit-notify pod `curl` backend 回调，token 从这个 Secret 挂载。

**为什么**

exit hook 是主 status 信号（CYB-3058）。Secret 缺失 → exit-notify pod 起不来 → status push 失效，只能靠 watcher 兜底（有延迟，且 watcher 也依赖第 1/6 项的 cluster 路由正确）。

**可代码托底（方向）** ✅

submit 时由 backend 确保 target namespace 存在该 Secret（ensure/create）。
⚠️ 但 [`AI-RULES.md`](AI-RULES.md) 规定「**Do NOT create or modify K8s Secrets without explicit approval；secret 内容必须来自用户**」——代码托底的 token 值必须来自受控来源（Secret Manager），不能代码里硬编码。

---

## 5. `workflow-runner` KSA ↔ 有 GCS 读写权限的 GSA（Workload Identity） 〔GCP IAM〕

**做什么**

- 业务 namespace 的 `workflow-runner` KSA 通过 Workload Identity 绑到一个**有目标 GCS bucket 读写权限**的 GSA（真实 pipeline pod 读输入 / 写输出）。
- 两步：
  1. `gcloud iam service-accounts add-iam-policy-binding <GSA> --role roles/iam.workloadIdentityUser --member "serviceAccount:<PROJECT>.svc.id.goog[<NS>/workflow-runner]"`
  2. KSA 加 annotation `iam.gke.io/gcp-service-account=<GSA>`
- 该 GSA 需要 bucket 的 `roles/storage.objectAdmin`（或更细的 object 读写）。

**为什么**

漏了 → pipeline pod 读/写 GCS `403`。注意这个 GSA（数据面）与第 1 项 backend GSA（控制面）职责不同，别混用。

---

## 6. `clusters` 表新行 〔DataBrew 配置〕

**做什么**（建议走 admin UI / CRUD API，会 `Invalidate` factory 缓存；不要手 `psql`，见 [`AI-RULES.md`](AI-RULES.md) migration 规矩）

`clusters` 关键字段（表结构见 [`migrations/20260715070000`](../../backend/migrations/20260715070000_add_clusters_and_link_targets.sql) + [`20260716120000`](../../backend/migrations/20260716120000_add_cluster_auth_fields.sql)）：

| 字段 | 填什么 |
|------|--------|
| `name` / `display_name` | 集群标识（如 `delivery-clust`） |
| `is_default` | `false`（只允许一个 default） |
| `k8s_api_endpoint` | 新集群 K8s API host（如 `https://<ip>`） |
| `k8s_audience` | WIF token audience（`gke_wif` 必填，否则 `ErrClusterMisconfigured`） |
| `k8s_ca_data` | 集群 CA（base64 或 PEM，backend 两种都认） |
| `auth_type` | 默认 `gke_wif`；保留值 `bearer`/`ack_wif`/`eks_wif` 未接 |
| `argo_server_url` | **留空 = CRD 模式**（见顶部 ⚠️）；非空 = 走该 argo-server |
| `argo_namespace` | 该集群业务 ns |

---

## 7. `execution_targets` 建 target 指向新集群 〔DataBrew 配置〕

**做什么**

- 建 `execution_targets` 行（admin UI / CRUD），`cluster_id` = 第 6 项 `clusters.id`，`namespace` = 该集群业务 ns，`service_account` / quota 按需。
- `cluster_id` 是 `NOT NULL` FK → `clusters(id)`（[migration `20260715070000`](../../backend/migrations/20260715070000_add_clusters_and_link_targets.sql)）；不填默认 `cluster-default`。

**为什么**

pipeline run 通过 execution target 路由到 cluster：submit 用 `argoFactory.ForTarget(target)`；watcher / lifecycle（stop/retry/delete/resource-usage）用 `resolveRunClusterID(run)` → `execution_targets.cluster_id`（见 [`usecase.go`](../../backend/internal/usecase/pipeline/usecase.go)）。target 没建或 `cluster_id` 错 → run 打到错集群。

---

## 端到端验证

接完 7 项后，跑一条小 pipeline 到新 target，确认：

1. **submit 成功** — run 进入 `Running`，新集群里出现 Workflow CRD + pod。
2. **镜像拉起** — 无 `ImagePullBackOff`（第 3 项）。
3. **状态回传** — node 状态更新；exit-notify pod 成功（第 4 项），否则确认 watcher 兜底能收敛。
4. **GCS 输出** — 输出 asset 写入 bucket，无 `403`（第 5 项）。
5. **生命周期** — 对该 run 执行 stop / delete，确认作用在新集群（`resolveRunClusterID` 路由），不是 `cyber-clust`。

---

## 代码托底 roadmap（已讨论方向）

| 项 | 现状 | 代码托底方向 | 现成 seam |
|----|------|--------------|-----------|
| #3 拉镜像 | 每集群手工给节点 SA 加 `artifactregistry.reader` | submit 时按 target 注入 `imagePullSecret` | `transpiler.Options.ImagePullSecrets` 已存在 |
| #4 exit-hook secret | 每业务 ns 手工建 `databrew-run-webhook-token` | submit 时 backend ensure target ns 有该 secret | 受 AI-RULES「不建 secret」约束，token 需来自受控来源 |

其余项（#1/#2/#5 授权，#6/#7 建行）是一次性配置，短期不值得代码化——等接入集群数 > 一两个、出现第二个真实场景再抽象。
