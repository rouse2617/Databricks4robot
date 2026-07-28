-- CYB-3778: subscription_tasks 从「单模板」改为「多模板 fan-out」。
-- 一个 topic/subscription 拉到的每条消息,对绑定的每个流水线模板各下发一个批次。
-- pipeline_bindings 元素形如 {"templateId","templateVersion"(可空),"targetId"}。
-- dev 表当前为空、prod 尚无此表,DROP/ADD 列安全。

ALTER TABLE subscription_tasks DROP COLUMN IF EXISTS template_id;
ALTER TABLE subscription_tasks DROP COLUMN IF EXISTS template_version;
ALTER TABLE subscription_tasks DROP COLUMN IF EXISTS target_id;
ALTER TABLE subscription_tasks DROP COLUMN IF EXISTS last_batch_id;

ALTER TABLE subscription_tasks ADD COLUMN pipeline_bindings JSONB NOT NULL DEFAULT '[]';
ALTER TABLE subscription_tasks ADD COLUMN last_batch_ids    JSONB NOT NULL DEFAULT '[]';
