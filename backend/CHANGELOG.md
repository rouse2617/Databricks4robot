# Changelog

本文件记录后端服务的所有重要变更。格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [Unreleased]

### Added
- **Postgres 迁移** — 默认存储后端从 Bigtable 切换到 PostgreSQL
  - Docker Compose 新增 `postgres:16-alpine` 服务，端口 5432，pgdata 持久卷
  - `001_init.sql` — 6 张表（assets, mcap_files, deliveries, delivery_items, asset_algo_events, idempotency_keys）、8 索引、3 外键、pgcrypto 扩展
  - `002_seed.sql` — 种子数据（3 McapFiles, 5 Assets, 2 Deliveries, 4 DeliveryItems, 3 AlgoEvents）
  - `003_mock_10k.sql` — 万级 mock 数据生成脚本（10,000 assets, 50 mcap_files, 20 deliveries, 2000 algo_events）
  - `segment_locator` 字段 — Asset 模型新增 SHA-1(mcap_file_id + start_ns + end_ns) 稳定标识
  - `ComputeSegmentLocator()` 纯函数 + `pgregory.net/rapid` 属性测试
  - Postgres repos 补全 `end_timestamp_ns`、`segment_locator`、`cf_files` 读写
  - `STORAGE_BACKEND` 默认值改为 `postgres`
  - `.env` / `.env.example` 新增 DB_HOST/PORT/USER/PASSWORD/NAME

### Changed
- **STORAGE_BACKEND 默认值** — 从 `bigtable` 改为 `postgres`

### Fixed
- **虚拟字段过滤失败** — `ValidateFilters` 对 `algo_status`、`has:delivery` 等虚拟字段调用 `ResolveField` 导致 "field not allowed" 错误，现在跳过虚拟字段的 ResolveField 校验
- **has:delivery SQL 路径错误** — `delivery_count` 在 Postgres 中存储于 `cf_meta` JSONB 内，`has:delivery` 虚拟字段的 SQL 从 `delivery_count > 0` 修正为 `COALESCE((cf_meta->>'delivery_count')::int, 0) > 0`

### Documentation
- `backend/README.md` — 新增 Postgres 本地开发指南、docker-compose 用法、数据重置方法
- `docs/sql.md` — 标注 Postgres 为主存储后端

### Added
- **MCAP 文件列表接口** — `GET /api/v1/mcap-files` 支持分页查询
  - `McapFileRepository.List()` 接口定义
  - Bigtable 实现：PrefixRange 扫描 + 早期终止 + 10s 超时
  - PostgreSQL 实现：COUNT + LIMIT/OFFSET 分页
  - Handler 替换占位符，支持 `page` / `page_size` 参数
  - 前端 McapFilesPage 完整表格（ID、GCS Path、大小、状态、Channels、Chunks、Owner、更新时间）
  - Dashboard MCAP 文件 KPI 卡片显示真实数量
  - OpenAPI 规范更新
  - API 使用指南更新
- **Bigtable 全端点对等** — 所有 API 端点在 Bigtable 模式下完全可用，与 PostgreSQL 功能对等
  - `CheckAndMutateRow` 支持（btTable 接口扩展）
  - `MergeCfAlgo` 乐观锁实现（算法字段原子合并 + 版本控制）
  - `AlgoEventRepo` 新增（Insert + ListByAsset，行键 reverse_timestamp 排序）
  - `Asset.Set()` / `rowToAsset()` 补全 cf:files 和 LifecycleMeta 字段
  - `ListWithFilters` 全表扫描 + 内存过滤/排序/分页
  - `ListDeliveries` 端点实现（DeliveryRepository 接口扩展）
  - `main.go` Bigtable 分支接线 AlgoHandler
- **算法依赖链自动 Unblock** — action_annotation 依赖全部完成后自动从 blocked → pending
- **算法 finish 幂等** — 相同 run_id 重复 finish 返回 200
- **统一错误码** — `httpresp/codes.go` 集中定义 19 个错误码常量，消除硬编码字符串
- **声明式校验** — `internal/validate/` 包，自定义校验规则（valid_status、valid_algo_key），label tag 友好错误消息
- **配置热加载** — fsnotify 监听 algo_registry.yaml / tag_registry.yaml，100ms 防抖自动重载
- **Swagger 自动文档** — swaggo 注释 + `/swagger/*any` 路由，`make swagger` 生成
- **结构化请求日志** — `middleware/logger.go`，slog 输出 method/path/status/latency/request_id/client_ip，healthz 跳过
- **可配置日志** — LOG_LEVEL / LOG_FORMAT / LOG_FILE 环境变量，支持 text/json 格式 + 文件输出
- **请求链路追踪** — request_id 透传到 context.Context，`middleware.L(ctx)` 获取带 trace 的 logger
- **连接池预热** — `Client.Warmup()` 启动时对 8 个 Bigtable 表做空读，消除首次请求冷启动
- **gRPC 连接池** — 显式设置 `WithGRPCConnectionPool(4)`
- **URI 长度保护** — `RequestGuard` 中间件，超过 2048 字符返回 414
- **限流中间件** — per-IP token bucket，RATE_LIMIT_RPS / RATE_LIMIT_BURST 配置，默认关闭
- **熔断中间件** — 滑动窗口 circuit breaker，CB_ENABLED / CB_THRESHOLD / CB_COOLDOWN_SEC 配置，默认关闭
- **端到端测试** — `e2e_test.go` 使用 httptest + fakeTable 覆盖 Asset CRUD、算法生命周期、ListDeliveries
- **属性测试** — 7 个正确性属性（MergeCfAlgo 往返/锁拒绝、AlgoEvent 往返/过滤、Asset 往返、ListWithFilters 过滤/排序）
- **集成测试脚本** — `scripts/test_bigtable_e2e.sh` 连接真实 Bigtable 验证
- **极端 Case 测试** — `test_edge_cases.sh` + `test_edge_cases_2.sh`，85 个边界用例
- **千级测试** — `scripts/megaTest/`，20 维度 1000 用例，生成 Markdown 报告
- **压测工具** — `scripts/bench/main.go`，支持 create/get/algo/all 场景，可配置并发和请求数

### Changed
- **列族常量修正** — CFMeta/CFAlgo/CFTag/CFProcess/CFFiles 从 `cf:meta` 改为 `meta`，与 bootstrap_bigtable.sh 一致
- **UnpackInt64 兼容** — 兼容旧数据的字符串格式（非 8 字节时 fallback 到 strconv.ParseInt）
- **Handler 构造函数** — `assetH.New()` 新增 `DeliveryRepository` 参数
- **AlgoRegistry / TagRegistry** — 加 sync.RWMutex 线程安全 + Reload() 方法

### Fixed
- **超长 URL 500** — 5000+ 字符 URL 不再触发 panic，改为 414 URI_TOO_LONG

### Documentation
- `backend/README.md` — 完整重写，含架构、API 表格、开发指南、测试指南、性能数据
- `docs/api-guide.md` — 全部端点 curl 示例、错误码参考、工作流示例
- `docs/goframe-research.md` — GoFrame 调研报告，可移植功能分析
- `docs/go-web-frameworks-research.md` — 23 个 Go Web 框架全景调研
- `CLAUDE.md` — 功能开发后必须更新的文件清单 + Definition of Done checklist
- `README.md` — Current Status 更新
