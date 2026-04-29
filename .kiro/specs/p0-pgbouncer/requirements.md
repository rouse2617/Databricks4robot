# Requirements — P0-6: PgBouncer 入栈

## Introduction

在 Backend 与 PostgreSQL 之间加入 PgBouncer 连接池代理，收敛横扩后的连接数。

Source of truth:
- #[[file:docs/review/data-platform-design.md]] §6.3（PgBouncer 数据库接入层）
- #[[file:docs/review/next-steps-tasks.md]] P0-6

## Requirements

### R1: Docker Compose 加 PgBouncer 服务
- `deploy/local/docker-compose.yml` 和 `docker-compose.all.yml` 新增 PgBouncer 容器
- Transaction pooling 模式，`max_client_conn=1000`、`default_pool_size=25`、`reserve_pool=10`

### R2: Backend 改指向 PgBouncer
- `DB_HOST` / `DB_PORT` 指向 PgBouncer（6432），业务代码 0 改动
- 本地 + 预发跑 24h 无连接异常

### R3: K8s 部署清单
- PgBouncer 独立 Deployment + Service（2 副本）
