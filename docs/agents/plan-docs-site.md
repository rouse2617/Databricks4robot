# cyber-databrew 文档站点落地方案

> 日期：2026-05-31
> 状态：方案设计（待评审）
> 目标：建立类似 [Scale Nucleus](https://nucleus.scale.com/docs/getting-started) 的完整开发者文档站点

---

## 目录

1. [工具选择论证](#1-工具选择论证)
2. [部署方案](#2-部署方案)
3. [目录结构](#3-目录结构)
4. [内容规划](#4-内容规划)
5. [搜索方案](#5-搜索方案)
6. [版本化方案](#6-版本化方案)
7. [域名建议](#7-域名建议)
8. [实施步骤](#8-实施步骤)
9. [工作量评估](#9-工作量评估)

---

## 1. 工具选择论证

### 1.1 候选工具对比

| 维度 | **Docusaurus** | VitePress | Astro Starlight | Nextra | MkDocs |
|------|:--------------:|:---------:|:---------------:|:------:|:------:|
| **框架** | React 18 | Vue 3 | Astro (Islands) | Next.js (React) | Python |
| **团队匹配度** | ✅ React 栈，AntD 可复用 | ❌ Vue — 团队不熟 | ⚠️ 类 React 但隔离 | ✅ React | ❌ Python — 风格割裂 |
| **MDX 支持** | ✅ 原生 | ❌ MD (需插件) | ✅ 原生 | ✅ 原生 | ❌ Markdown |
| **版本化** | ✅ 内置 | ✅ 内置 | ✅ 内置 | ⚠️ 需手动 | ✅ 多版本插件 |
| **搜索** | ✅ Algolia / 本地 | ✅ 本地 mini | ✅ Pagefind | ✅ FlexSearch | ✅ 插件生态 |
| **i18n** | ✅ 内置 | ✅ 内置 | ✅ 内置 | ⚠️ 需配置 | ✅ 插件 |
| **API 文档集成** | ✅ OpenAPI 插件丰富 | ⚠️ 有限 | ✅ Starlight OpenAPI | ❌ 弱 | ⚠️ 插件 |
| **主题定制** | ✅ CSS + Swizzle | ⚠️ 有限 | ✅ Tailwind 友好 | ⚠️ 有限 | ✅ 模板 |
| **构建性能** | ⚠️ 中 (1000+ 页慢) | ✅ 快 | ✅ 快 | ⚠️ 中 | ✅ 快 |
| **社区/生态** | ✅ Meta 维护, 大 | ⚠️ 中 | ✅ Astro 团队 | ⚠️ 中 | ✅ 历史久 |
| **CI 集成** | ✅ npm build → 静态 | ✅ | ✅ | ✅ | ✅ Python |

### 1.2 最终推荐：✅ **Docusaurus**

**核心理由：**

1. **React 栈深度匹配** — 团队主力 React 19 + AntD 5，Docusaurus 的组件化、Swizzle 机制、MDX 能力允许用 React 组件扩展文档，甚至嵌入 AntD 组件复现品牌风格
2. **功能全覆盖** — 版本化、搜索、i18n、OpenAPI 集成（`docusaurus-plugin-openapi-docs` 或 `redocusaurus`）均为内置或一等插件，无需拼凑
3. **企业级文档站共识** — Meta 团队维护，React Native、Jest、Docusaurus 自身均用它，长期稳定
4. **与现有前端构建工具链兼容** — 同为 npm 生态，可复用现有 Node 版本（`node:20-alpine`）和 CI 流程模式

**否决其他方案的理由：**

- **VitePress**（Vue）— 团队无 Vue 经验，学习成本高，AntD 主题无法复用
- **Astro Starlight** — 架构不同（Islands），扩展需 Astro 组件知识
- **Nextra** — 依赖 Next.js 全框架，过度设计；内容管理不如 Docusaurus 直观
- **MkDocs**（Python）— 虽然简单但无法嵌入 React 组件，OpenAPI 集成弱，长期维护成本高

---

## 2. 部署方案

### 2.1 方案对比

| 方案 | 延迟 | 成本 | 运维复杂度 | CI 一致性 | 域名灵活性 |
|:----|:----:|:----:|:----------:|:---------:|:----------:|
| A. **独立 Cloud Run 服务** | 低 (中国可能高) | ~$5-15/月 | 低 | ✅ 与现有 CI 一致 | ✅ URL map |
| B. 挂现有前端下 (子路由) | 低 | 0（复用） | ⚠️ 高（路由冲突） | ⚠️ 耦合 | ❌ 受限 |
| C. Cloudflare Pages | ✅ 全球快 | 免费(免费计划) | ⚠️ 多平台 | ❌ 两套 CI | ✅ 自定义域 |

### 2.2 推荐方案：✅ **A. 独立 Cloud Run 服务**

**理由：**

1. **与现有部署模式一致** — 项目已用 Cloud Run + Artifact Registry + Tekton CI，新增一个服务完全复用现有流水线模板（参考 `.tekton/push-frontend-cloudrun-dev.yaml`）
2. **域名独立** — `docs.cyberorigin.ai` 或 `docs.cyberdatabrew.ai` 单独映射，不干扰主前端
3. **资源需求极低** — Docusaurus 输出纯静态文件，Nginx 容器即可服务，CPU 0.5、内存 256Mi 足够
4. **零运行时依赖** — 无数据库、无后端逻辑，静态站点零安全面

### 2.3 资源估算

| 环境 | CPU | 内存 | Min/Max Instances | 预算/月 |
|:----|:---:|:----:|:-----------------:|:-------:|
| **dev** | 0.5 | 256Mi | 0/2 | ~$2-5 |
| **prod** | 1.0 | 512Mi | 0/3 | ~$5-15 |

Docusaurus 构建本身需要更多资源（`node:20-alpine` 中 `npm run build` 约需 512Mi-1Gi），但这是 CI 阶段，不是运行时。构建产物只有纯 HTML/CSS/JS，运行时开销极低。

### 2.4 CI/CD 策略

**分支策略**（沿用现有 Tekton 模式）：

```
release/docs-v1.0 ──► prod: cyber-databrew-docs
main              ──► dev:  cyber-databrew-docs-dev
```

**流水线步骤**（参考 `.tekton/push-frontend-cloudrun-dev.yaml`）:

1. `git-clone` — 拉取仓库
2. `build-docs` — `cd docs-site && npm ci && npm run build`
3. `build-and-push` — BuildKit 构建 Docker 镜像 → Artifact Registry
4. `deploy-cloudrun` — `gcloud run deploy`

---

## 3. 目录结构

### 3.1 推荐方案：独立目录 `docs-site/`（monorepo 内）

```
cyber-databrew/
├── docs-site/                          # ★ 文档站点独立目录
│   ├── docusaurus.config.ts            # Docusaurus 配置（品牌、搜索、插件）
│   ├── sidebars.ts                     # 侧边栏结构
│   ├── package.json                    # 独立构建依赖
│   ├── tsconfig.json
│   ├── Dockerfile                      # Cloud Run nginx 镜像
│   ├── nginx.conf                      # Cloud Run nginx 模板
│   ├── static/                         # 静态资源（favicon, logos, images）
│   │   ├── img/
│   │   │   ├── logo.svg
│   │   │   └── favicon.ico
│   │   └── redirects/
│   ├── src/
│   │   ├── css/
│   │   │   └── custom.css              # AntD 色板 + brand 定制
│   │   ├── components/                 # 自定义 React 组件
│   │   │   ├── OpenApiSpec.tsx         # 内嵌 OpenAPI 渲染
│   │   │   ├── AntDThemeAdapter.tsx    # 复用 AntD token 变量
│   │   │   └── SDKInstallGuide.tsx     # 动态版本 SDK 安装命令
│   │   └── theme/                      # Swizzle 覆盖
│   └── docs/                           # ★ 文档内容（Markdown / MDX）
│       ├── intro.md                    # 首页/欢迎页
│       ├── getting-started/
│       │   ├── overview.md
│       │   ├── quickstart.md
│       │   ├── authentication.md
│       │   └── client-libraries.md
│       ├── guides/
│       │   ├── assets/
│       │   ├── search/
│       │   ├── deliveries/
│       │   ├── lakehouse/
│       │   ├── algorithms/
│       │   └── pipeline/
│       ├── api-reference/
│       │   └── openapi/                # ★ OpenAPI 自动生成
│       ├── sdk/
│       │   ├── installation.md
│       │   ├── quickstart.md
│       │   ├── client.md
│       │   └── reference/
│       ├── tutorials/
│       │   ├── end-to-end-pipeline.md
│       │   └── custom-algorithm.md
│       ├── concepts/
│       │   ├── data-model.md
│       │   ├── asset-lifecycle.md
│       │   └── outbox-cdc.md
│       └── deployment/
│           ├── architecture.md
│           ├── local-development.md
│           └── production-setup.md
├── api/
│   └── openapi.yaml                    # ★ 源 OpenAPI 规范（docs-site 构建时引用）
├── sdk/
│   ├── README.md                       # ★ 源 SDK 文档（docs-site 构建时嵌入/引用）
│   └── pyproject.toml                  # SDK 版本信息
├── docs/                               # 现有内部文档（不变）
│   ├── agents/
│   └── review/
└── ...
```

### 3.2 为何不单独建 repo

| 方案 | 优点 | 缺点 |
|:----|:----|:-----|
| **monorepo 内 `docs-site/`** | ✅ OpenAPI/sdk/README 可原位引用；PR 统一；CI 复用 | 仓库较大 |
| 独立 repo | 分权限管理 | ❌ OpenAPI/sdk 需同步；两套 CI；PR 割裂 |

推荐 monorepo 内，但文档团队可只关注 `docs-site/docs/` 目录。

> **注意：** Docusaurus 的 `docs/` 目录与项目现有的 `docs/` 冲突。解决方案有二：
> - (推荐) 使用 Docusaurus 的 `path` 选项将内容目录指向 `docs-site/docs/`
> - 或将项目现有 `docs/` 重命名为 `docs-internal/`（不推荐，增加迁移成本）

---

## 4. 内容规划

### 4.1 第一期（MVP）页面清单

| # | 页面 | 路径 | 内容来源 | 优先级 |
|:-:|:----|:----|:---------|:------:|
| 1 | **首页 / 欢迎** | `/` | 新建 | P0 |
| 2 | **快速开始** | `/getting-started/quickstart` | 从 `docs/review/api-guide.md` + 现有 README 提炼 | P0 |
| 3 | **认证** | `/getting-started/authentication` | 从 `docs/review/api-guide.md` + `docs/review/data-platform-design.md` 提炼 | P0 |
| 4 | **客户端库 (SDK)** | `/getting-started/client-libraries` | 从 `sdk/README.md` 提炼 | P0 |
| 5 | **API 参考** | `/api-reference/` | 从 `api/openapi.yaml` 自动生成（Redoc 或 OpenAPI 插件） | P0 |
| 6 | **资产管理** | `/guides/assets/` | 从 `docs/review/asset-model-expansion.md` + `docs/review/api-guide.md` 提炼 | P0 |
| 7 | **搜索** | `/guides/search/` | 从 `docs/review/unified-search-enhancement.md` + `docs/review/api-guide.md` 提炼 | P0 |
| 8 | **交付管理** | `/guides/deliveries/` | 从 `docs/review/api-guide.md` + `docs/review/data-platform-design.md` 提炼 | P0 |
| 9 | **SDK 安装指南** | `/sdk/installation` | 从 `sdk/README.md` 纳入 | P0 |
| 10 | **SDK 快速开始** | `/sdk/quickstart` | 从 `sdk/README.md` 提炼 | P0 |
| 11 | **数据模型概览** | `/concepts/data-model` | 从 `docs/review/schema-reference.md` 提炼 | P1 |
| 12 | **架构概览** | `/deployment/architecture` | 从 `docs/review/data-platform-design.md` 提炼 | P1 |
| 13 | **本地开发** | `/deployment/local-development` | 从根 `README.md` + `deploy/local/` 文档提炼 | P1 |
| 14 | **端到端流水线教程** | `/tutorials/end-to-end-pipeline` | 从 `docs/review/pipeline-frontend-guide.md` + `docs/review/data-production-line.md` 提炼 | P1 |

### 4.2 第二期（增强）

| # | 页面 | 路径 |
|:-:|:----|:----|
| 1 | 湖仓查询 | `/guides/lakehouse/` |
| 2 | 算法运行管理 | `/guides/algorithms/` |
| 3 | 流水线编排 | `/guides/pipeline/` |
| 4 | 审计与血缘 | `/guides/audit/` |
| 5 | 算法工程师教程 | `/tutorials/custom-algorithm` |
| 6 | 部署至生产 | `/deployment/production-setup` |
| 7 | Outbox CDC 概念 | `/concepts/outbox-cdc` |
| 8 | 资产生命周期概念 | `/concepts/asset-lifecycle` |
| 9 | MCAP 预览服务 | `/guides/mcap-preview/` |
| 10 | 交付规则 | `/guides/delivery-rules/` |

### 4.3 内容策略

- **直接引用源文档**：`sdk/README.md` 通过 Docusaurus MDX `import` 或 symlink 嵌入
- **API 参考自动生成**：使用 `redocusaurus`（Redoc 渲染器）或 `docusaurus-plugin-openapi-docs` 从 `api/openapi.yaml` 生成
- **代码示例**：从 `sdk/` 和 `scripts/` 提取真实代码片段，用 MDX `CodeBlock` 组件
- **设计文档转化**：`docs/review/` 下的设计文档经裁剪、重写后发布为新版指南

---

## 5. 搜索方案

| 方案 | 集成难度 | 成本 | 搜索质量 | 维护成本 | 推荐 |
|:----|:--------:|:----:|:--------:|:--------:|:----:|
| **Algolia DocSearch** | 低（申请即可） | 免费 （OSS） | ✅ 极好 | 零维护 | ✅ |
| Docusaurus 内置搜索 | 零配置 | 免费 | ⚠️ 本地查询 | 零维护 | 备选 |
| MeiliSearch 自建 | 高（需部署） | ~$20-50/月 | ✅ 好 | 高 | ❌ |
| Typesense | 中 | ~$10-20/月 | ✅ 好 | 中 | ❌ |

### 推荐：✅ Algolia DocSearch 免费计划 + 内置搜索兜底

**配置方式：** Docusaurus `docusaurus.config.ts` 中设置 `algolia` 字段，只需 appId / apiKey / indexName。DocSearch 爬虫自动运行，无须自建。

**兜底：** 若 Algolia 申请不通过，启用内置 `@easyops-cn/docusaurus-search-local` 插件（客户端本地搜索，无需外部服务）。

---

## 6. 版本化方案

### 6.1 Git tag → Docusaurus 版本映射

```
v0.1.0  ──► docs-site/docusaurus.config.ts version v0.1
v0.2.0  ──► docs-site/docusaurus.config.ts version v0.2  (current)
v1.0.0  ──► docs-site/docusaurus.config.ts version v1.0
```

### 6.2 工作机制

1. **发布新版本时**：`npm run docusaurus docs:version v0.2.0`
   - 生成 `docs-site/versioned_docs/version-v0.2.0/` + `docs-site/versioned_sidebars/version-v0.2.0/`
2. **默认显示 latest**：`docs-site/docs/` 永远是最新开发版
3. **Git tag 绑定**：每次 release 打 tag 时，同时 commit 版本化快照
4. **多版本切换**：用户可在文档站点下拉菜单中切换 v0.1/v0.2/latest

### 6.3 注意

- SDK 版本与文档版本在 `pyproject.toml` 中用 `tag_regex = "^sdk/v(?P<version>.*)$"` 独立标记
- API 版本在 `api/openapi.yaml` 中 `info.version` 独立控制
- 文档站点以产品大版本（v0.1, v0.2, v1.0）为粒度，而非 SDK 或 API 的补丁版本

---

## 7. 域名建议

### 推荐：✅ `docs.cyberorigin.ai`

**理由：**

- 项目现有域名基座是 `cyberorigin.ai`（`cyber-databrew-dev.cyberorigin.ai`、`api-cyber-databrew-dev.cyberorigin.ai`）
- `docs.cyberorigin.ai` 一致性强，DNS 管理和 SSL 证书沿用现有体系
- Cloud Run 自定义域名映射 + Google-managed SSL 即可

**备选：** `docs.cyberdatabrew.ai`（若团队计划独立建品牌）

**Cloud Run 域名映射步骤：**

```
# 1. 验证域名所有权
gcloud domains verify docs.cyberorigin.ai

# 2. 映射自定义域名到 Cloud Run
gcloud beta run domain-mappings create \
  --service cyber-databrew-docs \
  --region us-central1 \
  --domain docs.cyberorigin.ai

# 3. 配置 DNS CNAME（输出中会提示）
# docs.cyberorigin.ai CNAME ghs.googlehosted.com.
```

或更简单的方案 — 使用 `ghs.googlehosted.com` 的 Cloud Run 默认映射。

---

## 8. 实施步骤

### 8.1 第一期实施步骤

#### Step 1: 初始化 Docusaurus 项目（~0.5 天）

```bash
cd /Users/rick/cyber-databrew
npx create-docusaurus@latest docs-site classic --typescript
```

添加核心依赖：

```bash
cd docs-site
npm install redocusaurus                   # OpenAPI 渲染
npm install @easyops-cn/docusaurus-search-local  # 本地搜索（兜底）
```

#### Step 2: 配置 docusaurus.config.ts（~0.5 天）

- 站点元数据：title、tagline、url（`https://docs.cyberorigin.ai`）
- 主题配置：基于 AntD 5 色板定制 CSS 变量（`@ant-design/colors` 主题色）
- 插件配置：Redocusaurus、搜索、sitemap
- 导航栏：左侧产品文档、API 参考、SDK、教程
- 脚注：项目 GitHub、Linear、Feishu 入口

#### Step 3: 主题定制 — AntD 风格适配（~1 天）

Docusaurus 的 Infima CSS 可用 AntD 5 色板覆盖：

```css
/* src/css/custom.css */
:root {
  --ifm-color-primary: #1677ff;          /* AntD 蓝色主色 */
  --ifm-color-primary-dark: #0958d9;
  --ifm-color-primary-darker: #003eb3;
  --ifm-color-primary-light: #4096ff;
  --ifm-color-primary-lighter: #69b1ff;
  --ifm-color-secondary: #f5f5f5;        /* AntD 灰色背景 */
  --ifm-border-radius: 6px;              /* AntD 圆角 */
  --ifm-font-family-base: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto;
  --ifm-heading-font-weight: 600;
  --ifm-code-font-size: 0.875em;
  --ifm-code-border-radius: 4px;
}
```

#### Step 4: 配置侧边栏结构（~0.5 天）

`sidebars.ts` — 对应第 4 节内容规划的四级分类。

#### Step 5: 编写第一期内容（~5-7 天）

| 内容 | 工作量 | 来源 |
|:----|:-----:|:----|
| 首页 / 欢迎 | 0.5 天 | 新建 |
| 快速开始 + 认证 | 1 天 | 从 `docs/review/api-guide.md` + Web UI 截图提炼 |
| 客户端库 / SDK 安装 | 0.5 天 | 从 `sdk/README.md` 提炼 |
| 资产管理指南 | 1 天 | 合并 `docs/review/asset-model-expansion.md` + `api-guide.md` |
| 搜索指南 | 0.5 天 | 从 `docs/review/unified-search-enhancement.md` + `api-guide.md` |
| 交付指南 | 0.5 天 | 从 `docs/review/api-guide.md` + `data-platform-design.md` |
| SDK 文档 | 1 天 | 从 `sdk/README.md` 转化 |
| 数据模型 / 架构 | 1 天 | 从 `docs/review/data-platform-design.md` + `schema-reference.md` |
| API 参考集成 | 0.5 天 | Redocusaurus 配置，从 `api/openapi.yaml` 自动生成 |
| 端到端教程 | 1 天 | 从 `docs/review/pipeline-frontend-guide.md` + 操作截图 |
| 本地开发指南 | 0.5 天 | 从根 `README.md` + `Makefile` 提炼 |

#### Step 6: 创建 Dockerfile + nginx.conf（~0.5 天）

```dockerfile
# docs-site/Dockerfile
# Stage 1: Build
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Stage 2: Serve
FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
# No proxy needed — purely static
EXPOSE 80
```

```nginx
# docs-site/nginx.conf
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets aggressively
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

#### Step 7: 创建 Cloud Run 部署脚本（~0.5 天）

```bash
# scripts/deploy-docs-dev.sh
#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="green-valley-442103"
REGION="us-central1"
SERVICE="cyber-databrew-docs-dev"
IMAGE_TAG="${1:-$(git rev-parse --short HEAD)}"
IMAGE="us-central1-docker.pkg.dev/${PROJECT_ID}/video-proc-images/cyber-databrew-docs:${IMAGE_TAG}"

# Build
docker build -t "${IMAGE}" -f docs-site/Dockerfile docs-site/

# Push to Artifact Registry
docker push "${IMAGE}"

# Deploy to Cloud Run
gcloud run deploy "${SERVICE}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --platform managed \
  --image "${IMAGE}" \
  --port 80 \
  --allow-unauthenticated \
  --min-instances 0 \
  --max-instances 2 \
  --cpu 0.5 \
  --memory 256Mi \
  --timeout 30

echo "Deployed: https://${SERVICE}-wtttm6suaq-uc.a.run.app"
```

#### Step 8: 创建 Tekton CI 流水线（~0.5 天）

参考 `.tekton/push-frontend-cloudrun-dev.yaml`，新增 `.tekton/push-docs-cloudrun-dev.yaml` 和 `push-docs-cloudrun-prod.yaml`，内容类似但指向 `docs-site/Dockerfile`。

#### Step 9: 域名 & DNS 配置（~0.5 天）

映射 `docs.cyberorigin.ai` 到 Cloud Run 服务，配置 DNS CNAME 记录。

#### Step 10: 验证 & 上线（~0.5 天）

- 本地 `npm run start` 验证所有页面
- 构建 `npm run build` 验证无错误
- 部署到 dev 环境验证
- 部署到 prod

### 8.2 第一期总工期：**~10-12 人天**

---

## 9. 工作量评估

### 9.1 时间线建议

```
Week 1:
  Mon-Tue: 初始化 + 配置 + 主题定制（~2 天）
  Wed-Fri: 核心内容编写（快速开始、SDK、资产管理）（~3 天）

Week 2:
  Mon-Wed: 剩余内容 + API 集成 + 教程（~3 天）
  Thu:     Dockerfile + CI + 域名（~1 天）
  Fri:     部署验证 + 上线 + 文档（~1 天）
```

### 9.2 分角色工作量

| 角色 | 工作量 | 主要任务 |
|:----|:-----:|:--------|
| **前端/文档工程师 (1人)** | ~10 天 | 初始化、主题、内容编写、Dockerfile |
| **DevOps/后端 (支持)** | ~1 天 | CI 流水线模板、域名配置、Cloud Run 服务创建 |
| **技术评审** | ~0.5 天 | 内容审查、架构合理性 |

### 9.3 持续维护成本

| 活动 | 频率 | 耗时 |
|:----|:----:|:----:|
| 版本发布（切文档版本） | 每发布一次 | ~0.5 天 |
| 内容更新（API 变更） | 随 feature 流程 | ~0.25 天/次 |
| 新增指南 | 按需 | ~1-2 天/篇 |
| 搜索优化 | 季度 | ~0.5 天 |

### 9.4 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:----|:----:|:----:|:---------|
| OpenAPI 5K 行 -> 渲染性能 | 中 | 低 | Redocusaurus 分片加载，或只渲染 public 端点 |
| Docusaurus 与现有 docs/ 目录冲突 | 高 | 低 | 使用 `docs-site/docs/` 路径，通过 `docusaurus.config.ts` 的 `path` 选项指定 |
| 版本化与现有 git tag 不匹配 | 低 | 中 | 首次文档发布时创建 `docs-v0.1.0` tag |
| 中国用户访问 Cloud Run 慢 | 中 | 中 | 若团队有中国成员，将来增加 Cloudflare 前置或 GCP CDN |
| 文档内容过期 | 高 | 高 | 在 OpenSpec 流程中增加文档更新 checklist 项 |

---

## 附录

### A. 相关文件清单

| 类型 | 路径 |
|:----|:-----|
| SDK README | `sdk/README.md` |
| SDK 版本 | `sdk/pyproject.toml` |
| OpenAPI 规范 | `api/openapi.yaml` |
| 设计文档 | `docs/review/data-platform-design.md` |
| API 指南 | `docs/review/api-guide.md` |
| Schema 参考 | `docs/review/schema-reference.md` |
| Pipeline 指南 | `docs/review/pipeline-frontend-guide.md` |
| 流水线设计 | `docs/review/data-production-line.md` |
| 部署配置 | `deploy/cloudrun/frontend-dev.sh` |

### B. Cloud Run 服务创建命令（仅首次）

```bash
# 创建 dev 服务
gcloud run deploy cyber-databrew-docs-dev \
  --project green-valley-442103 \
  --region us-central1 \
  --platform managed \
  --image us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/cyber-databrew-docs:init \
  --port 80 \
  --allow-unauthenticated \
  --min-instances 0 \
  --max-instances 2 \
  --cpu 0.5 \
  --memory 256Mi \
  --no-cpu-throttling \
  --concurrency 80 \
  --timeout 30

# 创建 prod 服务（正式上线时）
gcloud run deploy cyber-databrew-docs \
  --project green-valley-442103 \
  --region us-central1 \
  --platform managed \
  --image us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/cyber-databrew-docs:v1.0.0 \
  --port 80 \
  --allow-unauthenticated \
  --min-instances 0 \
  --max-instances 3 \
  --cpu 1 \
  --memory 512Mi \
  --no-cpu-throttling \
  --concurrency 80 \
  --timeout 30
```

### C. 设计参考

- **Scale Nucleus**: https://nucleus.scale.com/docs/getting-started
- **Docusaurus 官方**: https://docusaurus.io/
- **Redocusaurus**: https://github.com/rohit-gohri/redocusaurus
- **Our blog posts**: https://ant.design/docs/react/introduce (AntD 设计语言参考)
