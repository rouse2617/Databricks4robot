# Proposal — CYB-3065

## Why
`pipeline_runs.manifest` 用 `yaml.Marshal(wf)`(yaml.v3 反射式序列化)存储,K8s `resource.Quantity` 类型(CPU/内存的 request/limit)真实数值存在私有字段里,反射序列化拿不到,只留下 `format` 字段——任何声明了资源的 pipeline,manifest 反序列化都会失败。这让 CYB-3063(终态 run 跳过直连 Argo)的降级路径在生产环境里可能从未真正生效过。

## What Changes

### Modified Capabilities
- pipeline: manifest 序列化改用 `sigsyaml.Marshal`(与反序列化用同一个库),CPU/内存等资源数值不再丢失

## Impact
- **Affected code**: `backend/internal/usecase/pipeline/usecase.go`(序列化点)
- **New APIs**: 无
- **Dependencies**: 无新增(`sigs.k8s.io/yaml` 已是现有依赖,`reconstruct_from_db.go`/`resource_usage.go` 已经在用)

## Scope
- **In scope**: manifest 生成时的序列化方式
- **Out of scope**:
  - 数据库里已存在的历史 manifest 数据——真实数值在序列化那一刻已经丢失,无法恢复,不做迁移
  - `templateResourcesFromManifest` 的调用逻辑本身——它已经有 fallback 机制,manifest 读取失败不影响其最终结果,不需要额外处理

## Success Criteria
- [ ] 修复上线后新创建的 run,其 manifest 能被 `sigsyaml.Unmarshal` 正确解析,资源数值(cpu/memory 等)与提交时的原始值一致
- [ ] 历史(修复上线前)已存在的 manifest 继续保持现状——反序列化失败时安全 fallback,不报错、不返回残缺数据(沿用 CYB-3063 已有的回退机制,本身不用改)
