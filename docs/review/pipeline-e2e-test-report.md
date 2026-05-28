# Pipeline 全功能 E2E 测试报告

> **环境**: https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app  
> **最近复测**: 2026-05-28（用户反馈已修复后）  
> **方式**: Chrome DevTools MCP + `curl` + Network 抓包  
> **结论**: **路由已从 404 恢复为 503（handler 已挂载）；前端多项 UX/数据问题已修；全链路仍被 `k8s/argo workflows client is not configured` 阻塞。**

---

## 复测摘要（2026-05-28）

| 维度 | 上次（404） | 本次 |
|------|-------------|------|
| `GET/POST /pipelines` | 404 | **503** — `pipeline service unavailable: k8s/argo workflows client is not configured` |
| `GET /deployments` | 404 | **503**（同上） |
| `GET /workflows` | 404 | **503**（同上） |
| `GET /components` | 200 | **200** ✅ |
| 侧栏组件数 | 3（Pass Through ×2） | **2** ✅（去重生效） |
| 镜像展示 | `busybox:latest:latest` | **`busybox:latest`** ✅ |
| 画布空态 | 无 | **「从左侧拖入组件 → 连接圆点 → 保存/部署」** ✅ |
| 部署 Tab 标题 | 「运行记录」 | **「已保存模板 / 部署历史」** ✅ |
| 导入/导出契约 | ✅ | ✅（`step-1.output → step-2.input`） |
| 保存/部署 | ❌ 404 | ❌ **503**（需配 K8s/Argo） |

**下一步（运维）**：在 backend-dev 配置 Argo Workflows client（`K8S_*` / in-cluster config），确认启动日志无 `k8s client unavailable`，直至 `POST /pipelines` 返回 **201**。

---

## 问题汇总

### P0 — 阻塞

| ID | 问题 | 状态 | 说明 |
|----|------|------|------|
| **P0-1** | Backend 未注册 pipeline/workflow 路由 | **已缓解** | 404 → 503，说明 handler 已挂载 |
| **P0-2** | **K8s/Argo client 未配置** | **未解决** | 503：`k8s/argo workflows client is not configured`；保存/部署/列表均不可用 |
| **P0-3** | 全链路 E2E | **未通过** | 依赖 P0-2 |

### P1 — 功能 / 数据

| ID | 问题 | 状态 |
|----|------|------|
| **P1-1** | 组件重复（Pass Through ×2） | **✅ 已修**（UI 显示 2 个组件；API 仍可能有两条，前端已去重） |
| **P1-2** | 镜像 `image:tag:tag` | **✅ 已修**（展示 `busybox:latest`） |
| **P1-3** | 部分组件 ports 为空 | ⚠️ 未单独复测 |
| **P1-4** | 画布手工连线 | **✅ 导入链路边可见**（a11y：`Edge from step-1 to step-2`）；拖 handle 仍未自动化 |
| **P1-5** | 空 template_id FK | ✅ 代码已 save→deploy；**待 P0-2 后复测** |

### P2 — UI / UX

| ID | 问题 | 状态 |
|----|------|------|
| **P2-1** | 画布节点偏大 | ⚠️ 未改（仍双行 + min-width） |
| **P2-2** | 空组件库文案 | ⚠️ 未测（当前有 2 组件） |
| **P2-3** | 画布空态引导 | **✅ 已修** |
| **P2-4** | 部署 Tab 标题 | **✅ 已修** |
| **P2-5** | API 失败静默 | ⚠️ 503 时列表仍为 0，未见 Alert（toast 可能被快照漏掉） |
| **P2-6** | a11y 表单 id | ⚠️ 仍有 1 条 Chrome issue |

---

## 1. API 状态（dev-token）

| API | HTTP | 响应摘要 |
|-----|------|----------|
| `GET /api/v1/components` | **200** | 含 Pass Through、Python Script |
| `GET /api/v1/pipelines` | **503** | `pipeline service unavailable: k8s/argo...` |
| `POST /api/v1/pipelines` | **503** | 同上（保存请求体契约正确，见下） |
| `GET /api/v1/deployments` | **503** | 同上 |
| `GET /api/v1/workflows` | **503** | `workflow service unavailable: k8s/argo...` |

---

## 2. 测试矩阵与结果（复测）

### A. `/pipeline` — 画布

| # | 功能 | 结果 | 备注 |
|---|------|------|------|
| A1 | 页面加载、Tab | ✅ | 画布/组件/部署 |
| A2 | 组件库 API | ✅ | 2 个组件，镜像正常 |
| A3 | 导入 JSON 两节点+边 | ✅ | 名称 `e2e-retest-chain`，边 `step-1→step-2` |
| A4 | 导出 JSON | ✅ | port 边正确 |
| A5 | 保存 | ❌ | `POST /pipelines` **503** |
| A6 | 部署 | ⬜ | 未测（保存失败） |
| A7 | 画布空态文案 | ✅ | P2-3 |
| A8 | 清空确认 | ⬜ | 未复测 |

### B. `/pipeline` — 组件

| # | 功能 | 结果 |
|---|------|------|
| B1 | 注册表列表 | ✅ |
| B2 | 镜像展示 | ✅ P1-2 |

### C. `/pipeline` — 部署

| # | 功能 | 结果 |
|---|------|------|
| C1 | 标题 | ✅「已保存模板 / 部署历史」 |
| C2 | 模板/历史列表 | ❌ API 503 → 0 条 |

### D. `/workflows`

| # | 功能 | 结果 |
|---|------|------|
| D1 | 页面标题 | ✅「流水线运行」 |
| D2 | 列表 | ❌ `GET /workflows` **503** |

### E. 导航

| # | 功能 | 结果 |
|---|------|------|
| E1 | 流水线 / 流水线运行 | ✅ |
| E2 | Console error | ⚠️ 503 资源加载失败 1 条 |

---

## 3. 保存请求体（契约正确，供后端联调）

`POST /api/v1/pipelines` 在 503 前发出的 body 片段：

```json
"edges": [{ "source": "step-1.output", "target": "step-2.input" }],
"nodes": [
  { "id": "step-1", "inputs": [{"name":"input"}], "outputs": [{"name":"output"}], ... },
  { "id": "step-2", ... }
]
```

---

## 4. 推荐验收流程（K8s 就绪后）

```text
1. GET /pipelines、/workflows → 200（非 503）
2. 组件 Tab：Pass Through 唯一
3. 画布：拖 2 节点 → 连线 → 导出 JSON
4. 保存 → 201
5. 部署 → deploy/template 201
6. 部署 Tab / workflows 列表有记录 → Succeeded
7. 详情 DAG + 日志
8. 模板编辑回画布 → 再部署
```

---

## 5. 产品结论

| 维度 | 评价 |
|------|------|
| **路由注册** | 进步 — 404 已消除 |
| **前端契约/UX** | 明显改善 — 去重、镜像、空态、Tab 标题 |
| **可交付 E2E** | **仍否** — 需 backend-dev 配好 Argo/K8s client |

---

## 附录：相关文档

- [pipeline-frontend-guide.md](./pipeline-frontend-guide.md)
- [pipeline-requirements.md](./pipeline-requirements.md)
