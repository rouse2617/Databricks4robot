# Design — P0-6: PgBouncer 入栈

## Source of Truth

- #[[file:docs/review/data-platform-design.md]] §6.3（PgBouncer 拓扑、配置、约束）

## 变更范围

1. `deploy/local/docker-compose.yml`：新增 `pgbouncer` 服务，挂载 `pgbouncer.ini` + `userlist.txt`
2. `deploy/local/docker-compose.all.yml`：同上
3. `deploy/local/pgbouncer/pgbouncer.ini`（新增）：transaction pool 配置
4. `deploy/local/pgbouncer/userlist.txt`（新增）：PG 用户凭据
5. `backend/.env.example`：`DB_HOST` 默认改为 `pgbouncer`，`DB_PORT` 改为 `6432`

## 约束（§6.3）
- 不能用 session-level prepared statement
- 不能用 `SET LOCAL` 之外的 `SET`
- 不能用临时表跨事务
- 不能用 advisory lock 跨事务
- 当前代码已规避以上所有
