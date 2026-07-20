-- 资源池无法删除修复 (SQLSTATE 23503)。
--
-- pipeline_runs.execution_target_id 是 NOT NULL + FK ON DELETE NO ACTION,
-- 所以任何被历史 run 引用过的 execution_target(资源池)都删不掉:
--   ERROR: update or delete on table "execution_targets" violates foreign key
--   constraint "pipeline_runs_execution_target_id_fkey" on table "pipeline_runs"
--
-- run 已有 target_snapshot(jsonb NOT NULL)保存池的快照,历史信息不依赖这个 FK。
-- 改为 nullable + ON DELETE SET NULL(与相邻 template_id FK 同一模式):删池时
-- 历史 run 的 execution_target_id 置 NULL,池信息继续看 target_snapshot。应用层
-- 新建 run 仍会填 execution_target_id;nullable 只为允许删池后置空。

ALTER TABLE "pipeline_runs" ALTER COLUMN "execution_target_id" DROP NOT NULL;

ALTER TABLE "pipeline_runs"
  DROP CONSTRAINT "pipeline_runs_execution_target_id_fkey";
ALTER TABLE "pipeline_runs"
  ADD CONSTRAINT "pipeline_runs_execution_target_id_fkey"
  FOREIGN KEY ("execution_target_id") REFERENCES "execution_targets" ("id")
  ON UPDATE NO ACTION ON DELETE SET NULL;
