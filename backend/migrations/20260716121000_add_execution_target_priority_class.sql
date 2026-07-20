-- CYB-3486 PR pool.2: 显式优先级
--
-- 目前 workflow-controller 里所有 pod 优先级一样(K8s default PriorityClass
-- 或 preempt-lower-priority=Never 的没配置)。 现在业务能表达"这个 target 属于
-- 高优业务线,应该抢先跑"或"这个是 batch,遇到高优先级排队等"。
--
-- 复用 K8s PriorityClass 机制:
--   - 空(默认)= workflow 不 set PodPriorityClassName,K8s 走 global-default
--   - 非空    = transpiler 把 PriorityClassName 写到 Workflow.Spec.PodPriorityClassName
--
-- 与 pool.1(EQ 定资源池)正交:pool.1 决定占哪个池的额度,pool.2 决定池内
-- 优先级。 常见组合:同一 EQ 里有 prod 和 batch 两条 target,batch 挂低优先级
-- 让 prod 抢先跑。

ALTER TABLE "execution_targets"
  ADD COLUMN "priority_class_name" text NOT NULL DEFAULT '';
