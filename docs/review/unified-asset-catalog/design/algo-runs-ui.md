# 算法 Run 展示 — UI 设计规范（CYB-1018 UI）

| 字段 | 值 |
|------|-----|
| 状态 | **Design spec**（P1 tracer）|
| Linear | [CYB-1031](https://linear.app/cyberorigin/issue/CYB-1031) |
| 依赖 API | `GET /api/v1/algo-runs/{run_id}`（CYB-1018 backend）|
| 设计系统来源 | ui-ux-pro-max — Data-Dense Dashboard + Ant Design 现网 token |
| 关联 | [`algo-runs.md`](./algo-runs.md)、[`asset-versioning-provenance-ui.md`](./asset-versioning-provenance-ui.md) |

---

## 1. 设计目标

| 用户 | 任务 | 成功标准 |
|------|------|----------|
| 运营 | 看某 asset 上算法是否绑定了全局 run | 算法 Tab 有 **来自 run** 列，16 位 id 可点 |
| 工程师 | 快速看 run 元数据（谁触发、状态、统计）| 点击 id → Popover 摘要，无需新页面 |
| 版本审计 | 区分「版本升级触发 run」与「算法执行 run」| 溯源 Tab 保留文案 **触发 run**；仅 16 位 id 可链到 `algo_runs` |

**非目标**：Run 列表页、受影响 asset 反查、取消 run。

---

## 2. 视觉与交互（ui-ux-pro-max）

- **Pattern**：Data-Dense Dashboard — 表格内链、Popover 详情、不增页面跳转
- **颜色**：沿用 Ant `colorPrimary` / `colorTextSecondary`；run 状态 Tag 用 `success` / `processing` / `error` / `default`
- **字体**：run id 用 `Typography.Text code` + `text-xs`（与 asset_id 一致）
- **触控**：链接最小点击区 ≥ 44px 高度（`padding` 扩展 hit area）
- **动效**：Popover 打开 150–200ms；加载用 `Spin` size small

---

## 3. 组件：`RunIdLink`

```
[来自 run]  R001…def456   ← Button type="link" size="small" + monospace
                │
                ▼ click
        ┌─────────────────────────┐
        │ hand_track @ 2.0        │
        │ status: ok              │
        │ triggered_by: manual:…  │
        │ processed: 10 / ok: 9   │
        │ [复制完整 run_id]       │
        └─────────────────────────┘
```

**规则**

| `run_id` 形态 | 展示 |
|---------------|------|
| 空 | `—` |
| 非 `^[0-9A-Za-z]{16}$`（遗留短 id）| 纯文本 `code`，无请求 |
| 合法 16 位 | 截断 `前4…后4` + 可点击 Popover |

**错误**：404 → Popover 内「未找到 run 登记」；网络错误 → 简短 message。

---

## 4. 落点

| 位置 | 变更 |
|------|------|
| **算法处理** Tab 表格 | 新增列 **来自 run**（在「状态」后），宽 ~140px |
| **版本与溯源** 时间线 | `触发 run:` 行用 `RunIdLink`（合法 id 时）|
| **资产列表** AlgoStatusPopover | Run ID 行改用 `RunIdLink` |

---

## 5. 线框（算法 Tab 片段）

```
| 算法              | 状态   | 来自 run        | 开始时间   | 耗时 | 产物/原因 | 操作 |
| hand_track@2.0    | 待处理 | —               | —          | —    | —         |      |
| env_analysis@1.0  | 完成   | R001…f456 ↗     | 05-22 …    | 12s  | gs://…    | 重置 |
```

---

## 6. 验收

- [ ] `9KnuP7F3` / 有 `run_id` 的测试资产：Popover 能拉到 run 行
- [ ] `cyb1013-verify` 类短 id：仍为纯文本，无 404 请求
- [ ] Console 无 error；`npm run build` 通过
