# CYB-3826: PoolManager 调度配置列展示不全 — 修

## Problem

流水线 → 资源池管理的「调度配置」列,只有部分 target 的调度配置能显示;新建的 `rtx-flex-start (video-proc-prod)` 明明配了 `templateNodeSelector` + `tolerations`,列里却显示「—」。交付集群池同理,`scheduling.tolerations` 也从不展示(只显示 podLabels + schedulerName)。

## Root cause

`Frontend/src/components/pipeline/PoolManager.tsx` 调度配置列 render(约 568-583 行)读取路径不一致:

- `podLabels` / `schedulerName` / `priorityClassName` 从 `rd.scheduling.*` 读 ✓
- `templateTolerations` / `templateNodeSelector` **只**从 `rd.*` **顶层**读 ✗

但真实数据都在 `rd.scheduling.*` 子对象下。后端 `backend/internal/usecase/pipeline/scheduling.go`(`mapValue`)是**别名兼容**读法(顶层 + 嵌套 + 多个键名都认),所以调度实际生效,只是 UI 看不到。

## Solution

改 render 逻辑,与后端别名对齐:

- **tolerations**:`sched.templateTolerations` ?? `sched.tolerations` ?? `rd.templateTolerations` ?? `rd.tolerations` ?? []
- **templateNodeSelector**:`sched.templateNodeSelector` ?? `sched.nodeSelector` ?? `sched.nodeSelectors` ?? `rd.templateNodeSelector` ?? `rd.nodeSelector` ?? `rd.nodeSelectors` ?? {}

后端 `scheduling.go:19,71` 定义的正是这几个别名。TS 类型(`Frontend/src/api/pipelineApi.ts` 的 `TargetScheduling`)按需补齐可选字段。

**纯 UI 显示修**:无 API 变更、无 migration、不改鉴权、不改后端。

## Out of scope

- gpuStepNodeSelector 的展示(后端有别名 `gpuStepNodeSelector`/`gpuNodeSelector`,当前列没显示——不在本 PR 范围)。
- podAnnotations 展示。
- Pool 编辑表单的字段名统一(编辑逻辑另论)。
