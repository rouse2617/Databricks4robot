# GoFrame 调研报告 — 可移植功能分析

## 概述

[GoFrame (gf)](https://github.com/gogf/gf) 是一个国产全栈 Go 框架，v2.10.0，13.1k stars。核心卖点是"开箱即用"——内置了从 HTTP 到 ORM 到链路追踪的全套组件。

我们当前用 Gin + 自建中间件。不建议换框架（迁移成本大、Gin 生态更广），但 GoFrame 有几个设计值得借鉴移植。

## 值得移植的功能（按优先级排序）

### P0: 全链路 OpenTelemetry 追踪

GoFrame 最强的设计：所有组件（HTTP、DB、Cache、Log）都内置 OpenTelemetry 支持，trace_id 自动透传到每一层。

**当前问题：** 我们的 request_id 只在中间件层打印，usecase/repo 层看不到。

**移植方案：**
- ✅ 已完成：`middleware.WithRequestID(ctx, rid)` 把 request_id 注入 context
- ✅ 已完成：`middleware.L(ctx)` 获取带 request_id 的 logger
- 待做：在 usecase/repo 层的关键操作加 `middleware.L(ctx).Info(...)` 日志
- 进阶：接入 OpenTelemetry SDK，用 trace_id + span_id 替代自定义 request_id，支持 Jaeger/Cloud Trace 可视化

**GoFrame 做法参考：**
```go
// GoFrame 的 ctx 自动带 trace 信息，所有组件自动读取
g.Log().Ctx(ctx).Info("doing something") // 自动带 trace_id
db.Ctx(ctx).Model("user").All()          // SQL 日志自动带 trace_id
```

### P1: 结构化日志 + 文件轮转

GoFrame 的 glog 支持：JSON/Text 格式切换、文件轮转（按大小/日期）、多输出（stdout + file）、日志级别动态调整。

**当前状态：** ✅ 已实现基础版（LOG_LEVEL/LOG_FORMAT/LOG_FILE 可配置）

**待完善：**
- 日志文件轮转（按天/按大小），可用 `lumberjack` 库
- 异步写入（高并发下减少 I/O 阻塞）

### P2: 输入校验框架

GoFrame 内置了声明式校验，支持 struct tag 定义规则：
```go
type CreateReq struct {
    Name string `v:"required|length:1,100"`
    Age  int    `v:"required|between:0,150"`
}
```

**当前做法：** Gin 的 `binding:"required"` + 手动校验

**移植方案：** 不需要换库，但可以借鉴思路——把 tag 校验规则（algo_registry、tag_registry）做成声明式的，减少 handler 层的手动校验代码。

### P3: 自动 API 文档生成

GoFrame 从代码注释自动生成 OpenAPI spec + Swagger UI。

**当前做法：** 手动维护 `api/openapi.yaml`

**移植方案：** 可以用 [swaggo/swag](https://github.com/swaggo/swag) 给 Gin handler 加注释，自动生成 OpenAPI。但当前手动维护也够用，优先级低。

### P4: 配置热加载

GoFrame 的 gcfg 支持配置文件变更自动重载，不需要重启服务。

**当前做法：** 环境变量 + .env 文件，改了要重启

**移植方案：** 可以用 `fsnotify` 监听 config 目录变化，热加载 algo_registry.yaml 和 tag_registry.yaml。对于环境变量类配置（端口、DB 连接）不需要热加载。

### P5: 错误码体系

GoFrame 有完整的错误码框架（gcode），支持错误码 + 错误消息 + 错误详情的分层结构。

**当前做法：** `httpresp.Error(c, status, code, message, details)` 已经够用

**移植方案：** 如果错误码越来越多，可以定义一个 `errors/codes.go` 集中管理所有错误码常量，避免散落在各个 handler 里。

## 不需要移植的功能

| GoFrame 功能 | 原因 |
|-------------|------|
| ORM (gdb) | 我们用 Bigtable，不需要 SQL ORM |
| 微服务框架 | 当前单进程架构，不需要 |
| 代码生成工具 (gf gen) | 项目结构已稳定 |
| Session/Cookie 管理 | API 服务不需要 |
| 模板引擎 | 前后端分离 |
| 定时任务 (gcron) | 用 Dagster 编排 |
| 国际化 (gi18n) | 暂不需要 |

## 实施建议

### 短期（本迭代可做）

1. 在 usecase 层关键操作加 `middleware.L(ctx)` 日志（2h）
2. 加日志文件轮转（用 lumberjack，1h）

### 中期（下个迭代）

3. 接入 OpenTelemetry SDK，替换自定义 request_id 为标准 trace_id（1d）
4. 集中管理错误码常量（2h）

### 长期（按需）

5. 配置热加载（algo/tag registry）
6. 自动 API 文档生成

## 结论

GoFrame 是个好框架，但对我们来说太重了。它的核心价值在于"全组件统一 OpenTelemetry 追踪"，这个思路我们可以在 Gin 上用轻量方式实现。其他功能要么已经有了（日志、错误码），要么不需要（ORM、微服务）。

不换框架，选择性移植 P0-P1 即可获得 80% 的收益。
