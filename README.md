# Agent Control Plane

Agent Control Plane 是一个 **Gateway 型控制平面**，用于 **观测和治理外部现成 agent 的行为**。

它不假设你能改造对方 agent，也不要求接入 SDK。它的核心目标是：**让 agent 的行为不再黑盒**。

当前仓库提供一个可运行的 MVP，包含：
- Go API Gateway
- PostgreSQL 持久化
- React 观察台
- 本地 Docker 依赖与演示数据

项目采用 Apache License 2.0，详见 [LICENSE](./LICENSE)。

## 项目定位

这个项目关注的是：

- 已经存在的 agent / automation / tool runner
- 这些 agent 对外部系统发起的动作
- 在动作执行前后的策略判断、审批和审计
- 基于 session / timeline / event 的可追踪与可观测能力

这不是一个“嵌入到 agent 内部”的 SDK 平台，而是一个 **位于 agent 与目标系统之间的 gateway control plane**。

## 当前 MVP 能力

### 1. Gateway 预执行决策（Preflight）

在动作真正执行前，接收一次 preflight 请求并进行策略判断，返回：

- `ALLOW`
- `BLOCK`
- `REQUIRE_APPROVAL`

当前输入语义已经收敛为明确的 tool/action 模型，例如：

- `agent_id`
- `environment`
- `tool`
- `action`
- `resource`
- `input_summary`
- `command`

### 2. Gateway 后执行记录（Postflight）

动作完成后，接收 postflight 请求并记录执行结果，沉淀为行为事件与会话投影。

当前事件语义包括：

- `tool_requested`
- `policy_blocked`
- `approval_requested`
- `tool_completed`
- `tool_failed`

### 3. Session / Timeline / Event 观测

系统围绕 session 构建观察视图，支持：

- 查看 session 列表
- 查看单个 session 的概览信息
- 查看按时间排序的行为时间线
- 查看单个事件的详细上下文

这使你可以追踪：

- 是哪个 agent 发起了动作
- 动作发生在什么环境
- 对哪个资源进行了什么 action
- 为什么被放行、阻断或要求审批
- 最终执行结果是什么

### 4. 审批流（Approvals）

当高风险动作命中策略时，系统会生成审批请求。

当前支持：

- 查询待审批列表
- 执行 approve / reject
- 将审批与对应行为事件关联

### 5. 策略治理（Policies）

当前支持：

- 查询策略
- 启用 / 禁用策略
- 查看策略命中带来的决策结果

### 6. React 观察台

前端已切换为 React，实现了一个面向观测的控制台：

- Dashboard Summary
- Sessions 列表
- Observability Trace
- Event Detail
- Pending Approvals
- Policies

页面定位已经统一为：

> Gateway observability console for external agent behavior

## 架构设计

当前后端使用三层结构：

- **HTTP 层**（`apps/api/internal/http`）
  - 路由、请求解析、响应结构、状态码处理
- **Service 层**（`apps/api/internal/service`）
  - gateway 决策逻辑、事件语义、审批编排、session 投影更新
- **Repository 层**（`apps/api/internal/repo`）
  - PostgreSQL 持久化与查询映射

前端与后端已经前后端分离：

- `apps/api`：Go API
- `apps/web`：React + Vite 观察台

API 提供 CORS，前端通过 API Base 直接连接后端。

## 技术栈

- Go
- Chi Router
- PostgreSQL
- pgx / pgxpool
- React
- Vite
- Docker Compose

## 仓库结构

- `apps/api`：Gateway API 服务
- `apps/web`：React 观察台
- `apps/worker`：Worker 服务（当前为占位）
- `db/migrations`：数据库迁移脚本
- `db/seed.sql`：演示数据
- `deploy/docker-compose.yml`：本地基础设施
- `apps/api/openapi/openapi.yaml`：OpenAPI 合同草案

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

默认健康检查：

```bash
curl http://localhost:8080/healthz
```

### 4）启动前端观察台

```bash
cd apps/web
npm install
npm run dev
```

默认前端地址：

- `http://localhost:5173`

默认 API 地址：

- `http://localhost:8080/api/v1`

### 5）启动 Worker（可选）

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

- session
- tool event
- policy rule
- approval

可用于直接查看观察台效果。

## API（MVP）

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

OpenAPI 草案见：

- `apps/api/openapi/openapi.yaml`

## 验证

### 后端测试

```bash
cd apps/api
go test ./... -count=1
```

### 前端构建

```bash
cd apps/web
npm run build
```

### 示例请求

```bash
curl -s http://localhost:8080/api/v1/dashboard/summary
curl -s http://localhost:8080/api/v1/sessions
curl -s "http://localhost:8080/api/v1/sessions/sess_1/timeline?limit=10&offset=0"
curl -s "http://localhost:8080/api/v1/approvals?status=pending&page=1&page_size=10"
curl -s "http://localhost:8080/api/v1/policies?page=1&page_size=10"
```

## 当前阶段边界

当前版本优先完成的是：

- 外部 agent 行为的 gateway 化接入语义
- preflight / postflight 决策与记录
- session / timeline / event 观察台
- policy / approval 基础治理能力

当前还 **不是**：

- 通用 agent SDK 平台
- 完整的 agent 编排系统
- 全量 MCP 代理层
- 面向所有协议的一体化接入网关

MVP 目标很明确：先把 **“外部 agent 行为可观测、可追踪、可治理”** 这件事做扎实。

## 许可证

本项目采用 Apache License 2.0，详见 [LICENSE](./LICENSE)。
