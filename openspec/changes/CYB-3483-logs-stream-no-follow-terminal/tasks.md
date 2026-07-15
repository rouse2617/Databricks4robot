# Tasks

- [x] `argo/client.go` `GetWorkflowLogStream`:去掉写死 `opts.Follow=true`,尊重调用方
- [x] `handlers/workflow/logs_sse.go`:`opts.Follow = hasNode && !node.Fulfilled()`(节点级终态判定)
- [x] 单测:running 节点 → Follow=true;finished 节点 → Follow=false;StructuredEvents 回归
- [x] 编译 + vet + workflow/argo 包全量回归
- [ ] 部署 dev 后验证:已完成节点日志秒回(非 60s)、无新 `/logs/stream` 504、前端不卡
