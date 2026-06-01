# API Overview

Cyber Databrew 的 REST API 文档以仓库根目录的 `api/openapi.yaml` 为真相源。完整接口、参数、响应结构和可在线试调的请求面板都放在交互式 API Reference 中维护，避免在文档站点里手写一份会漂移的端点表。

[打开交互式 API Reference](/doc/api/reference)

## Base URL

本地开发默认使用：

```text
http://localhost:8080
```

Dev / Cloud Run 环境请使用团队提供的当前服务地址。仓库里的验证脚本会通过 `scripts/dev-backend-env.sh` 解析 dev 后端入口，避免误用过期网关域名。

## Authentication

API 支持两类认证入口：

| 场景 | 方式 | 说明 |
| --- | --- | --- |
| SDK、脚本、服务端调用 | `X-Databrew-Token` header | Python SDK 和自动化脚本使用。 |
| 浏览器端登录 | `POST /api/v1/auth/email-login` | 成功后浏览器获得 `databrew_session` JWT cookie，前端后续请求随 cookie 发送。 |

程序化调用示例：

```bash
curl -H "X-Databrew-Token: g-xxx" \
  http://localhost:8080/api/v1/sdk-config
```

浏览器登录请求示例：

```bash
curl -X POST http://localhost:8080/api/v1/auth/email-login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}' \
  -c cookies.txt
```

## Try Requests In The Browser

使用 [Interactive API Reference](/doc/api/reference) 可以直接浏览 OpenAPI 生成的接口说明，并在页面中填写参数、认证信息后发送请求。接口列表、schema、错误响应与 `api/openapi.yaml` 同步，不再由本页维护。

## Error Responses

JSON API 错误统一返回以下字段：

```json
{
  "code": "INVALID_ARGUMENT",
  "message": "invalid request body",
  "request_id": "req-xxx",
  "details": {"field": "asset_id"}
}
```

Python SDK 会把这些字段映射到异常对象：`e.code`、`e.message`、`e.request_id`、`e.details`，并额外提供 `e.http_status`。排查线上问题时优先保留 `request_id`。

## Maintaining API Docs

新增或修改 HTTP API 时，请在同一个 PR 中更新：

| Artifact | Purpose |
| --- | --- |
| `api/openapi.yaml` | API reference 的真相源。 |
| `docs/review/api-guide.md` | 仅保留面向评审的补充说明、场景限制和验证备注。 |
| SDK / Frontend API clients | 当公开 REST surface 变化时同步调用方。 |
| Smoke scripts | 覆盖新接口 happy path 和至少一个错误路径。 |
