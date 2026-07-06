# Design — CYB-3065

## Architecture Context
- **Constraints**: manifest 字段的读取方(`reconstruct_from_db.go`/`resource_usage.go`)已经统一用 `sigsyaml.Unmarshal`,序列化方式必须与之配对才能正确往返
- **Goals**: 新生成的 manifest 能被正确反序列化,资源数值不丢失
- **Non-Goals**: 修复/迁移历史已存在的 manifest 数据

## Affected Modules
- `backend/internal/usecase/pipeline/usecase.go` — manifest 序列化那一行(`yaml.Marshal` → `sigsyaml.Marshal`)

## Architecture Decisions

### Decision 1: 不做历史数据迁移,接受历史 manifest 的资源数值永久丢失
- **Approach**: 只改序列化代码,让**修复上线之后新产生**的 manifest 数据正确;历史数据保持原样,继续依赖现有的安全回退机制(反序列化失败 → fallback 到直连 Argo,不报错、不返回残缺数据)
- **Alternative**: 写一个迁移脚本,重新构造历史 manifest(比如从 `pipeline_run_nodes`/其他已持久化数据反推资源数值,重新生成一份 manifest 回写数据库)
- **Rationale**: 真实的资源数值(K8s Quantity 的内部表示)在原始序列化那一刻就已经丢失,没有任何地方还保留着这份原始信息可供"还原"——唯一的数据来源是当年提交时的 pipeline 定义,如果该定义本身没有被单独存档,就是真的拿不回来了。即使勉强从别处(如组件默认 resources 配置)反推出一个"大概"的数值回填,也无法保证与当年提交时的真实值一致,反而可能制造出看起来正确但实际是猜测的数据。相比之下,让历史数据继续走已经验证过是安全的 fallback 路径,是更诚实、风险更低的选择。
- **Trade-off**: 历史 run 在 CYB-3063 的"跳过 Argo"优化上不会有收益,只有修复上线后的新 run 才能享受到。
- **Risk**: 无——fallback 路径本身是 CYB-3063 已经设计好并验证过的安全机制。

## Risks / Trade-offs
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 序列化格式从 yaml.v3(全小写字段名)变成 sigsyaml(遵循 json tag 的大小写)| manifest 字段的原始文本样式改变 | 两种格式都能被 `sigsyaml.Unmarshal` 正确解析(json 字段匹配大小写不敏感),除了本次要修的 Quantity 字段外,其余字段的读取行为不受影响,不需要额外兼容处理 |
| 其他还未被发现的、依赖 manifest 具体文本格式的代码路径 | 未知 | 已通过 grep 核查全代码库,只有两处真正反序列化 manifest(reconstruct_from_db.go、resource_usage.go),均已确认兼容;没有其他代码路径依赖 manifest 的原始文本格式(如字符串匹配) |
