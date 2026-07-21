## ADDED Requirements

### Requirement: 定时任务规则(自助配置的自动下发)

The system SHALL let a user define, in the UI (a 「定时任务」 tab under the Pipeline page), a scheduled-task rule specifying a pipeline template, a resource pool (execution target + scheduling), a data source, and a trigger mode. The backend SHALL execute enabled rules on a schedule, fetch asset-ids from the source, and create a batch run using the rule's template and resource pool. Batches so created SHALL appear in the existing 执行记录 → 批量任务 view. Rules SHALL be creatable, editable, enable/disable-able, and manually runnable by any authorized user without touching external infrastructure.

#### Scenario: create and run a scheduled rule from the UI

- **GIVEN** a user opens the 「定时任务」 tab and creates a rule (pipeline template T, target/pool P, a REST data source S, trigger mode incremental, interval 1h)
- **WHEN** the rule is enabled and its interval elapses
- **THEN** the backend SHALL fetch asset-ids from S and create a batch for template T on pool P over those ids
- **AND** the batch SHALL be listed under 执行记录 → 批量任务

#### Scenario: manual run-now

- **GIVEN** an enabled or paused rule
- **WHEN** the user clicks 立即运行 (optionally supplying a date range or explicit ids)
- **THEN** the backend SHALL run the rule once immediately with that window and create a batch

### Requirement: 可配置数据源(通用 REST 源,凭证仅引用密钥)

A scheduled-task rule SHALL carry a data source described by a `source_type` and `source_config`. The system SHALL provide a generic REST-JSON source such that different endpoints and different REST APIs are expressible purely as configuration (base URL, auth, query filters, pagination, id extraction path) without code changes. Source auth credentials SHALL be referenced by a secret-manager key only and SHALL NOT be stored or displayed in plaintext. Non-REST platforms MAY be added later as coded adapters behind the same `AssetSource` interface.

#### Scenario: different endpoint / different API is config-only

- **GIVEN** an existing REST source rule pointing at endpoint/API A
- **WHEN** a user creates another rule pointing at a different endpoint or a different REST API B (different base URL / filters / id path / auth)
- **THEN** B SHALL be usable by supplying configuration only, with no code change or deploy

#### Scenario: credentials are not stored in plaintext

- **GIVEN** a REST source needing authentication
- **WHEN** the rule is saved
- **THEN** the credential SHALL be persisted only as a secret-manager reference, never as plaintext in the config table, UI, or logs

### Requirement: 触发模式与增量水位线

A rule SHALL support four trigger modes: incremental (watermark cursor on a time field), rolling window (`[now-N, now]`), fixed range (`[from, to]`), and explicit ids. In incremental mode the system SHALL fetch only records newer than the stored cursor and advance the cursor after a successful fetch; a missed cycle SHALL be covered on the next run without gaps or re-scanning. The system SHALL NOT perform asset×template de-duplication (out of scope by product decision).

#### Scenario: incremental advances the watermark and self-heals

- **GIVEN** a rule in incremental mode with cursor C
- **WHEN** it runs and fetches records with time-field values up to C'
- **THEN** it SHALL create a batch for those ids and set the cursor to C'
- **AND** if a scheduled cycle is missed, the next run SHALL still start from C (no gap)

#### Scenario: a source or batch failure does not lose position

- **GIVEN** an incremental rule
- **WHEN** the source fetch or batch creation fails
- **THEN** the cursor SHALL NOT advance and the failure SHALL be recorded, so the next run retries the same window

### Requirement: 多副本单飞

When the backend runs with more than one replica, a due rule SHALL be executed by at most one replica per cycle (via a claim / lock), so scheduled runs are not double-triggered.

#### Scenario: two replicas, one execution

- **GIVEN** two backend replicas and one due rule
- **WHEN** both replicas' scheduler loops tick at the same time
- **THEN** exactly one replica SHALL execute the rule and create a batch; the other SHALL skip
