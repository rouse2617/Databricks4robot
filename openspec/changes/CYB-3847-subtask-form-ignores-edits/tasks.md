# CYB-3847 Tasks

- [x] Linear CYB-3847（child of CYB-3778）
- [x] OpenSpec proposal
- [ ] Fix `handleSubmit`: prefer `vals.X` over `editingTask?.X` for `projectId`, `pullIntervalSeconds`, `maxMessagesPerPull`; drop stale comment
- [ ] Tier S+ 验证：`tsc --noEmit` + `npm run build`
- [ ] Dev 部署 + Chrome DevTools MCP 验证：编辑任务 10→20 保存后回读 20；新建任务 60 → 创建后 GET 返回 60
- [ ] PR（Linear + OpenSpec + 模板）
