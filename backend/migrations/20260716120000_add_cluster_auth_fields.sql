-- CYB-3486 PR auth.1: cluster auth abstraction
--
-- Prep for接第一个非-GCP 客户.  今天所有集群走 GKE Workload Identity Federation
-- 里 metadata-token 分支,单一路径写死在 k8s.buildConfigFromCluster。 客户环境要么
-- 是 ACK / EKS(其它 cloud provider metadata token),要么完全离线的自签 kubeconfig,
-- 都不再是"WIF audience"这一个字段能表达的。
--
-- 变更(只加不删):
--   1. clusters.auth_type text,default 'gke_wif' — 显式区分 metadata-token WIF、
--       bearer token(kubeconfig)、TODO ACK / EKS 的其它 metadata endpoint 等
--   2. clusters.auth_secret_ref text,default '' — 供非-WIF auth 存 Secret Manager
--       /  外挂 secret 的引用路径。 WIF 分支不用。 值为空就是"不走 secret",
--       和之前语义等价。
--
-- Backend 侧 config 构建按 auth_type dispatch;此 PR 的 auth_type 默认值让所有现存
-- cluster 行零回归,新增值(bearer / ack_wif / eks_wif)由后续 PR 支持。

ALTER TABLE "clusters"
  ADD COLUMN "auth_type"       text NOT NULL DEFAULT 'gke_wif',
  ADD COLUMN "auth_secret_ref" text NOT NULL DEFAULT '';

-- Backfill:所有现存行(包括 cluster-default seed)保持 gke_wif,值已由 default
-- 兜底,ALTER TABLE 语义会覆盖到已有行。
