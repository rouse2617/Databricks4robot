# Tasks — P0-6: PgBouncer 入栈

> Source of truth: #[[file:docs/review/data-platform-design.md]] §6.3

## Milestone 1: 本地 Docker Compose

- [ ] 1.1 新增 `deploy/local/pgbouncer/pgbouncer.ini`：transaction pool，`max_client_conn=1000`，`default_pool_size=25`，`reserve_pool=10`
- [ ] 1.2 新增 `deploy/local/pgbouncer/userlist.txt`
- [ ] 1.3 `deploy/local/docker-compose.yml` 新增 `pgbouncer` 服务（端口 6432）
- [ ] 1.4 `deploy/local/docker-compose.all.yml` 同步
- [ ] 1.5 `backend/.env.example` 更新 `DB_HOST=pgbouncer`、`DB_PORT=6432`

## Milestone 2: 验证

- [ ] 2.1 `make dev-up` 启动后 backend 通过 PgBouncer 连接 PG 正常
- [ ] 2.2 `go test ./...` 全过（不依赖 PgBouncer）
- [ ] 2.3 手动跑 24h 无连接异常
