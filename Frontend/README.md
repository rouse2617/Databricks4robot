# Frontend

React 前端工程，用于资产检索、详情浏览、交付视图等。

## Tech Stack

- React 19
- TypeScript
- Vite
- Ant Design
- TailwindCSS

## Scripts

```bash
npm install
npm run dev
npm run build
npm run lint
```

## Environment

复制并配置：

```bash
cp .env.example .env
```

关键变量：

- `VITE_API_BASE_URL`：后端地址（默认 `http://localhost:8080`）

## Directory Layout

- `src/pages/`：页面
- `src/components/`：复用组件
- `src/api/`：接口封装
- `src/hooks/`：业务 hooks
- `src/types/`：类型定义

## Development Notes

- 认证使用本地 `GRACE_TOKEN`（Phase 0 占位方案）
- 当前页面以资产列表/详情为主
- 后续可逐步补充交付管理、索引查询、报表视图

