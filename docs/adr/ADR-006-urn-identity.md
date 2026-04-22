# ADR-006:外部 API 统一 URN 命名

- **Status**: Accepted(Phase 0 冻结)
- **Date**: 2026-04-22
- **Related**: [06-borrowed-patterns.md](../06-borrowed-patterns.md) / [ADR-007](ADR-007-write-path-and-event-contract.md)
- **Scope 修订**:本 ADR 适用于**外部边界**(public API、事件 subject、浏览器 URL、审计日志)。内部服务间 gRPC wire 允许继续用 UUID,详见 ADR-007。

---

## Context

平台中存在多种 entity(asset、mcap file、tag、user、algo、run、dataset、export),跨服务 / UI / 事件 / 外部接口都需要指代。
若每个地方自定义 ID 格式,会出现:
- 事件载荷里的 "id" 字段难以解析
- URL 路由难以统一
- 审计难以关联
- 系统外集成难以互通

## Decision

**采用 URN(Uniform Resource Name)作为全平台 entity 标识**,格式借鉴 DataHub:

```
urn:grace:<entity_type>:<entity_key>
```

- `grace`:namespace,固定(将来多租户时可改为 `grace:<tenant>`)
- `entity_type`:受控字典(见下)
- `entity_key`:该类型内部唯一的 key,可以是 UUID、URI、composite key

## URN 规范

| Entity Type | Key 格式 | 示例 |
| --- | --- | --- |
| `asset` | `<uuid>` | `urn:grace:asset:a1b2c3d4-5e6f-7890-abcd-ef1234567890` |
| `mcap` | `<uri>` | `urn:grace:mcap:gs://grace-raw-mcap/.../robot42.mcap` |
| `segment` | `<asset_id>:<sub_id>`(物化子段) | `urn:grace:segment:a1b2c3:s1` |
| `file` | `<asset_id>:<kind>:<version>` | `urn:grace:file:a1b2c3:sam2_v3:3.1.2` |
| `tag` | `<classification>.<tag>` | `urn:grace:tag:Scene.urban` |
| `classification` | `<name>` | `urn:grace:classification:Scene` |
| `glossary` | `<term>` | `urn:grace:glossary:valid_segment` |
| `user` | `<email>` | `urn:grace:user:alice@company.com` |
| `algo` | `<name>@<version>` | `urn:grace:algo:sam2@v3.1.2` |
| `run` | `<dagster_run_id>` | `urn:grace:run:dagster-run-abc123` |
| `dataset` | `<name>@<version>` | `urn:grace:dataset:train_2026q1` |
| `export` | `<export_id>` | `urn:grace:export:exp-xyz` |
| `tenant` | `<name>` | `urn:grace:tenant:cyber-grace` |

**规则**:
- URN **全小写**(除了 asset UUID 是原样)
- `:` 在 key 内需 URL encode
- 最大长度 2048 字节(适配 URL)
- URN 是**不可变**的,entity 改名不动 URN

## 边界规则(关键)

| 面 | ID 形态 | 责任人 |
| --- | --- | --- |
| 浏览器 URL / BFF REST / GraphQL | **URN** | BFF |
| SDK 对外 API(`Urn` 对象) | **URN** | grace-sdk |
| MCE / MCL 事件 `subject` | **URN** | asset-service(发事件时包装) |
| 审计日志 / Cloud Logging | **URN** | asset-service、BFF |
| **内部服务 gRPC wire**(asset-service ↔ mcap-gateway ↔ search-service) | **UUID**(及类型 enum) | 各服务 |
| **Bigtable row key 组件 / PG 主键** | **UUID** | asset-service |

**转换边界**:
- BFF 进入 → URN parse → UUID,出去 → UUID wrap → URN
- SDK `Urn` 对象内部持有 `.uuid` 属性,SDK 发 gRPC 时转 UUID
- 所有在事件 / 日志 / 响应体上的 id **必须 URN**,不暴露裸 UUID

## Alternatives Considered

| 方案 | 为什么不选 |
| --- | --- |
| 纯 UUID | 丢失类型信息,事件载荷看不懂 |
| 复合字符串(`asset:<uuid>`) | 没有 namespace,外部集成冲突 |
| URL 格式(`/assets/<uuid>`) | 耦合 API 路径变更 |
| URI(`grace://asset/...`) | 各工具支持不一致;URN 更标准 |

## Consequences

### Positive
- ✅ 事件 / 日志 / 审计跨系统可追溯
- ✅ 与业界标准(DataHub)对齐
- ✅ 跨服务传递时类型明确,parse 友好
- ✅ 外部集成时有标准身份

### Negative
- ⚠️ 有点冗长(但 gzip / protobuf 压缩后影响小)
- ⚠️ 需要严格维护 entity_type 字典

### Mitigations
- SDK 提供 `Urn` 类封装,`Urn.parse()` / `Urn.asset(id)` 等工厂方法
- Schema registry(`fields.yaml`)列出所有合法 entity_type

## Acceptance Criteria

- 所有**外部** API 响应的 id 字段都是 URN 格式
- 所有 MCE / MCL 事件的 subject 字段是 URN
- SDK 提供 `Urn` 类及单元测试覆盖 100%
- BFF 中间件:请求体内 URN → UUID、响应体内 UUID → URN 的转换必写单测
- 内部 gRPC `.proto` 文件的 `asset_id` 字段注释明确标记 "UUID(wire),URN 仅在 BFF/SDK 边界使用"

## References

- DataHub URN: https://datahubproject.io/docs/what/urn/
- RFC 8141 URN syntax: https://www.rfc-editor.org/rfc/rfc8141
