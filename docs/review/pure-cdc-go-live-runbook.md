# Pure CDC 上线 Runbook（生产基线）

> 目标读者：后端、数据工程、SRE、前端联调同学  
> 适用范围：Databricks4robot 从本地联调到预发/生产上线  
> 决策结论：**搜索与湖仓链路统一采用 Pure CDC 驱动，不保留定时全量 Reconciler 作为常驻路径**

---

## 1. 决策与范围

## 1.1 决策（必须执行）

- 主链路：`PostgreSQL WAL -> Debezium/Kafka -> backend CDC consumers -> Elasticsearch + Iceberg`
- 生产路径中，**不允许**依赖周期性“全量重建索引”作为常驻兜底机制。
- `reindex` 仅作为人工运维动作（故障恢复/历史补数），不是业务主通道。

## 1.2 非目标（本阶段不做）

- 不在本阶段引入第二条实时复制链路（例如双写 ES）。
- 不在本阶段引入复杂编排平台替代现有 CDC worker。
- 不在本阶段做 3.x 多模态检索升级（Lance/向量库等）。

---

## 2. 目标架构（上线口径）

## 2.1 数据流

1. 业务写入 PostgreSQL（资产、MCAP、交付、标签、算法投影等 current-state tables）。
2. PostgreSQL 以逻辑复制（WAL logical）产生变更事件。
3. Debezium 读取 WAL，写入 Kafka topic。
4. 后端 CDC Consumer 消费 Kafka：
   - 分支 A：构建/更新 ES 文档（资产搜索与筛选）
   - 分支 B：写入 Iceberg Bronze（用于审计、回放、离线分析）
5. 前端只读 API，API 由 PG + ES（按场景）提供查询结果。

## 2.2 一致性语义

- CDC 为**至少一次**投递语义，消费者必须做到幂等。
- ES 文档更新必须具备去重/覆盖逻辑（按主键与版本时间）。
- Iceberg 写入必须允许重复消费下的幂等落盘策略（基于事件 key/version）。

---

## 3. 上线前硬门槛（Go/No-Go）

以下 5 类全部通过，才允许上线：

## 3.1 功能门槛

- 登录后 `/dashboard`、`/assets`、`/assets/:id`、湖仓验证页可正常加载。
- `5173 -> /api` 与 `8080` 直连行为一致，禁止出现系统性 `502`。
- 新增资产后可在可接受 SLA 内被搜索命中。

## 3.2 一致性门槛

- 样本对账：`PG count`、`ES count`、`Iceberg count` 偏差在阈值内。
- 随机抽样字段对齐（如 `asset_id`、状态、时间戳、标签）通过率达到目标。

## 3.3 性能门槛

- 指定压测规模下，CDC 端到端延迟满足目标（例如 P95 < 30s，按业务调整）。
- Kafka consumer lag 在压测结束后可回落到稳定低水位。

## 3.4 稳定性门槛

- 连续运行窗口（建议 12h~24h）无关键链路中断。
- 无持续增长的死信/重试积压。

## 3.5 可运维门槛

- 告警规则完整并经过演练（接口 5xx、consumer lag、ES bulk fail、Iceberg 写入失败）。
- 回滚与恢复 runbook 已演练至少 1 次。

---

## 4. 配置基线清单（必须对齐）

## 4.1 PostgreSQL / Debezium

- `wal_level=logical` 已启用。
- 复制槽、publication 正常创建并可持续推进。
- Debezium connector 使用明确数据库编码（`UTF8`）与目标库名。
- Kafka topic 命名稳定，不允许环境间随意漂移。

## 4.2 Backend

- `CDC_ENABLED=true`（按环境变量口径）。
- 生产配置中，关闭本地临时兜底任务（如定时全量 ES 重建逻辑）。
- consumer group id 固定，避免滚动发布时重复消费风暴。

## 4.3 Frontend / 网关

- 前端反向代理（Vite/Nginx）上游必须稳定指向后端服务。
- SPA 路由需支持直链刷新（`try_files` 回退策略正确）。
- 登录 token 传递方式与后端鉴权口径一致（`X-Grace-Token` / `Authorization`）。

---

## 5. 代码与配置迭代计划（建议两周）

## Iteration 1：链路可用（P0）

- 修复 `5173 /api` 代理异常（当前重点：消除 502）。
- 修复前端直链 404（如 `/assets/:id`）。
- 统一本地/预发 compose 与环境变量模板，避免“本地好、预发坏”。

**DoD**

- Dashboard 无“数据加载失败”阻塞；
- 主路径接口 2xx/4xx 预期稳定；
- 前端路由刷新通过。

## Iteration 2：Pure CDC 收口（P0）

- 删除/禁用常驻 Reconciler（仅保留人工 reindex 命令）。
- 校准 CDC consumer 的幂等、重试、错误分类日志。
- 明确 topic -> consumer -> sink 的映射文档。

**DoD**

- 不依赖周期全量任务也能稳定投影到 ES；
- 新写入数据在 SLA 内可检索；
- consumer lag 与错误率可观测。

## Iteration 3：压测与对账（P1）

- 批量写入资产（建议分层：1k/10k/50k）。
- 观测 PG 写入吞吐、Kafka lag、ES bulk、Iceberg 增量。
- 出具报告：吞吐、P95 延迟、错误率、一致性偏差。

**DoD**

- 报告可复现；
- 数据偏差可解释；
- 无不可接受的数据丢失风险。

## Iteration 4：上线演练（P1）

- 进行预发灰度发布（小流量 + 监控盯盘）。
- 模拟故障：后端短暂不可用、Kafka 重平衡、ES 短抖动。
- 按 runbook 演练恢复并记录 MTTR。

**DoD**

- 全链路恢复流程可执行；
- 告警有效，值班同学可独立处理。

---

## 6. 测试与验收设计（详细）

## 6.1 单元测试

- Debezium 事件解析：payload 包裹、主键提取、空字段健壮性。
- CDC 事件映射：`asset_id`、`mcap_file_id`、表名路由。
- 幂等行为：重复事件输入后 ES/Iceberg 结果不异常膨胀。

## 6.2 集成测试

- 启动最小链路：PG + Debezium + Kafka + backend consumers + ES。
- 写入一组资产变更（insert/update/delete）。
- 断言 ES 文档状态与 PG current-state 一致。

## 6.3 端到端测试

- 浏览器场景：登录 -> dashboard -> 资产列表 -> 资产详情 -> 搜索命中。
- 验证控制台/网络层无系统性 5xx。
- 至少保留一条“新建资产后在 UI 可查到”的自动化用例。

## 6.4 压测（推荐脚本化）

- 输入模型：
  - 批量创建资产（带标签、状态变化）
  - 批量写入 MCAP 元数据
  - 混合更新（小比例 delete/update）
- 指标采集：
  - API 吞吐（RPS）与错误率
  - CDC 端到端延迟（写 PG 到 ES 可查）
  - Kafka lag 曲线
  - ES bulk reject/失败数
  - Iceberg 写入速率与失败数

## 6.5 对账（必须）

- 计数对账：按业务主键统计 PG/ES/Iceberg。
- 抽样对账：随机抽样 N 条对比核心字段。
- 差异分层：可接受（延迟内）/需补偿（超时未到达）/缺陷（逻辑错误）。

---

## 7. 可观测性与告警（上线必备）

## 7.1 指标分层

- **入口层**：API 4xx/5xx、请求时延、关键端点成功率。
- **CDC 层**：消费速率、消费失败率、重试次数、lag。
- **ES 层**：bulk 成功率、reject、写入延迟。
- **Iceberg 层**：提交成功率、失败次数、批处理时延。

## 7.2 告警建议

- `API_5XX_Rate_High`：持续窗口内 5xx 比例超阈值。
- `CDC_Lag_High`：consumer lag 超阈值并持续增长。
- `CDC_Consumer_Error_Burst`：单位时间异常暴增。
- `ES_Bulk_Failure`：bulk 失败/拒绝超过阈值。
- `Iceberg_Write_Failure`：连续写入失败。

## 7.3 日志规范

- 统一结构化日志字段：`trace_id`, `table`, `topic`, `partition`, `offset`, `asset_id`。
- Error 日志必须可定位到“哪条事件、在哪个 sink 失败”。
- 禁止长期保留高噪声 debug 日志，避免掩盖真实告警。

---

## 8. 发布、灰度与回滚

## 8.1 发布策略

- 先预发全链路验证，再生产灰度。
- 灰度阶段只放一部分读流量，写流量保持真实但可控。
- 每一步发布都要有“可观测通过条件”。

## 8.2 回滚触发条件

- 连续高比例 5xx 且无法在短窗口内恢复。
- CDC lag 持续恶化并影响业务可见性。
- ES/Iceberg 数据偏差超过业务可接受阈值。

## 8.3 回滚动作（最小化）

1. 回滚后端版本到上一个稳定版本。
2. 保留 CDC 原始事件，不做破坏性清理。
3. 使用人工 `reindex` 或补偿作业恢复投影一致性。
4. 输出事故复盘：触发原因、影响范围、修复项和截止时间。

---

## 9. 故障排障 Runbook（值班向）

## 9.1 现象：Dashboard 显示“数据加载失败”

- 检查 `5173/api` 返回码是否 502/504。
- 对照 `8080` 直连是否正常：
  - 若 8080 正常、5173 异常：优先排查前端代理/网关上游。
  - 若 8080 也异常：排查 backend 与依赖服务。

## 9.2 现象：新资产写入后搜不到

- 检查 Kafka lag 是否堆积。
- 检查 consumer error 日志是否集中在特定表/字段。
- 核对 ES 是否有写入失败（bulk reject/mapping error）。

## 9.3 现象：Iceberg 数据延迟或缺失

- 检查 Bronze consumer 健康与提交失败日志。
- 检查对象存储/目录权限与 catalog 可用性。
- 必要时执行补偿回放（按时间窗或 offset 区间）。

## 9.4 现象：Debezium 连接器异常

- 检查 Postgres 逻辑复制配置、复制槽状态、连接器错误日志。
- 检查 connector 配置变更（库名/编码/topic 路由）是否漂移。
- 恢复后验证 offset 连续性与重复消费影响。

---

## 10. 安全与配置治理

- 不在仓库存放生产密钥；使用 Secret 管理系统注入。
- 认证 token、数据库密码、ES 凭据执行最小权限与轮换策略。
- 对外接口和内部 CDC 日志避免泄露敏感字段。

---

## 11. 团队执行清单（可直接打卡）

## 11.1 每日检查

- API 5xx 是否异常
- CDC lag 是否回落
- ES 写入失败是否归零
- Iceberg 提交失败是否归零

## 11.2 每周检查

- 样本对账报告（PG/ES/Iceberg）
- 压测回归（小规模）
- 告警误报与漏报复盘

## 11.3 上线前最终检查

- Pure CDC 路径已启用，常驻 Reconciler 已关闭
- 关键链路告警已打开并通过演练
- 回滚方案演练完成并可在值班文档中检索
- 产品/研发/SRE 三方签字（或等价发布审批）

---

## 12. 建议的仓库内落地动作（与本 Runbook 配套）

- 在后端配置中新增显式开关：`SEARCH_RECONCILER_ENABLED=false`（生产固定为 false）。
- 在部署模板中区分 `local` 与 `production` 配置，避免测试兜底误入生产。
- 增加 `make cdc-smoke` / `make cdc-loadtest` / `make cdc-reconcile-report` 等标准化命令。
- 在 CI 增加 CDC 冒烟用例：写入 -> CDC -> ES 可查。

---

## 13. 结论

你选择的“Pure CDC 驱动”是可上线方案，前提是：

- 主链路稳定（不靠常驻全量补偿）；
- 指标与告警完备；
- 压测与对账可复现；
- 回滚可执行。

这份文档即为上线执行蓝本。后续所有迭代，请以“减少隐藏兜底、提升链路可观测与可恢复”为第一原则。

