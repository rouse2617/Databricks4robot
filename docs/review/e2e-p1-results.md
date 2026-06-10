# P1 E2E 测试结果 — Pipeline UI

**日期:** 2026/5/29
**环境:** http://127.0.0.1:5176/pipeline (dev-token)
**执行方式:** Chrome DevTools MCP

---

## 结果汇总

| TC | 描述 | 结果 | 备注 |
|----|------|------|------|
| TC-1 | 保存模板 | **PARTIAL** ⚠️ | Save API 201 成功，但 Deploy tab 模板列表未刷新（UI bug） |
| TC-2 | 部署流程 | **PASS** ✅ | dialog 弹出 → 预览 → 部署成功 → workflow ID `my-pipeline-55d37f` |
| TC-3 | 导入/导出 JSON | **PASS** ✅ | Export 显示 JSON panel；Import modal 接受 JSON 并渲染画布 |
| TC-4 | 模板列表选择加载 | **PASS** ✅ | 空画布时模板列表显示；点击"编辑"后模板加载到画布 |
| TC-5 | 组件面板拖拽 | **PASS** ✅ | 左侧组件面板显示 5 个组件；拖拽到画布节点成功添加 |
| TC-6 | 节点配置面板 | **PASS** ✅ | 点击节点 → 右侧面板显示节点信息；双击/右键 → 配置 dialog 弹出含完整配置项 |

---

## 详细记录

### TC-1: 保存模板 — PARTIAL ⚠️
- Save 点击后 network 请求返回 **201 Created**
- API payload 正确：`my-pipeline` 包含 2 个节点（busybox + step-1-writer）
- API response 包含 pipeline ID: `56254165-15ad-4617-830d-65dab737945c`
- **问题:** 切换到 Deploy tab 后，模板列表仍显示"暂无已保存的流水线模板"，列表未刷新
- 截图: `docs/review/e2e-p1-tc1-deploy-tab-empty.png`

### TC-2: 部署流程 — PASS ✅
- Canvas 有节点时点击"部署" → Deploy dialog 正确弹出
- Dialog 显示：当前模板 `my-pipeline`，2 个节点，0 条连线
- 可选绑定资产（高级折叠展开正常）
- 点击"部署" → "正在部署流水线..." → **"部署成功"** → 显示 workflow ID `my-pipeline-55d37f`
- 部署时间: `2026/5/29 09:59:11`
- 提供"查看 Workflow" 和"查看部署"按钮

### TC-3: 导入/导出 JSON — PASS ✅
- 点击"导出" → JSON panel 出现，显示完整 pipeline JSON（含 name/version/nodes/edges）
- 点击"导入" → Import modal 弹出，提示"粘贴 Pipeline JSON，将替换当前画布内容"
- 粘贴 JSON 后点击"导入" → Canvas 出现 2 个节点（busybox, step-1-writer）

### TC-4: 模板列表选择加载 — PASS ✅
- 空画布时模板列表正常显示（共 60 个模板）
- 点击模板行的"编辑"按钮 → 画布加载该模板的节点
- 画布 heading 从"我的流水线"变为"流水线设计"，节点出现在画布

### TC-5: 组件面板拖拽 — PASS ✅
- 左侧组件面板显示 5 个可用组件（busybox, Pass Through, step-1-writer, step-2-reader, verify-test-comp）
- 从组件面板拖拽 "Pass Through" 到画布 → 节点成功添加到画布
- 拖拽后画布节点数量从 2 增加到 3
- 截图: `docs/review/e2e-p1-tc5-drag-result.png`

### TC-6: 节点配置面板 — PASS ✅
- 单击画布节点 → 右侧配置面板显示节点基本信息（名称、镜像）
- 提供"配置节点"按钮和"关联资产"显示
- 双击节点 → **"节点配置" dialog** 完整弹出，包含：
  - 节点名称（可编辑）
  - 命令（sh -c）
  - 参数（可添加）
  - 环境变量（可新增）
  - CPU / 内存 / 磁盘资源配置
  - 取消和保存按钮
- 截图: `docs/review/e2e-p1-tc6-node-config.png`

---

## 截图清单
- `docs/review/e2e-p1-initial-state.png` — 初始画布状态
- `docs/review/e2e-p1-tc1-deploy-tab-empty.png` — TC-1 保存后 Deploy tab 模板列表空（bug 证据）
- `docs/review/e2e-p1-tc5-drag-result.png` — TC-5 拖拽结果（3 个节点）
- `docs/review/e2e-p1-tc6-node-config.png` — TC-6 节点配置 Dialog

---

## Bug 记录
1. **TC-1 模板列表不刷新**: Save API 返回 201 成功，但切换到 Deploy tab 后模板列表未更新显示新保存的模板
