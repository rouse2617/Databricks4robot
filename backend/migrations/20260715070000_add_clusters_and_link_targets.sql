-- CYB-3425 Phase B PR 1: 多集群支持 - schema 层
--
-- 现状:backend 假设单集群,k8s.NewClientset("") 和 argo.NewClient(...) 都是全局
-- 单例,从 env vars 读一次。execution_targets.cluster 字段虽然存在但只是自由字符
-- 串,未做集群路由。
--
-- 变更:新增 clusters 表存集群连接配置(K8s API endpoint + WIF audience + CA
-- data + Argo server URL);execution_targets 新增 cluster_id FK 硬绑到一个
-- cluster。老 target backfill 到默认集群,行为不变。runtime 后续 PR 才消费。
--
-- 3-phase safe migration in one file(target 数量极少,同一 tx):
--   1. 新表 clusters + seed 一个 default row(从 env-implied 配置派生)
--   2. execution_targets 加 cluster_id nullable
--   3. backfill 所有老 target → default cluster
--   4. cluster_id SET NOT NULL + FK
--
-- 未来扩展:cluster_id 也可指向 Karmada hub cluster(N > 5 时迁移)。

-- ┌─────────────────────────────────────────────────────────────────┐
-- │ 1. clusters 表                                                    │
-- └─────────────────────────────────────────────────────────────────┘
CREATE TABLE "clusters" (
  "id"                text NOT NULL,
  "name"              text NOT NULL,
  "display_name"      text NOT NULL DEFAULT '',
  "description"       text NOT NULL DEFAULT '',
  "is_default"        boolean NOT NULL DEFAULT false,
  "status"            text NOT NULL DEFAULT 'available',
  -- K8s 连接配置(WIF 用 metadata token,不存 bearer token)
  "k8s_api_endpoint"  text NOT NULL DEFAULT '',
  "k8s_audience"      text NOT NULL DEFAULT '',
  "k8s_ca_data"       text NOT NULL DEFAULT '',
  -- Argo 连接配置(argo-server URL 和默认 namespace)
  "argo_server_url"   text NOT NULL DEFAULT '',
  "argo_namespace"    text NOT NULL DEFAULT '',
  -- 标记该集群是否装了 Koordinator(前端 ElasticQuota panel 判断是否显示)
  "koord_installed"   boolean NOT NULL DEFAULT false,
  "created_at"        timestamptz NOT NULL DEFAULT now(),
  "updated_at"        timestamptz NOT NULL DEFAULT now(),
  "deleted_at"        timestamptz,
  PRIMARY KEY ("id")
);

-- name 唯一(排除已软删的)
CREATE UNIQUE INDEX "idx_clusters_name_active"
  ON "clusters" ("name") WHERE "deleted_at" IS NULL;

-- 只允许一个 is_default 集群
CREATE UNIQUE INDEX "idx_clusters_one_default"
  ON "clusters" ("is_default") WHERE "is_default" AND "deleted_at" IS NULL;

-- ┌─────────────────────────────────────────────────────────────────┐
-- │ 2. Seed default cluster row(从当前 env 派生的 placeholder)      │
-- └─────────────────────────────────────────────────────────────────┘
-- 值留空:backend 读到空字段时 fallback 到 env vars(K8S_API_ENDPOINT / K8S_AUDIENCE
-- / K8S_CA_DATA / ARGO_SERVER_URL 等),保持向后兼容。管理员后续可通过 CRUD API
-- 填入具体值,填后 backend 就用 DB 里的值而不是 env。
INSERT INTO "clusters" ("id", "name", "display_name", "description", "is_default", "status", "koord_installed")
VALUES (
  'cluster-default',
  'default',
  'Default cluster',
  'Legacy single-cluster (config from env vars). Backend reads K8S_* / ARGO_* env when DB fields are empty.',
  true,
  'available',
  true  -- cyber-clust 已装 Koord(CYB-3420)
);

-- ┌─────────────────────────────────────────────────────────────────┐
-- │ 3. execution_targets 加 cluster_id(nullable)                    │
-- └─────────────────────────────────────────────────────────────────┘
ALTER TABLE "execution_targets" ADD COLUMN "cluster_id" text;

-- ┌─────────────────────────────────────────────────────────────────┐
-- │ 4. Backfill 所有老 target → default cluster                       │
-- └─────────────────────────────────────────────────────────────────┘
UPDATE "execution_targets" SET "cluster_id" = 'cluster-default' WHERE "cluster_id" IS NULL;

-- ┌─────────────────────────────────────────────────────────────────┐
-- │ 5. NOT NULL + FK                                                 │
-- └─────────────────────────────────────────────────────────────────┘
ALTER TABLE "execution_targets"
  ALTER COLUMN "cluster_id" SET NOT NULL,
  ADD CONSTRAINT "execution_targets_cluster_id_fkey"
    FOREIGN KEY ("cluster_id") REFERENCES "clusters" ("id") ON DELETE RESTRICT;

CREATE INDEX "idx_execution_targets_cluster_id"
  ON "execution_targets" ("cluster_id");
