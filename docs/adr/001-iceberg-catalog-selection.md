# ADR-001: Iceberg REST Catalog 选型 — Polaris vs Lakekeeper

- **状态**: 已决定
- **日期**: 2025-01-15
- **决策者**: 平台团队
- **设计依据**: data-platform-design.md §5.6.4, §5.11, §6.2

## 背景

平台 2.0 引入 Apache Iceberg 作为湖仓表格式（§5.6.4），需要一个 REST Catalog 服务管理 Iceberg 表的 metadata pointer、snapshot 推进和跨引擎权限。

设计文档 §5.11.1 明确候选为 **Polaris**（首选）或 **Lakekeeper**（轻量替代），核心诉求：

1. 跨云零绑定 — 不依赖 Glue / Unity Catalog / BigLake 等托管 Catalog
2. 标准 Iceberg REST Catalog 协议 — PyIceberg、Trino、Spark 均可直连
3. 运维简单 — 单容器部署，本地 docker-compose + 生产 K8s Deployment

## 候选方案对比

| 维度 | Apache Polaris | Lakekeeper |
|------|---------------|------------|
| 仓库 | [apache/polaris](https://github.com/apache/polaris) | [lakekeeper/lakekeeper](https://github.com/lakekeeper/lakekeeper) |
| 语言 | Java (Dropwizard) | Rust |
| 协议 | Iceberg REST Catalog spec（官方参考实现） | Iceberg REST Catalog spec |
| 成熟度 | Apache 顶级项目（2024 年底毕业），Snowflake 捐赠 | 社区项目，活跃开发中 |
| 存储后端 | 内置 EclipseLink（PG / MySQL / H2） | SQLite / PostgreSQL |
| 权限模型 | 内置 RBAC（catalog / namespace / table 粒度） | 基础 ACL |
| 容器镜像 | `apache/polaris:latest`（~300 MB，JVM） | `lakekeeper/lakekeeper:latest`（~50 MB，Rust 静态链接） |
| 资源占用 | JVM 堆 512 MB–1 GB | ~50 MB RSS |
| 多引擎支持 | Trino / Spark / PyIceberg / Flink 均已验证 | Trino / PyIceberg 已验证；Spark 社区报告可用 |
| 生产案例 | Snowflake 内部大规模使用；开源后多家公司采用 | 早期采用者，生产案例较少 |
| 本地开发 | `apache/iceberg-rest-fixture` 轻量镜像可替代 | 原生轻量，适合本地 |

## 决策

**选择 Apache Polaris 作为生产 Iceberg REST Catalog。**

本地开发阶段继续使用 `apache/iceberg-rest-fixture`（轻量 REST Catalog 参考实现），与 Polaris 协议兼容，无需本地跑完整 JVM 服务。

生产部署时切换到 Polaris 完整镜像，配置 PostgreSQL 作为 metadata 后端。

## 理由

1. **官方参考实现**：Polaris 是 Apache Iceberg 社区的官方 REST Catalog 参考实现，协议兼容性最强，升级风险最低。

2. **成熟度与生态**：Apache 顶级项目，Snowflake 捐赠并持续投入，社区活跃度和生产验证远超 Lakekeeper。

3. **内置 RBAC**：Polaris 自带 catalog / namespace / table 粒度的权限控制，未来多团队协作时无需额外开发。

4. **跨引擎验证**：Trino、Spark、PyIceberg、Flink 均有官方集成测试，减少对接风险。

5. **运维可控**：虽然 JVM 资源占用高于 Rust，但生产环境 512 MB 堆完全可接受；K8s Deployment 标准部署。

## 不选 Lakekeeper 的原因

- 社区项目，尚未进入 Apache 孵化，长期维护风险较高
- 生产案例少，遇到边界问题时社区支持有限
- 权限模型较基础，未来扩展需自行开发
- 资源优势（50 MB vs 512 MB）在 K8s 环境下不构成决定性因素

## 后续行动

1. 本地 docker-compose 继续使用 `apache/iceberg-rest-fixture`（已配置）
2. 生产 K8s 部署时使用 `apache/polaris:latest` + PostgreSQL metadata backend
3. 如 Polaris 在 2026 中前仍未满足需求（如启动时间过长、资源占用不可接受），可评估切换到 Lakekeeper（协议兼容，业务代码零改动）

## 升级触发条件

参见 data-platform-design.md §5.11.3：当前选型不锁死，协议层兼容保证切换成本极低。
