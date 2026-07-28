# Decisions — CYB-3744 定时任务

## 2026-07-21 — 数据源:通用 REST 源(配置驱动),非 adapter 堆

- **Context**: 需求要"今天 A 接口、明天 B 接口、不同端点也可能不同 API",且要**别人也能自助加**(核心痛点:只有一个人会管外部服务)。
- **Decision**: v1 做一个**通用 REST-JSON 源**,接口由 `source_config` 声明(base_url/auth/query/paging/id_path)。不同端点/不同 API = 不同配置,UI 里加,无需写代码。底层留 `AssetSource` 接口,非 REST 平台以后写 adapter。
- **Alternatives**: 每平台写死 adapter → "加接口=写代码+部署",回到只有 dev 能加,违背自助;保留外部 Cloud Run Job 仅变可配 → 仍外部、仍单人管。
- **Rationale**: 配置化 REST 直接消灭"只有你会弄";adapter 逃生口保留扩展性但不预建(遵循"通用抽象等第二个真实场景",而这里 REST 就是已知的多场景)。

## 2026-07-21 — 不做去重(产品决定)

- **Decision**: 不做 asset×template 去重,与现有 grace-sync 一致。增量水位线天然避免重复扫;不加"已跑过跳过"。
- **Rationale**: 用户明确"不用去重";水位线足够;少一层复杂度。

## 2026-07-21 — 触发窗口做成 per-rule 模式(增量/滚动/范围/ids)

- **Decision**: 不全局定一个窗口;每条规则选模式。增量(水位线)为推荐默认;范围/ids 供补跑(手动「立即运行」)。复刻 grace-sync 已有的 `--date/--from-to/--ids/lookback`。
- **Rationale**: "每个需求不一样" —— 模式化覆盖持续同步 + 补跑两类场景。

## 2026-07-21 — 复用现有零件,几乎不加基础设施

- **Decision**: 复用 `grace.Client`(HTTP/超时/UA)、`grace.Syncer` 循环接线、backfill 建批量、`StartJobReconciler` ticker 模式、`dispatcher_configs` 的(表+admin API+UI)模式。只新增:`scheduled_tasks` 表、REST 源、调度 usecase、CRUD API、前端 tab。
- **Rationale**: 后端本就跑着 grace 同步循环 + 批量创建 + 配置面;外部 Cloud Run Job 是重复能力。折进来成本低。

## 2026-07-21 — 单飞(多副本)

- **Decision**: 调度循环用条件 UPDATE 认领(或 pg advisory lock)保证一条到点规则每周期只被一个副本执行。
- **Rationale**: 后端多副本;否则重复建批量(正是之前批量堆积的一类隐患)。镜像 dispatcher 的多副本协调。

## 2026-07-21 — 凭证只用 Secret Manager 引用(off-limits 纪律)

- **Decision**: 数据源鉴权凭证只存 `secret_ref`,配置表/UI/日志不落明文;后端 Fetch 时解引用。
- **Rationale**: 安全红线,不破例(见 AI-RULES:不提交凭证)。

## 2026-07-21 — 退役外部 grace-sync 走灰度 + 可回滚

- **Decision**: in-app 上线 → prod 建等价规则与外部 grace-sync **并行核对** → pause Cloud Scheduler 观察 → 再 delete Cloud Scheduler + Cloud Run Job。保留 `services/grace-sync/deploy.sh` 以便回滚重建。
- **Rationale**: prod 数据同步不能断档;可逆优先(pause 先于 delete)。

## 2026-07-21 — OpenSpec checkpoint 批准 + 飞书就地 copy 不预抽公共包

- 用户「你来安排 OpenSpec OK,开工」→ 进入编码。
- 飞书通知:先从 `services/grace-sync/main.go:notifyFeishu` copy 到 backend 作为规则告警的第一版实现;等 backend 里出现第二个飞书消费者再抽共享 `notifier` 包(遵循"通用抽象等第二个真实场景")。

## 2026-07-21 — 频率用简单间隔(不做 cron)

- **Decision**: 每条规则的频率 = "每 N 分钟/小时"(简单间隔),不支持 cron 表达式。
- **Rationale**: 用户选 A;实现简单;够用。以后要"每天 02:00"这种具体时刻再评估加 cron。

## 2026-07-21 — 告警复用现有飞书接口

- **Decision**: 告警走**现有飞书接口**(grace-sync 的 `notifyFeishu` / CYB-3071 已上线的飞书通知),不新建告警通道。范围:**规则运行失败** + **规则陈旧(连续 N 周期/超阈值没成功)**;空拉取通知设可开关。
- **Rationale**: 用户"复用现场飞书";定时同步静默坏掉最该告警;不加新基础设施。

## 2026-07-21 — 先做通 Grace,参考 grace-sync

- **Decision**: v1 先把 **Grace 这条数据源端到端做通**,直接**参考 `services/grace-sync/main.go`** 复刻查询(`/grace/video_steps` + `step_key/status=success/last_status_at` + `data[].video_id` + page/size + basic auth)。通用 REST 源仍是架构底座(Grace 就是它的第一份配置),但先以 Grace 验证跑通,不预先堆其他平台。
- **Rationale**: 用户"先实现和 grace 的";最小可用先落地,generic 能力由 Grace 这份实例验证。

## 2026-07-21 — 范围与后续

- 本 change = 需求 1(定时任务)。需求 2、3(如有)另开,不塞进本 change(一功能一 PR / 一 change 的连贯性)。
- 非 REST 平台、去重、更多分页/鉴权类型 —— 出现真实需求再扩,不预建。
