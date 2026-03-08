# Agent Control Plane（Go MVP）

Agent Control Plane 是一个面向 AI Agent 与自动化工具的治理后端。它用于记录工具活动、执行策略决策，并在高风险操作上支持人工审批。

本仓库提供一个可运行的 Go MVP，包含基于 PostgreSQL 的 API、演示种子数据，以及基于 Docker 的本地运行环境。

项目采用 Apache License 2.0 许可证，详见 [LICENSE](./LICENSE)。

## 项目功能

系统当前聚焦四类核心能力：

1. **预执行决策（Preflight）**
   - 在工具动作执行前进行策略评估。
   - 返回 `ALLOW`、`BLOCK` 或 `REQUIRE_APPROVAL`。

2. **后执行审计（Postflight）**
   - 持久化执行结果与时间线事件。
   - 持续更新会话级投影视图。

3. **审批流（Approval Workflow）**
   - 查询审批请求列表。
   - 查看审批详情。
   - 支持携带元数据的通过/拒绝决策。

4. **策略与风险可视化（Policy & Risk）**
   - 查询和管理策略规则。
   - 进行策略预览评估（evaluate preview）。
   - 查看 Dashboard 汇总指标。

## 架构设计

API 使用三层后端架构：

- **HTTP 层**（`apps/api/internal/http`）
  - 路由、请求解析、响应结构、状态码处理。
- **Service 层**（`apps/api/internal/service`）
  - 业务流程编排与领域逻辑。
- **Repository 层**（`apps/api/internal/repo`）
  - PostgreSQL 持久化与查询映射。

`apps/worker` 当前是心跳占位实现，用于后续异步/后台任务扩展。

## 技术栈

- Go（API + Worker）
- Chi Router
- PostgreSQL
- pgx / pgxpool
- Docker Compose（本地依赖）

## 仓库结构

- `apps/api`：主 API 服务
- `apps/worker`：Worker 服务（当前为心跳占位）
- `apps/web`：前端静态页面（MVP 控制台）
- `db/migrations`：数据库迁移脚本
- `db/seed.sql`：演示数据
- `deploy/docker-compose.yml`：本地基础设施
- `apps/api/openapi/openapi.yaml`：API 合同草案

## 快速开始

### 1）启动本地依赖

```bash
docker compose -f deploy/docker-compose.yml up -d
```

### 2）执行数据库迁移

使用你熟悉的 migration 工具，按 `db/migrations` 顺序执行。

### 3）启动 API

```bash
cd apps/api
go run ./cmd/server
```

### 4）启动 Worker

```bash
cd apps/worker
go run ./cmd/worker
```

## 初始化演示数据

迁移完成后，执行：

```bash
psql "postgres://acp:acp@localhost:5432/acp?sslmode=disable" -f db/seed.sql
```

若本机未安装 `psql`，可使用 Docker：

```bash
docker run --rm --network deploy_default -v "$PWD/db:/db" postgres:16 \
  sh -lc 'psql "postgres://acp:acp@postgres:5432/acp?sslmode=disable" -f /db/seed.sql'
```

种子数据包含：

- 20 条 session
- 100 条 tool event
- 4 条 policy rule
- 20 条 approval（含 pending 与 decided）

## API 接口（MVP）

### Gateway

- `POST /api/v1/gateway/preflight`
- `POST /api/v1/gateway/postflight`

### Sessions

- `GET /api/v1/sessions`
- `GET /api/v1/sessions/{id}`
- `GET /api/v1/sessions/{id}/timeline`

### Dashboard

- `GET /api/v1/dashboard/summary`

### Approvals

- `GET /api/v1/approvals`
- `GET /api/v1/approvals/{approval_id}`
- `POST /api/v1/approvals/{approval_id}/decision`

### Policies

- `GET /api/v1/policies`
- `POST /api/v1/policies`
- `PATCH /api/v1/policies/{policy_id}`
- `POST /api/v1/policies/{policy_id}/enable`
- `POST /api/v1/policies/{policy_id}/disable`
- `POST /api/v1/policies/evaluate-preview`

## 验证

运行 API 测试：

```bash
cd apps/api
go test ./...
```

示例请求：

```bash
curl -s http://localhost:8080/api/v1/dashboard/summary
curl -s "http://localhost:8080/api/v1/approvals?status=pending&page=1&page_size=10"
curl -s "http://localhost:8080/api/v1/policies?page=1&page_size=10"
```

## 许可证

本项目采用 Apache License 2.0，详见 [LICENSE](./LICENSE)。
