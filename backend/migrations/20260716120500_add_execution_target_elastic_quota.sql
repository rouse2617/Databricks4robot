-- CYB-3486 PR pool.1: pool 概念的细粒度控制
--
-- 之前 CYB-3422 P3.1b 讨论过给 ExecutionTarget 加 elastic_quota_name,但受
-- Koordinator PoC 排期影响暂缓。 现在 Koord 生产已上,delivery-clust 也接入
-- 完成,可以把这一步走完:
--
-- 允许一个 ExecutionTarget 显式把自己"钉"到一个具体的 ElasticQuota 上。
-- 语义:
--   空(默认)= 走 namespace 层的默认 EQ(koord-scheduler 用 parent quota)
--   非空    = transpiler 会把 quota.scheduling.koordinator.sh/name=<value>
--              label 写到该 target 上跑的每个 workflow pod 上,koord 就把
--              这些 pod 的资源占用记到指定的 EQ 上,而不是 ns default 里。
--
-- 这样"资源池"UI 就能落到两种粒度:粗(namespace)+细(EQ);同一 ns 里
-- 多个 target 可以分别指向不同 EQ,方便按业务线切。

ALTER TABLE "execution_targets"
  ADD COLUMN "elastic_quota_name" text NOT NULL DEFAULT '';
