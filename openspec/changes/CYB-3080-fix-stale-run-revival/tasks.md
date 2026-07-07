# Tasks — CYB-3080

## Implementation
- [ ] [backend] `refreshRunStatus`：进入 `shouldWaitForWorkflowCreation` → `isPendingBatchWorkflowCreation` 的"等待创建"分支后，在清空 `run.Message` / 调用 `persistRunObservation` 之前，用 `uc.runRepo.FindByID` 重新核对当前持久化状态；如果当前状态已经不是 `isActiveDeploymentStatus`，跳过复活，直接返回。

## Local verification
- [ ] `cd backend && go test ./internal/usecase/pipeline/...`
- [ ] `cd backend && go test ./...`
- [ ] `make fmt && make vet`

### 测试要点
- [ ] 复现场景：run 的当前持久化状态是 `Failed`（带消息），但传入 `refreshRunStatus` 的内存快照仍是 `Pending`——验证不会被复活覆盖，`Failed` + 原始消息保持不变。
- [ ] 回归场景：run 当前持久化状态确实仍是 `Pending`/活跃——验证"等待创建"逻辑照常工作，不受这次改动影响。
- [ ] `reconcileMisclassifiedRunFromArgo` 现有测试全部保持通过（确认这次改动没有影响误判恢复路径）。

## Deploy verification
- [ ] 部署 backend dev。
- [ ] 在 dev 上用一个会被资源守卫拒绝的流水线（或直接复现之前卡住的场景）验证：run 被判定 `Failed` 后，即使批次列表页/轮询继续刷新，也不会被冲回 `Pending`。

## PR
- [ ] PR 描述包含 Linear ID（CYB-3080）与 OpenSpec change-id。
- [ ] 说明这是在排查 CYB-3071 相关问题时发现的独立 bug。
