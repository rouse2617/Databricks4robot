-- CYB-3246 Phase 2: 标签注册表 DB 化。
--
-- 现状:受管标签定义写死在 backend/config/tag_registry.yaml,新增/修改需改文件 +
-- 重启后端。本表把标签定义搬到 DB,让管理员可通过 admin API / Settings 界面自助
-- 注册,无需改配置或重启(启动时若表空则从 YAML 播种;运行时以 DB 为准,校验热
-- 路径仍只读内存 map)。
--
-- 说明:
--   type='enum' 时 values 为允许值列表;type='string' 时 max_length 生效
--   (0 = 无限制)。propagation='descendants' 表示标签自动传播到子资产(CYB-1068)。
--   开放词汇(CYB-3246 Phase 1):未注册 key 仍作自由字符串接受,不入本表。
CREATE TABLE "tag_registry" (
  "key"         text NOT NULL,
  "description" text NOT NULL DEFAULT '',
  "type"        text NOT NULL,
  "values"      text[] NOT NULL DEFAULT '{}',
  "max_length"  integer NOT NULL DEFAULT 0,
  "propagation" text NOT NULL DEFAULT 'none',
  "created_by"  text NOT NULL DEFAULT '',
  "created_at"  timestamptz NOT NULL DEFAULT now(),
  "updated_at"  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("key"),
  CONSTRAINT "tag_registry_type_check" CHECK (type = ANY (ARRAY['enum'::text, 'string'::text])),
  CONSTRAINT "tag_registry_propagation_check" CHECK (propagation = ANY (ARRAY['none'::text, 'descendants'::text]))
);
