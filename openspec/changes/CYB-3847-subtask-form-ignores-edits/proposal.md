# CYB-3847: fix subscription-task form ignoring user edits

## Problem

订阅任务面板抽屉里的 `拉取间隔（秒）` / `单次最大消息数` / `GCP 项目 ID` 三个字段虽然能改，但保存后取值不变：编辑维持原值，新建强制默认。

## Root cause

`SubscriptionTasksPanel.tsx` `handleSubmit` 里这三个字段用 `editingTask?.X ?? default`，**完全忽略** `form.validateFields()` 拿到的 `vals.X`。CYB-3778 表单简化时，注释按"字段不 surface"设计，但字段其实还在表单里 —— 注释与实现脱节，是回归。

## Fix

改成 `vals.X ?? editingTask?.X ?? default`：优先取用户表单输入，回退到原值，最后到默认。同步删除误导注释。

## Scope

Frontend 单文件 3 行的正确性修复；无 API 契约变更、无 migration、无鉴权改动、无新字段。

## Verification

- 手工：编辑任务 10→20，保存后列表/回读均为 20（Chrome DevTools MCP on dev）。
- 自动：`tsc --noEmit` + `npm run build`。
