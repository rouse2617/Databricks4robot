-- CYB-3486 pool 重构:池调度配置统一到 resource_defaults JSONB,后端零硬编码。
--
-- pool.1 / pool.2 曾给 execution_targets 加了 elastic_quota_name /
-- priority_class_name 两个专用列。 问题:每加一个池属性(scheduler、
-- annotation、别的 quota 机制…)就要 ALTER TABLE + 改 model + 改 repo,不可
-- 扩展;而且后端要硬编码 koord EQ label key / koord-scheduler 才能用它们。
--
-- 改为:池的所有调度/配额指令放进 resource_defaults 的 "scheduling" 子对象
-- (JSONB),后端通用注入(schedulerName / podLabels / podAnnotations /
-- priorityClassName / nodeSelector / tolerations),不认识 koord、不硬编码任何
-- key/value。 新增池属性只需 JSONB 加 key,不动 schema。
--
-- 这两列还没被任何 target 使用(pool.1/2 merge 后没配过),drop 无数据损失。

ALTER TABLE "execution_targets" DROP COLUMN IF EXISTS "elastic_quota_name";
ALTER TABLE "execution_targets" DROP COLUMN IF EXISTS "priority_class_name";
