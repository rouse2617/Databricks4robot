# Design — CYB-3744 定时任务

## Overview

A **scheduled task rule** = 「从某个数据源按触发模式取 asset-id → 用指定流水线+资源池建批量」。规则存 DB、UI 管理、后端循环执行。数据源和触发窗口都 per-rule 可配。

```
[定时任务规则] --(触发模式:到点/手动)--> [数据源.Fetch(cursor,window) → asset-ids]
   --> [复用 backfill 建批量(template + target + scheduling)] --> 批量出现在「执行记录」
```

## Data model

New table `scheduled_tasks`:

| 列 | 说明 |
|----|------|
| `id`, `name`, `enabled` | 规则标识 + 开关 |
| `template_id`, `template_version` | 流水线 |
| `target_id`, `scheduling` (jsonb) | 资源池 + 调度(复用 resource_defaults.scheduling 形状) |
| `source_type` (text), `source_config` (jsonb) | 数据源(v1: `rest`) |
| `trigger_mode` (text) | `incremental` / `rolling` / `range` / `ids` |
| `trigger_config` (jsonb) | interval / lookback / from-to / ids,按模式取用 |
| `cursor` (text) | 增量模式的水位线(如 last_status_at 值),推进后回写 |
| `last_run_at`, `last_run_status`, `last_batch_id`, `last_error` | 运行观测 |
| `created_by`, timestamps | 审计 |

Migration: 手写 SQL 在 `backend/migrations/`(遵循 Atlas 流程),对应 GORM struct 只读参考。

## Source abstraction

```go
type AssetSource interface {
    // 取要导入的 asset-id + 推进后的游标;window 由触发模式给出
    Fetch(ctx context.Context, cursor string, window Window) (ids []string, nextCursor string, err error)
}
type Window struct { Start, End *time.Time; IDs []string } // range/ids 模式直接带
```

- **`restSource`(v1 唯一实现)**:按 `source_config` 描述的 REST-JSON 接口取数:
  - `base_url` + `auth{type: basic|bearer|header_key, secret_ref}`(凭证从 Secret Manager 解引用,**不落明文**)
  - `query`:静态过滤(如 `step_key`, `status`)+ 时间字段 `time_field` + 窗口模板 `{start}..{end}`(增量/滚动填游标或 lookback;range 填 from-to)
  - `paging{mode: page_size, page_param, size_param}` — v1 支持 page/size;翻页到 total
  - `id_path`:如 `data[].video_id`,从响应 JSON 抽 id(简单点表达式:`<列表字段>[].<id字段>`)
  - 复用现有 `grace.Client` 的 HTTP/超时/UA 处理(它已处理 Cloudflare UA、30s 超时)
- **非 REST 平台**:实现同一 `AssetSource` 接口写 adapter,注册进 source 工厂;v1 不预建。

Grace 现状 = 这个 restSource 的一份 config(`/grace/video_steps` + `step_key/status/last_status_at` + `data[].video_id` + page/size + basic auth),**逐字复刻现有 `services/grace-sync/main.go` 的查询**(过滤/分页/UA/超时/密码解析)。v1 优先跑通 Grace 这条,以验证 restSource 抽象;非 Grace 平台待有真实场景再加。

## Trigger modes

`resolveWindow(rule) → Window`:
- **incremental(默认)**:`window.Start = rule.cursor`(空则 now-lookback 兜底),`End=now`;Fetch 后 `cursor = nextCursor`(= 本批最大 `time_field`)。不漏不重复扫,漏跑一轮下轮自动补。
- **rolling**:`[now-lookback, now]`,不动 cursor。等价现有 grace-sync 默认行为。
- **range**:`[from, to]`,一次性(执行后可自动 disable 或标记完成)。
- **ids**:`window.IDs = 配置的 id 列表`,source 直接返回(REST 源可跳过查询)。

`incremental`/`rolling` 走定时(interval);`range`/`ids` 常配合「立即运行」。「立即运行」= 用规则的 source+流水线,但 window 由手动输入(range/ids)或"立即增量一次"。

## Scheduler loop + 单飞(single-runner)

一个后端 ticker(仿 `StartJobReconciler`),每个 tick:
1. 取 enabled 且到点(`now - last_run_at >= interval`,或手动触发标记)的规则。
2. **单飞认领**:`UPDATE scheduled_tasks SET last_run_at=now WHERE id=$1 AND (now - last_run_at >= interval) RETURNING ...`(条件更新做乐观锁),或 Postgres advisory lock `pg_try_advisory_lock(hash(id))`。**只有认领成功的副本执行**,避免多副本双触发。(镜像 dispatcher 的多副本协调。)
3. `resolveWindow` → `source.Fetch` → 得 asset-ids。
4. 空则记 last_run(+ 可选通知),跳过。
5. 非空 → 复用 backfill 建批量(template+version, target, scheduling, asset-ids, name=`<rule>-<ts>`)。
6. 回写 `cursor`(增量)、`last_run_status/last_batch_id/last_error`。

失败(source 报错/建批量失败):记 `last_error`,**不推进 cursor**(下轮重试同窗口),不 crash 循环。

频率 = **简单间隔**(`trigger_config.interval_seconds`,如 3600),不做 cron。

## 告警(复用现有飞书)

复用现有飞书接口(grace-sync 的 `notifyFeishu` / CYB-3071 已上线的飞书通知),不新建通道:
- **规则运行失败**:source 拉取或建批量失败 → 飞书通知(规则名 + 错误)+ 失败计数 metric。
- **规则陈旧(stuck)**:距上次成功运行 > 阈值(如 interval × N)→ 飞书告警(定时同步静默坏掉的兜底)。
- **空拉取(可开关)**:一直拉到 0 条 → 提示(默认关,规则上可开)。
- (可选)复用 CYB-3691 的 Prometheus gauge → GCP Cloud Monitoring,把上面失败/陈旧暴露成指标,和现有 backend-observability 看板/告警一致。

## API + UI

- **API**(admin,契约同步):`GET/POST/PUT/DELETE /scheduled-tasks`,`POST /scheduled-tasks/:id/pause|resume|run-now`。镜像 dispatcher_config handler 风格。
- **UI**:流水线页新增 tab key=`schedules` label「定时任务」(`PipelinePage.tsx` 的 tabs 数组)。列表 + 新建/编辑抽屉:复用模板选择器、target/调度选择器;数据源用一个 REST-源表单(base_url/auth-secret-ref/query/paging/id_path);触发模式单选 + 对应参数。含「立即运行」「启停」。
- 前端改动 → 部署后 **Chrome DevTools MCP 验收**。

## 迁移与退役(external grace-sync)

1. in-app 定时任务上线 + dev 验证(增量拉数 → 建批量 → 批量可见)。
2. prod 建一条等价规则(Grace REST 源,复刻现有 step_key/lookback),先与外部 grace-sync **并行灰度**,核对产出一致。
3. 确认 in-app 稳定后:**暂停** Cloud Scheduler(`grace-sync-prod`/`grace-sync-daily`)→ 观察一两周期 → **删除** Cloud Scheduler + Cloud Run Job(`grace-sync`/`grace-sync-prod`)。
4. 回滚:in-app 规则停用 + resume Cloud Scheduler 即恢复(pause 可逆;删除前保留 `services/grace-sync/deploy.sh` 可重建)。

## Security

- 数据源凭证**只存 Secret Manager 引用**(`secret_ref`),配置表/UI/日志绝不落明文。后端在 Fetch 时解引用。
- 定时任务的 CRUD 走现有 admin 鉴权边界(不新开无鉴权路径)。

## Alternatives considered

- **保留外部 Cloud Run Job,仅把它变可配** —— 仍是外部服务、仍只有你会管,不解决核心痛点。否决。
- **Cloud Scheduler → 打 DataBrew endpoint(薄触发)** —— 去掉 Cloud Run Job 但留 Cloud Scheduler(仍外部)。不如全 in-app。
- **每平台写死 adapter(不做通用 REST 源)** —— "加接口要写代码 + 部署",回到"只有 dev 能加",违背自助目标。故 v1 做通用 REST 源,adapter 仅作非 REST 逃生口。
- **做 asset×template 去重** —— 产品明确不做;水位线已避免重复扫,保持与现状一致。

## Test plan

- 单测:`restSource.Fetch`(分页、id_path 抽取、窗口模板、auth-secret 解引用 mock);`resolveWindow` 四模式;单飞认领(并发两副本只一个执行);cursor 推进 + 失败不推进。
- 集成:建规则 → 触发 → 建批量(mock source + 真 backfill 路径)。
- 部署验证:dev 建一条 Grace REST 规则,增量拉数建批量,批量在「执行记录」可见;前端 Chrome DevTools MCP 验收 tab。
