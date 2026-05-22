# 资产多版本设计

| 字段 | 值 |
|------|----|
| 状态 | Active（P1）|
| 关联 | `../README.md` §5 / `../schema.md` §3 / `algo-runs.md` |
| 决策来源 | rev.5 N1 严格线性 / rev.10 OPT1 / rev.11 AR1 / rev.12 C2/C3 |

---

## 0. 本文档边界（先看这个，避免误读）

**本文档讲：** 「同一逻辑物的多个物理版本」如何建模存储。

> 例：clipA 重切过 3 次 mp4 → 在 PG `assets` 表里有 v1 / v2 / v3 三行，每行一个独立 `asset_id`，但共享同一 `logical_asset_id`。

**本文档不讲：** 项目里另外两种「版本」概念，各自有独立模型，去对应文档：

| 听到的「版本」 | 实际指什么 | 字段在哪 | 文档 |
|---|---|---|---|
| **资产版本** ✅ 本文 | 资产行的内容版本（重切了 mp4 / 改了时间窗 / 换了主标签）| `assets.revision` + `logical_asset_id` | 本文档 |
| **算法版本** | 算法 SDK 的发版（`hand_track@1.0` / `hand_track@2.0`）| `algo_runs.algo_version` | [`algo-runs.md`](algo-runs.md) |
| **标签源版本** | 给资产打标签的来源系统的版本（`compliance_check@1.2` 规则版本）| `asset_tags.source_version` | [`asset-tagging.md`](asset-tagging.md) |

**三者会联动**（算法 v2.0 跑出来 → 触发资产升 v2），但建模上**完全独立**，不要混。详见 §6。

---

## 1. 业务问题（Why）

数据资产会被反复迭代：
- 算法升级（hand_track v1 → v2 → v3）重跑 → 同一段时间窗切出更精准的 clip
- 人工返工修正错误标注
- 时间窗微调（精度提升）

**没有版本模型的痛点：**
- 客户在 v1 时下载了文件，平台升级到 v2 后偷偷覆盖 → **「客户文件静默变化」事故**（合规雷区）
- 客户问「我之前买的 clipA 现在是哪版？」答不出
- 「这个 clip 经历了 v1 → v2 → v3，每次变了什么？」无法回溯
- 多个算法版本结果并存时怎么标识「同一物」

→ **必须**对「同一逻辑资产」的多版本演进显式建模。

---

## 2. 设计原则（How）

### 2.1 用 4 个字段表达「同一逻辑物的多个版本」

类比身份证。一个人换了三次身份证：每张卡的**序列号不同**，但**身份证号一样**，每张卡上印着「第 N 次换发」，钱包里只有一张是**当前有效**的。

资产也一样，4 个字段分工：

| 字段 | 干啥 | 身份证类比 |
|------|------|------|
| `asset_id` | 这一行（这个物理版本）的身份证 | 卡的**序列号**（每张不同）|
| `logical_asset_id` | 这一族（同一逻辑物的所有版本）的家族 ID | **身份证号**（不变）|
| `revision` | 在这一族里排第几版 | **换发次数**（1, 2, 3...）|
| `is_current` | 这一行是不是当前有效那一版 | 钱包里**当前那张**（只有一张）|

**具体演化例子**：先做了一个 clip，改了两次：

| 操作 | asset_id | logical_asset_id | revision | is_current |
|------|----------|------------------|----------|------------|
| ① 首版创建 | `aaa11111` | `aaa11111` ← 等于 asset_id（自引用）| 1 | ✅ true |
| ② 升 v2（同事务）| 旧版改：`aaa11111` | `aaa11111` | 1 | ❌ false |
|   | 新版插：`bbb22222` | `aaa11111` ← 抄首版的 | 2 | ✅ true |
| ③ 升 v3（同事务）| 旧版改：`bbb22222` | `aaa11111` | 2 | ❌ false |
|   | 新版插：`ccc33333` | `aaa11111` | 3 | ✅ true |

读法：
- 想找「**clipA 现在是哪版**」→ `WHERE logical_asset_id='aaa11111' AND is_current=true` → 一行
- 想找「**clipA 历史所有版本**」→ `WHERE logical_asset_id='aaa11111' ORDER BY revision`
- 客户引用 `aaa11111` → 永远拿到 v1 那行（不变内容，不会被覆盖）
- 客户引用 `bbb22222` → 永远拿到 v2 那行
- 客户引用 `logical='aaa11111' + current=true` → 永远拿到最新版

**关键不变式：首版 `logical_asset_id = asset_id`**（自己当家族族长）。后续所有新版都继承这个族长 ID，**永远不变**。

### 2.2 数据库层兜底：「同一族只能有一行 current」

不靠应用代码自觉，靠 PostgreSQL 一条**部分唯一索引**直接挡：

```sql
CREATE UNIQUE INDEX uq_assets_current_per_logical
  ON assets (logical_asset_id) WHERE is_current = TRUE AND is_deleted = FALSE;
```

「部分」(`WHERE`) 的意思是：只对**还没删除的当前版**那些行建唯一约束。历史的 false 行 / 已删除行随便重复无所谓。

**为什么要 DB 层挡？** 升新版那个事务要做两件事：①把旧版 `is_current=false` ②插新版 `is_current=true`。万一两个并发请求同时升级同一个 clip，应用代码再小心也可能让两行都 true。DB 这条索引会让其中一个事务**直接报唯一冲突回滚**，永远不会出现「两行都是 current」的脏数据。

### 2.3 版本链必须是一条直线，不能分叉

```
✅ 允许：v1 → v2 → v3 → v4

❌ 禁止三种：
   分叉：  v1 → v2a              （同时存在两个 v2）
                ↘ v2b
   跳号：  v1 ──────→ v3          （直接跳过 v2）
   倒退：  v3 → v2                （v3 之后又退回 v2）
```

落到代码层（N1 不变式）：调 `CreateRevision(old_id)` 升新版时，必须满足两条：
1. `old.is_current = TRUE`（不能从一个已经过气的版本派生）
2. `new.revision = old.revision + 1`（必须是相邻下一号）

**「那 A/B 实验怎么办？我就是要让 v2 同时存在两个变体试一下哪个好」**

走另一条路：`derived_asset` + `derived_from` 边。A/B 实验本质不是「同一物的两个版本」，是「**从同一个源派生的两个独立产物**」。在版本链外另起两条派生关系，互不干扰。

```
clipA_v2 (主版本链)
 ├─ derived_from ← exp_a (实验产物 A)   ← 这俩是 derived_asset，不算 v2 的分叉
 └─ derived_from ← exp_b (实验产物 B)
```

**为什么坚持单链？** 一旦允许分叉，「`GET /logical/clipA/current` 返回哪个？」这种 API 就没法回答了。客户调用方必须每次自己判断「这个 logical 现在有几个 current 啊？」——平台契约直接崩。

---

## 3. A/B 路由：改东西时走哪条路

> 「A/B 路由」是项目内造的术语，意思是**改一个已有 asset 时的两种走法**，不是 A/B 实验，也不是网络路由。

### 3.1 两条路的分工

改 asset 时按**改的是什么字段**自动决定走哪条：

| 走法 | 干啥 | 对应字段 | 类比 |
|---|---|---|---|
| **A 路由 = 原地改** | `asset_id` 不变，UPDATE 那一行 | tag / metric / lifecycle / owner / metadata 这些「附带元信息」 | 修改文件的属性（标签、备注），文件内容不变 |
| **B 路由 = 生新版** | 插一行新 asset，新 `asset_id`，旧版翻 `is_current=false` | `storage_uri` / `files` / 时间窗 / `primary_label` / `labels` 这些「核心产物字段」 | `git commit` 一个新版本，旧版还在历史里 |

**`revision_triggers` 是什么？**

字面意思是「**会触发升版的字段名单**」。每种 asset_type 各自维护一张清单（在代码 `AssetTypeRegistry` 里）。**改了清单里的字段 = 必须升版；改清单外的字段 = 原地改就行。**

类比身份证换发规则：

> 改手机号 → 派出所不管。
> 改家庭住址 → 派出所不管。
> 改姓名 / 出生日期 / 照片 → **必须换发新卡**，老卡作废。

派出所内部那张「换发触发字段清单」就是 `revision_triggers`。资产场景里 clip 的清单长这样：

```python
clip.revision_triggers = [
    "storage_uri",        # mp4 路径变了 = 视频内容变了
    "files",              # 附属文件清单变了
    "start_timestamp_ns", # 时间窗起点变了
    "end_timestamp_ns",   # 时间窗终点变了
    "primary_label",      # 主标签变了 = 这片段表达的语义变了
    "labels",             # 标签集变了
]
```

**核心判断标准：「改了会让客户拿到的内容发生变化吗？」**

- ✅ 会变 → 进 revision_triggers，必须升版（旧版引用还能拿到旧内容）
- ❌ 不会变 → 不进清单，原地改（只是平台内部账本变化，客户无感知）

为什么要这套机制？因为**「啥都升版」太重**（改个 owner 就插一行新 asset，数据库爆炸 + 计费混乱），**「啥都原地改」又太危险**（storage_uri 改了客户上次下载的引用突然指向新内容 = 合规事故）。`revision_triggers` 就是这条平衡线。

### 3.2 客户端必须显式选路

**一句话规则：改触发字段不能用 PATCH 偷偷升版，必须客户端主动调升版接口。**

**「显式选路」对调用方的负担**：理论上每次写操作都要先判断「我改的字段在不在 `revision_triggers` 名单里」。手写 HTTP 调用容易踩坑（少调一次 POST /revisions 就拿 422）。

后续会提供配套工具收敛这个负担：

| 工具 | 场景 | 怎么帮忙 |
|---|---|---|
| **AssetWriter SDK**（语言绑定）| 业务代码集成 | `client.update_asset(id, {...changes})` 自动判断走 PATCH 还是 POST /revisions，调用方无需手判清单 |
| **AI 场景专用 CLI**（rev.13+ 计划）| 算法 / 标注 agent 跑批 | 一条命令封装「读 → 改 → 选路 → 写」流水线，含幂等 key 自动生成、run_id 关联、批量 revision 合并等 |

→ 文档里描述的 `PATCH 422 → POST /revisions` 是**底层契约**（保证审计干净）；上层 SDK / CLI 让常见路径变成一行调用，业务代码不必直接 PATCH/POST。

具体底层表现：

```text
# ❌ 错误用法：用 PATCH 改触发字段（storage_uri）
PATCH /api/v1/assets/clipA_v1
{
  "storage_uri": "gs://.../r2-uuid/video.mp4"
}

# 服务端响应
← 422 Unprocessable Entity
{
  "error": "revision_triggers_field_changed",
  "fields": ["storage_uri"],
  "hint": "use POST /assets/clipA_v1/revisions"
}
```

```text
# ✅ 正确用法：明确调升版接口
POST /api/v1/assets/clipA_v1/revisions
{
  "changes": { "storage_uri": "gs://.../r2-uuid/video.mp4" },
  "revision_reason": { "type": "algo_rerun", "run_id": "R002" }
}

← 201 Created
{ "asset_id": "bbb22222", "revision": 2, ... }
```

**为什么不允许 PATCH 自动升版？** 让平台代客户决定升版是个大坑：

| 想象一下「PATCH 自动升版」会发生什么 | 真实后果 |
|---|---|
| 运营手抖把 `storage_uri` 写错了一个字符，PATCH 上去 | 生产库**自动多出一行 v2**，GCS 也开了一个新目录，计费/监控全乱 |
| 客户端代码 `PATCH /assets/clipA_v1` 然后又 `GET /assets/clipA_v1` | 拿到 404——服务端偷偷换 ID 了，客户端不知道 |
| 审计日志看「谁创建了 v2」 | actor 是普通运营改 tag 的请求，分不清是有意升版还是误触 |

所以规则是：**升版必须像 `git commit` 一样有仪式感**，客户端**主动**调 `POST /assets/{id}/revisions`，明确说「我要升版，原因是 xxx」。这样：
- 失误改字段 → 拿到 422 错误，立刻发现，不会污染数据
- 升版是显式业务事件 → audit trail 干净（每条 revision 都有意图、有 reason、有 run_id 关联）

### 3.3 显式 B 路由 API

```text
POST /api/v1/assets/clipA_v1/revisions
Idempotency-Key: <uuid>
{
  "parent_asset_version": 5,                       ← 老版当前 row_version
  "changes": {
    "storage_uri": "gs://.../r2-uuid/video.mp4",
    "files": {...}
  },
  "revision_reason": {                              ← 写入 asset_relations(revision_of).metadata
    "type": "algo_rerun",
    "run_id": "R002",
    "algo": "hand_track@2.0"
  }
}

← 201 Created
{
  "asset_id": "clipA_v2",                          ← 新 asset_id
  "logical_asset_id": "clipA_v1",                  ← 继承
  "revision": 2,
  "is_current": true,
  "previous_asset_id": "clipA_v1"
}
Location: /api/v1/assets/clipA_v2
```

### 3.4 B 路由完整事务（含乐观锁 OPT1）

```sql
BEGIN;
  -- ① 老版乐观锁检查
  SELECT row_version FROM assets WHERE asset_id='clipA_v1' FOR UPDATE;
  -- 若 row_version != 5（client 传的 parent_asset_version）→ ROLLBACK + 409

  -- ② INSERT 新 asset（row_version=1，OPT1 不变式）
  INSERT INTO assets (asset_id='clipA_v2', logical_asset_id='clipA_v1', revision=2,
                      is_current=true, row_version=1, ...);

  -- ③ 老版下线
  UPDATE assets SET is_current=false, row_version=row_version+1, updated_at=now()
   WHERE asset_id='clipA_v1' AND row_version=5;   -- 双保险条件写
  -- 若 0 行 → ROLLBACK 409

  -- ④ logical_assets 更新 3 字段（LA4 已撤销，B 路由只更 3 字段）
  UPDATE logical_assets SET
    current_revision = 2,
    total_revisions  = total_revisions + 1,
    updated_at       = now()
  WHERE logical_asset_id='clipA_v1';

  -- ⑤ revision_of 边
  INSERT INTO asset_relations (
    parent_asset_id='clipA_v2', child_asset_id='clipA_v1',
    relation_type='revision_of',
    metadata='{"type":"algo_rerun","run_id":"R002","algo":"hand_track@2.0"}'::jsonb
  );

  -- ⑥ asset_revised event
  INSERT INTO asset_events (
    asset_id='clipA_v2', event_type='asset_revised',
    actor='algo_sdk:hand_track@2.0', system_metadata='{"run_id":"R002",...}',
    event_payload='{"changed_fields":["storage_uri","revision"],"before":{...},"after":{...}}'::jsonb
  );

  -- ⑦ 老版 governance 更新
  UPDATE assets SET lifecycle_state='superseded', updated_at=now() WHERE asset_id='clipA_v1';
COMMIT;
```

**OPT1 关键点（rev.10）：**

- 新 asset_id 的 `row_version` 总是 1（DEFAULT），**与老版无关**（per-row 乐观锁概念，不跨行传递）
- `parent_asset_version` 校验**老版**的 row_version，新版自己从 1 起
- 老版被 UPDATE `is_current=false` 时 row_version +1（与所有 A 路由 UPDATE 规则一致）

---

## 4. 算法产物的三种版本对应场景

业务上算法产物与版本的关系有 **3 种不同语义**，必须分别建模：

### 4.1 场景 A：同算法不同版本 → 同 logical_asset_id

**故事**：`segment_A` 是一段 60 秒原始视频。`hand_track` 算法专门从中找「抓握动作」精确切出一个 clip。算法升级了 3 次，每次在 segment_A 上重跑：

| 跑次 | 算法版本 | 产出 | 时间窗精度 |
|---|---|---|---|
| t1 | hand_track@1.0 | `clipX_v1` | 抓握片段 12.3s - 17.8s（粗略）|
| t2 | hand_track@2.0 | `clipX_v2` | 抓握片段 12.5s - 17.5s（更准）|
| t3 | hand_track@3.0 | `clipX_v3` | 抓握片段 12.62s - 17.45s（最准）|

**核心问题**：这 3 个 clip 是「**3 个独立资产**」还是「**同一个资产的 3 个版本**」？

业务直觉答案：**同一个**。客户买的是「segment_A 里那段抓握动作」这个**业务意图**，不关心算法迭代。算法升级只是让那一段切得更准，含义没变。所以应该当成一条版本链。

**落到 4 个字段**：

| 字段 | clipX_v1 | clipX_v2 | clipX_v3 |
|---|---|---|---|
| `asset_id`（物理 ID）| `aaa11111` | `bbb22222` | `ccc33333` |
| `logical_asset_id`（族 ID）| `aaa11111` | `aaa11111` ← 抄首版 | `aaa11111` ← 抄首版 |
| `revision` | 1 | 2 | 3 |
| `is_current` | false | false | **true** |

「同 logical_asset_id」就是说**这三行的家族 ID 都是 `aaa11111`**。找当前版一条 SQL：`WHERE logical_asset_id='aaa11111' AND is_current=true`。

**边表 `asset_relations` 写两类边**（一条都不能省）：

**第一类：每版都连到 segment_A**（横向「来自哪儿」）

```
clipX_v1 ──derived_from──► segment_A   metadata={run_id:R001, algo:hand_track@1.0}
clipX_v2 ──derived_from──► segment_A   metadata={run_id:R005, algo:hand_track@2.0}
clipX_v3 ──derived_from──► segment_A   metadata={run_id:R012, algo:hand_track@3.0}
```

读法：「**clipX_v1 是从 segment_A 派生的**」「v2 也是」「v3 也是」。每条边 `metadata` 里记着是哪次跑、什么算法版本，回答「v3 是哪个算法跑的？」一查就知道。

**第二类：相邻版本之间连版本链**（纵向「上一版是谁」）

```
clipX_v2 ──revision_of──► clipX_v1
clipX_v3 ──revision_of──► clipX_v2
```

读法：「**clipX_v2 是 clipX_v1 的新版**」「v3 是 v2 的新版」。注意只连**相邻**的（v2→v1, v3→v2），**不直连** v3→v1（§2.3 严格线性，每条边只跨一步）。

**为什么要两类边？** 维度不同：

| 边类型 | 维度 | 典型查询 |
|---|---|---|
| `derived_from` | 横向：从哪派生、谁跑的 | 「hand_track@2.0 跑出过哪些产物」「这个 clip 来源是哪个 segment」|
| `revision_of` | 纵向：版本链上下游 | 「clipX 的历史所有版本」「v3 上一版是谁」|

一条边表达不了两件事，所以分。

**整体图**：

```
                        segment_A (来源)
                        ▲   ▲   ▲
                        │   │   │  ← 三条 derived_from 边
                        │   │   │     metadata 各带 algo 版本和 run_id
                        │   │   │
            ┌───────────┘   │   │
            │       ┌───────┘   │
            │       │       ┌───┘
        clipX_v1 ◄──── clipX_v2 ◄──── clipX_v3
         ↑              ↑              ↑
     首版 (族长)    revision_of    revision_of
                                 当前版 is_current=true

        三行都共享同一个 logical_asset_id = aaa11111
```

**一句话**：同算法不同版本 = 同一个东西越切越准 = 同 logical_asset_id（一条版本链）。每版同时连两类边：`derived_from` 记来源 + 算法身份，`revision_of` 记版本顺序。

### 4.2 场景 B：不同算法 → 各自独立 logical_asset_id

```
segment_A
  ├─ hand_track@2.0       ──► clipX_v1     (logical_id=L_clipX)
  └─ action_detector@1.5  ──► actionB_v1   (logical_id=L_actionB)
```

**业务直觉：** hand_track 和 action_detector 是**两件不同的事**，产物语义不同，**不存在版本关系**。

`asset_relations`：
```
clipX_v1   --derived_from--> segment_A   metadata={run_id:'R_handtrack_001', algo:'hand_track@2.0'}
actionB_v1 --derived_from--> segment_A   metadata={run_id:'R_actiondet_001', algo:'action_detector@1.5'}
clipX_v1 和 actionB_v1 无关系
```

### 4.3 场景 C：同算法一次跑出多个独立产物

**故事**：`action_detector` 这个算法跑一次 `segment_A`，**一次性识别出多个动作时间窗**（每个动作有自己的开始/结束时间 + 动作类型标签）。这是和场景 A 最大的区别：

- 场景 A：算法每次只切**一个**精准时间窗（hand_track 的产物就是一个抓握片段）
- 场景 C：算法一次切**多个独立片段**（action_detector 一次扫完整段视频，找出抓、拿、放等多个动作）

**第一次跑（v1.5 版算法）**：

```
segment_A
  └─ action_detector@1.5  →  run_id=R001
                              ├─ actionB_1  (logical_id=L_actB1, t=100-200ms, label=grasp)
                              ├─ actionB_2  (logical_id=L_actB2, t=300-450ms, label=pickup)
                              └─ actionB_3  (logical_id=L_actB3, t=600-700ms, label=drop)
```

**关键设计**：3 个 action 是 **3 个独立逻辑物**，各自有不同的 `logical_asset_id`：

| | actionB_1 | actionB_2 | actionB_3 |
|---|---|---|---|
| `asset_id` | `act11111` | `act22222` | `act33333` |
| `logical_asset_id` | `act11111` | `act22222` | `act33333` |
| `revision` | 1 | 1 | 1 |
| 时间窗 | 100-200ms | 300-450ms | 600-700ms |
| 动作标签 | grasp（抓握）| pickup（拿起）| drop（放下）|

**为什么不是「同一物的多版本」？** 这三个动作**时间窗不同 + 语义不同**——抓和拿是两个独立动作，不是「同一动作越切越准」。客户买的也是独立的「抓样本 / 拿样本 / 放样本」，不是「同一动作的迭代」。所以**各自一条版本链**。

**所有 action 共享同一个 run_id**（`R001`），都通过 `derived_from` 边连到 segment_A：

```
actionB_1 ──derived_from──► segment_A   metadata={run_id:R001, algo:action_detector@1.5, label:grasp}
actionB_2 ──derived_from──► segment_A   metadata={run_id:R001, algo:action_detector@1.5, label:pickup}
actionB_3 ──derived_from──► segment_A   metadata={run_id:R001, algo:action_detector@1.5, label:drop}
```

**第二次跑（v2.0 算法升级，识别更多动作）**：

```
segment_A
  └─ action_detector@2.0  →  run_id=R002
                              ├─ actionB_1' (t=105-200ms, label=grasp)    ← 跟 actionB_1 几乎一样，时间微调
                              ├─ actionB_2' (t=300-440ms, label=pickup)   ← 跟 actionB_2 几乎一样
                              ├─ actionB_3' (t=595-705ms, label=drop)     ← 跟 actionB_3 几乎一样
                              ├─ actionB_4  (t=400-500ms, label=lift)     ← 新发现的动作（v1.5 没识别出来）
                              └─ actionB_5  (t=800-850ms, label=rotate)   ← 新发现的动作
```

**两类产物分别处理**：

| 产物 | 走哪条路 | 原因 |
|---|---|---|
| actionB_1' / 2' / 3' | **场景 A 模式**：同 logical_id（`L_actB1` 等），revision=2 | 是旧动作的精修，业务上是「同一物的迭代」|
| actionB_4 / 5 | **全新 logical_id**（`L_actB4` / `L_actB5`），revision=1 | 是「新发现的物」，不是旧 action 的新版 |

**SDK 怎么决定走哪条路？** 由算法 SDK 用户在 `result.match_existing()` 里判断（详见 §5）。匹配规则可以是：
- 时间窗 IoU > 0.7 + label 一致 → 当作旧物的新版（同 logical_id）
- 否则 → 全新 logical_id

### 4.3.1 多类别动作的字段存储

一个 `action_detector` 跑次可以一次识别出多种动作（抓 / 拿 / 放 / 推 ...），每种动作要带各自的元信息（涉及物体、手、置信度等）。**每个识别出的动作都是独立 asset**（`asset_type='action'`），分层存储：

| 字段 | 放哪 | 为什么 |
|---|---|---|
| 主动作类型 `label`（grasp / pickup ...） | `actions.label` | 高频 GROUP BY / 单值 / 强类型索引 |
| `confidence` | `actions.confidence` | 数值排序过滤 |
| 涉及物体 / 手 / 场景等扩展维度 | `asset_tags`（key/value）| 多值 / 算法升级可加新维度 |
| 来源 + 算法身份 | `asset_relations.metadata`（含 `run_id` / `algo_version` / `label`） | 血缘查询 |
| 业务自由字段 | `assets.metadata` JSONB | 不索引的杂项 |

写入时 5 张表同事务（assets / actions / asset_tags / asset_relations / asset_events），完整 SQL 见 [`asset-tagging.md`](asset-tagging.md) §3 & [`asset-hierarchy-and-derivatives.md`](asset-hierarchy-and-derivatives.md) §3.2。常见查询（按 label / 双维度筛 / 跑次动作分布统计）见下文 §4.4。

**核心约定**：同一跑次（`run_id`）可一次产出几十上百个 action，全部通过 `run_id` 关联回 `algo_runs` 一行，便于回滚和审计。

### 4.4 常见筛选用例（高频）

业务方最常问的几类问题，对应 SQL 模板：

#### 用例 1：「指定算法处理过的所有产物」

```sql
-- 所有被 hand_track 算法（任何版本）处理过的 asset
SELECT DISTINCT ar.parent_asset_id AS asset_id
FROM asset_relations ar
WHERE ar.relation_type = 'derived_from'
  AND ar.metadata->>'algo_name' = 'hand_track';
```

#### 用例 2：「指定算法 + 指定版本的产物」

```sql
-- hand_track@2.0 跑出的所有 asset（直接走 metadata，不必 JOIN algo_runs）
SELECT parent_asset_id AS asset_id
FROM asset_relations
WHERE relation_type = 'derived_from'
  AND metadata->>'algo_name'    = 'hand_track'
  AND metadata->>'algo_version' = '2.0';
```

#### 用例 3：「指定算法 + 版本 + 当前版资产」

```sql
-- hand_track@2.0 跑出的、当前 is_current=true 的产物
SELECT a.asset_id, a.logical_asset_id, a.revision
FROM asset_relations ar
JOIN assets a ON a.asset_id = ar.parent_asset_id
WHERE ar.relation_type = 'derived_from'
  AND ar.metadata->>'algo_name'    = 'hand_track'
  AND ar.metadata->>'algo_version' = '2.0'
  AND a.is_current = true
  AND a.is_deleted = false;
```

#### 用例 4：「某个 logical 物的算法演进史」

> 「clipX 这条版本链每一版是哪个算法版本跑出来的？」（场景 A 的回看）

```sql
SELECT 
  a.revision,
  a.asset_id,
  a.is_current,
  ar.metadata->>'algo_version' AS algo_version,
  ar.metadata->>'run_id'        AS run_id,
  a.created_at
FROM assets a
LEFT JOIN asset_relations ar 
  ON ar.parent_asset_id = a.asset_id 
  AND ar.relation_type = 'derived_from'
WHERE a.logical_asset_id = 'L_clipX'
ORDER BY a.revision;
```

输出示例：

| revision | asset_id | is_current | algo_version | run_id    | created_at |
|----------|----------|------------|--------------|-----------|------------|
| 1        | clipX_v1 | false      | 1.0          | R001      | t1         |
| 2        | clipX_v2 | false      | 2.0          | R005      | t2         |
| 3        | clipX_v3 | true       | 3.0          | R012      | t3         |

#### 用例 5：「某次算法跑次的所有产物」

> 「R789 那次跑出来了哪些 asset？」（场景 C 的盘点）

```sql
SELECT a.asset_id, a.asset_type, a.logical_asset_id, a.start_timestamp_ns, a.end_timestamp_ns
FROM assets a
JOIN asset_relations ar ON ar.parent_asset_id = a.asset_id
WHERE ar.relation_type = 'derived_from'
  AND ar.metadata->>'run_id' = 'R789';
```

#### 用例 6：「同 segment 上多个算法各自产出对比」

> 「segment_A 上 hand_track 和 action_detector 各自跑出了什么？」（场景 B 的对比）

```sql
SELECT 
  ar.metadata->>'algo_name'    AS algo,
  ar.metadata->>'algo_version' AS version,
  ar.parent_asset_id           AS produced_asset,
  a.asset_type
FROM asset_relations ar
JOIN assets a ON a.asset_id = ar.parent_asset_id
WHERE ar.child_asset_id = 'segment_A'
  AND ar.relation_type = 'derived_from'
ORDER BY algo, version;
```

#### 用例 7：「指定算法版本范围（升级回滚场景）」

> 「找出所有由 hand_track@1.x 跑出来、还没被 2.x 重新处理过的 asset」

```sql
WITH old_results AS (
  SELECT parent_asset_id, child_asset_id
  FROM asset_relations
  WHERE relation_type = 'derived_from'
    AND metadata->>'algo_name' = 'hand_track'
    AND metadata->>'algo_version' LIKE '1.%'
),
new_processed AS (
  SELECT child_asset_id
  FROM asset_relations
  WHERE relation_type = 'derived_from'
    AND metadata->>'algo_name' = 'hand_track'
    AND metadata->>'algo_version' LIKE '2.%'
)
SELECT old.parent_asset_id AS asset_id, old.child_asset_id AS source_segment
FROM old_results old
LEFT JOIN new_processed new ON new.child_asset_id = old.child_asset_id
WHERE new.child_asset_id IS NULL;
-- → 这些 segment 还没被 hand_track@2.x 重跑，可以排队回填
```

#### 索引覆盖

`schema.md` 里这几条索引保证上述查询都走索引：

```sql
CREATE INDEX idx_arel_derived_run ON asset_relations((metadata->>'run_id'))
                                  WHERE relation_type = 'derived_from';
CREATE INDEX idx_assets_logical   ON assets(logical_asset_id);
CREATE INDEX idx_algo_runs_name_version ON algo_runs(algo_name, algo_version);
```

如果业务真的高频按 `algo_name` / `algo_version` 单独筛，可以考虑给 `asset_relations` 加 GIN on metadata 或拆出 `algo_name_norm` / `algo_version_norm` 生成列 + B-tree 索引（rev.13 评估）。

---

## 5. logical_asset_id 由算法 SDK 显式指定（rev.12 C3 关键决策）

**核心设计原则：** 平台**不做自动 matching**，由算法 SDK 自己判断「新产物是旧 logical 的新版本」还是「全新 logical 物」。

### 5.1 SDK 调用示例

```python
# 算法 SDK 用户代码
result = action_detector.run(segment_A, version="2.0")

for action in result.actions:
    if action.matched_to_existing:                  # 算法内部 matching 逻辑
        # 是 actionB_1 的新版本（场景 A）
        client.create_asset(
            asset_type="action",
            parent_asset_id="segment_A",
            logical_asset_id=action.matched_logical_id,   # ← 显式指定继承
            run_id="R002",
            payload={...}
        )
    else:
        # 全新的 logical 物（场景 C 新发现）
        client.create_asset(
            asset_type="action",
            parent_asset_id="segment_A",
            logical_asset_id=None,                  # ← 平台分配新 logical_id (= asset_id 自引用)
            run_id="R002",
            payload={...}
        )
```

### 5.2 为什么 SDK 决定而不是平台

| 角度 | 算法 SDK | 平台规则 |
|------|---------|---------|
| 知道产物语义 | ✓ 算法最懂（confidence / 时间窗匹配阈值等内部参数）| ✗ 只能用通用规则（如时间重叠 80%）|
| 边界 case 处理 | ✓ 每个算法独立 | ✗ 规则总会遗漏 |
| 调试 / 责任归属 | ✓ matching 错 = 算法 bug，清晰 | ✗ matching 错 = 平台 bug，难定位 |
| 实现成本 | 低（平台仅校验） | 高（规则引擎 + 配置 + 测试） |

### 5.3 API 行为

| API | logical_asset_id 处理 |
|-----|---------------------|
| `POST /assets` 不带 logical_asset_id | 平台分配新 logical_id = asset_id（self-ref，首版）|
| `POST /assets` 带 logical_asset_id | 校验该 logical_id 存在 + asset_type 一致 → revision++ 升级 |
| `POST /assets/{id}/revisions` 显式 B 路由 | 继承 parent 的 logical_asset_id，revision = parent.revision + 1 |

### 5.4 Validator 强约束

| # | 规则 |
|---|------|
| 带 logical_asset_id 时，该 id 必须已存在于 logical_assets 表 | 否则 422 |
| 新 asset 的 asset_type 必须与 logical_assets.asset_type 一致（LA1）| 否则 422 |
| 新 asset 的 revision 必须等于 max(同 logical revision) + 1 | 否则 422 |
| 新 asset is_current=true 时，旧的同 logical 必须 is_current=false 同事务切换 | DB partial unique 兜底 |

---

## 6. 业务场景 walkthrough

### 6.1 客户视角（API 体验）

```text
GET /api/v1/assets/abc123ef
→ {asset_id: "abc123ef", revision: 1, is_current: false, lifecycle: "superseded",
   storage_uri: "gs://.../r1-uuid/video.mp4"}
   ← 客户半年前买的，GCS 文件永远在，永远可下载

GET /api/v1/logical-assets/L_clipA/current
→ {asset_id: "hij789kl", revision: 3, is_current: true, lifecycle: "ready",
   storage_uri: "gs://.../r3-uuid/video.mp4"}
   ← 跟随到当前最新版

GET /api/v1/logical-assets/L_clipA
→ {revisions: [
     {asset_id: "abc123ef", revision: 1, lifecycle: "superseded"},
     {asset_id: "def456gh", revision: 2, lifecycle: "superseded"},
     {asset_id: "hij789kl", revision: 3, lifecycle: "ready"}
   ]}
   ← 查这个 clip 的完整演进史
```

### 6.2 算法升级 → 新版本通知

```text
① algo R001 跑出 clipX_v1
② 半年后 algo upgrade 到 v2.0，R002 跑出 clipX_v2（revision=2，继承 logical_id）

③ 平台自动触发 RevisionNotifier ActionHandler:
   SELECT delivery_id, customer_id
   FROM delivery_items di JOIN deliveries d USING (delivery_id)
   WHERE di.asset_id IN (
     SELECT asset_id FROM assets
     WHERE logical_asset_id='L_clipX' AND revision < 2
   )
   → 通知运营/客户「L_clipX 有新版本 v2 可续交付」

④ 客户决定续交付：
   POST /api/v1/deliveries (new)
   POST /api/v1/deliveries/{id}/items
   {asset_id: "clipX_v2", expected_revision: 2, payload_mode: "materialized"}
```

→ **客户老链接（clipX_v1）永久有效** + **平台主动推送新版本** = 完整商业增值闭环。

### 6.3 合规审计（PII 流向溯源）

```sql
-- 「PII 资产 seg_001 流向了哪些客户的 delivery」
-- 注意：边方向 split_from / derived_from / contains 的 parent_asset_id 是产物（下游），
--   child_asset_id 是源（上游）。所以「找下游」= 起点出现在 child_asset_id，跳到 parent_asset_id。
WITH RECURSIVE downstream AS (
  -- 第一层：seg_001 作为 child（源），找它的 parent（产物，即下游）
  SELECT parent_asset_id AS asset_id, 1 AS hops
  FROM asset_relations
  WHERE child_asset_id = 'seg_001'
    AND relation_type IN ('derived_from', 'split_from', 'contains')
  UNION ALL
  -- 递归：上一层的 asset 继续作为 child，找它的 parent
  SELECT ar.parent_asset_id, d.hops + 1
  FROM asset_relations ar
  JOIN downstream d ON d.asset_id = ar.child_asset_id
  WHERE d.hops < 10
    AND ar.relation_type IN ('derived_from', 'split_from', 'contains')
)
SELECT DISTINCT del.customer_id, di.asset_id, a.revision
FROM downstream w
JOIN delivery_items di ON di.asset_id = w.asset_id
JOIN assets a ON a.asset_id = di.asset_id
JOIN deliveries del ON del.delivery_id = di.delivery_id;
```

→ 一次 SQL 拉清「PII 资产 → 所有下游产物 → 所有 delivery → 所有客户」。

---

## 7. 不变式与边界

| # | 规则 | 实现 |
|---|------|------|
| **N1** | 同 logical_asset_id 仅一条 is_current=true | DB partial unique index |
| **N2** | revision 严格线性（v1→v2→v3，禁止分叉 / 跳号 / 倒退）| AssetWriteValidator |
| **AR1**（rev.11）| PATCH 命中 revision_triggers → 422 + 指引 POST /revisions | API gateway + Validator |
| **OPT1**（rev.10）| B 路由新 asset_id 行 row_version=1 起，与老版无关 | AssetWriter |
| **LA1** | logical_assets.asset_type 不可改 | PG trigger |
| **LA2** | current_revision <= total_revisions | DB CHECK |
| **AE1** | 任何业务表写入必同事务写 asset_events | AssetWriter 强制 |
| **AE2** | asset_events append-only（禁止 UPDATE/DELETE）| PG GRANT 限制 |
| **IR1**（rev.10 + rev.12 修订）| **强一致点查走 PG，列表/搜索走 ES（接受 1-3s 滞后）**| code review + API 文档 |

### 7.5 ES 投影延迟与 `is_current` 切换（rev.12 关键边界）

> **真盲点**：B 路由的「老版 is_current=false + 新版 is_current=true」切换在 PG 内是**同事务原子**，但 ES 投影通过 Outbox CDC 异步同步（含 ~2s safety lag + Pub/Sub 推送延迟，总滞后通常 1-3s）。
>
> 如果客户端在切换后立即查 ES，可能看到「老版 `is_current=true`」的过期数据。

**应用 IR1 解决：**

| 场景 | 路径 | 一致性 |
|------|------|--------|
| `GET /logical-assets/{logical_id}/current`（强一致点查最新版）| **PG**（按 `logical_asset_id` + `is_current=true` 查 `assets` 表，PG partial unique 兜底单一行）| **强一致**（B 路由原子事务）|
| `GET /assets/{asset_id}`（精确点查指定版本）| **PG** | **强一致**（PK 查询）|
| `POST /queries/run`（按 tag / metric / 时间窗 / `is_current` 复合过滤的列表搜索）| **ES** | **最终一致**（1-3s 滞后）；客户端必须容忍 |
| 列表里偶发显示「老版 `is_current=true`」 | 接受，1-3s 后自愈 | 业务可接受（搜索/浏览场景）|

**给 client 的契约**：
- 「我要稳拿当前版本的具体 asset_id」→ 调 `GET /logical-assets/{id}/current`（强一致）
- 「我要列出所有 is_current 的 raw_mcap」→ 调 `POST /queries/run`（接受 1-3s 滞后）
- **不允许**绕过这条规则去 ES 做点查最新版（会拿到过期数据）

---

## 8. 与其他模块的关系

| 模块 | 关系 |
|------|------|
| **`asset-hierarchy-and-derivatives.md`** | 任何 asset_type 都可走 B 路由（含 raw_mcap）|
| **`algo-runs.md`** | algo run 产物可带 logical_asset_id 表达 matching；revision_of 边 metadata 带 run_id |
| **`asset-tagging.md`** | tag 跟随 asset_id（不跟 logical）；新版本上线时 tag 重新打 |
| **`customers-and-deliveries.md`** | delivery_items 永远指**具体 asset_id**（frozen 快照，客户老链接稳定）|

---

## 9. 不做的事（Out of Scope）

| 不做的事 项 | 原因 |
|-------|------|
| **revision 分支 / merge**（如 git branch）| N1 严格线性；A/B 实验走 derived_asset，避免版本链复杂化 |
| **跨 logical_asset_id 合并** | 业务无诉求；merged_from 多对一已能表达融合 |
| **客户端可见 logical_asset_id 隐藏** | API 必须暴露（客户问"我买的现在最新版是啥"）|
| **PATCH 自动升 B**（A2）| AR1 改 422（rev.11 翻转）|
| **平台自动 matching**（rule-based）| 算法 SDK 显式指定（rev.12 C3）|
| **revision 自动 reuse**（删了 v2 重建）| 严格单调递增，不复用号 |

---

## 10. 决策追溯

| 决策 | 来源 |
|------|------|
| logical_asset_id + revision + is_current 三字段模型 | rev.5 初版 |
| N1 严格线性 | rev.5（grill round 1） |
| partial unique index 兜底 | rev.5 schema |
| OPT1 新 asset_id row_version=1 起 | rev.10 第二轮 grill F1 |
| AR1 PATCH 422（撤回 A2 自动升级）| rev.11 cleanup |
| 三场景产物-版本对应（同算法 / 不同算法 / 同算法多产物）| rev.12 C2 |
| logical_asset_id 由 SDK 显式指定 | rev.12 C3 |
| LA3/LA4 撤销（rev.11 砍 current_asset_id 后自动消除）| rev.11 |
| `revision_reason` 走 asset_relations(revision_of).metadata 而非 assets 列 | rev.9 |
