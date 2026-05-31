# Changelog

## v0.2.0 (2026-05)

### 新增功能
- 资产管理：支持血缘（Lineage）和谱系（Provenance）追踪
- 交付管理：两步提交模式（草稿 → 提交）
- Lakehouse 湖仓集成：报表、状态监控、同步进度
- 保存查询管理：创建、更新、删除保存的查询
- 算法运行（Algo Runs）：全生命周期管理

### SDK 更新
- 新增 `eval_metrics`、`actions`、`workflows`、`pipeline_components` 管理器
- 新增 SSE 事件流支持
- 改进配置管理：支持远端配置热更新
- 提升错误处理的精确度

### 修复
- 修复分页查询的边界情况
- 提升 MCAP 下载的稳定性

---

## v0.1.0 (2026-04)

### 初始版本
- 基础资产管理（CRUD）
- MCAP 文件存储
- 交付管理基础功能
- 客户管理
- Python SDK 发布
