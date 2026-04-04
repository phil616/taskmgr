# 任务管理器 — 支持MCP的数据管理平台

面向AI智能体的专项任务管理系统，提供倒计时/正计时、项目聚合、TODO待办、通知提醒、日程管理等功能。

## 功能特性

- **计时单元管理**：支持时间型（倒计时/正计时）和数值型（目标计数/累计计数）
- **项目管理**：将相关计时单元归属到项目中统一管理
- **TODO待办**：轻量级待办事项管理，支持分组和批量操作
- **通知提醒**：后台定时扫描，自动生成到期/超期提醒
- **认证方式**：JWT + API Token 双认证方式
- **RESTful API**：完整的 REST API，支持脚本和第三方集成
- **MCP 接入**：支持 AI 智能体通过 MCP + API Token 调用全部后端能力

## 技术栈

| 层面 | 技术 |
|------|------|
| 后端 | Go 1.22+, Gin, GORM, SQLite |
| 前端 | Vue 3, Vuetify 3, Pinia, TypeScript |
| 构建 | Vite 5, Docker |

## 快速开始

### 开发模式

**后端：**

```bash
cd ops-timer-backend
go mod tidy
go run ./cmd/server/
```

默认管理员账户：`admin` / `admin123`

**前端：**

```bash
cd ops-timer-frontend
npm install
npm run dev
```

前端开发服务器运行在 `http://localhost:5173`，API 请求自动代理到后端 `http://localhost:8080`。

### Docker 部署

```bash
docker compose up -d
```

访问 `http://localhost` 即可使用。

## 项目结构

```
cd-v2/
├── ops-timer-backend/       # Go 后端
│   ├── cmd/server/          # 程序入口
│   ├── internal/
│   │   ├── api/             # HTTP 处理层
│   │   ├── service/         # 业务逻辑层
│   │   ├── repository/      # 数据访问层
│   │   ├── model/           # 数据模型
│   │   ├── dto/             # 数据传输对象
│   │   ├── config/          # 配置管理
│   │   └── pkg/             # 工具包
│   ├── env.example           # 环境变量模板
│   └── Dockerfile
├── ops-timer-frontend/      # Vue 3 前端
│   ├── src/
│   │   ├── api/             # API 请求层
│   │   ├── components/      # 通用组件
│   │   ├── views/           # 页面视图
│   │   ├── stores/          # Pinia 状态管理
│   │   ├── router/          # 路由配置
│   │   ├── types/           # TypeScript 类型
│   │   └── utils/           # 工具函数
│   └── Dockerfile
└── docker-compose.yml
```

## API 认证

**JWT 认证：**

```bash
# 登录获取 Token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 使用 Token
curl http://localhost:8080/api/v1/units \
  -H "Authorization: Bearer <token>"
```

**API Token 认证：**

```bash
curl http://localhost:8080/api/v1/units \
  -H "X-API-Token: <your_api_token>"
```

## MCP 接入（AI 智能体）

后端内置 MCP 服务器（默认 `POST /mcp`），提供 **55 个专用工具**，覆盖全部后端功能。智能体通过一个 MCP 配置即可完全操控所有数据。

### 一键获取配置

访问 `GET /mcp/config` 即可获取可直接粘贴到 MCP 客户端的 JSON 配置片段：

```bash
curl http://localhost:8080/mcp/config
```

返回示例：

```json
{
  "mcpServers": {
    "ops-timer-mcp": {
      "url": "https://your-domain.com/mcp",
      "headers": {
        "X-API-Token": "<your-api-token>"
      }
    }
  }
}
```

将 `mcpServers` 部分粘贴到你的 MCP 客户端配置中（Cursor Settings → MCP、Claude Desktop `mcp_config.json` 等），替换 `<your-api-token>` 为你的真实 Token 即可。

### 可用工具分类（55 个）

| 模块 | 工具数 | 说明 |
|------|--------|------|
| 计时单元 | 10 | 完整 CRUD + 状态变更 + 步进/设值 + 日志 + 汇总 |
| 项目 | 6 | 完整 CRUD + 查看项目下的单元 |
| 待办 | 11 | 完整 CRUD + 状态 + 批量操作 + 分组管理（4 个） |
| 通知 | 5 | 列表 + 标记已读 + 全部已读 + 未读数 + 删除 |
| 日程 | 7 | 完整 CRUD + 关联/移除资源 |
| 钱包 | 5 | 完整 CRUD + 详情 |
| 收支分类 | 4 | 完整 CRUD |
| 收支记录 | 5 | 完整 CRUD + 详情 |
| 预算统计 | 1 | 汇总（按钱包/日期范围） |
| 通用代理 | 1 | `backend_request`（任意 HTTP 调用兜底） |

### 调用示例

```bash
# 使用专用工具创建待办
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Token: <your_api_token>" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "todo_create",
      "arguments": {
        "title": "部署生产环境",
        "priority": "high",
        "due_date": "2026-04-10"
      }
    }
  }'

# 使用通用工具（兜底）
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Token: <your_api_token>" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "backend_request",
      "arguments": {
        "method": "GET",
        "path": "/api/v1/projects"
      }
    }
  }'
```

## API 文档（OpenAPI）

后端 HTTP 接口的 **OpenAPI 3.0** 规范位于：

- [`ops-timer-backend/docs/openapi.yaml`](ops-timer-backend/docs/openapi.yaml) — 路径、请求/响应模型、认证（JWT / `X-API-Token`）、错误码
- [`ops-timer-backend/docs/README.md`](ops-timer-backend/docs/README.md) — 导入 Postman、Swagger UI、代码生成说明

第三方与自动化工具可直接导入该文件进行对接。

## 配置说明

所有配置通过 `TASK_MANAGER_*` 环境变量读取，也可将变量写入 `.env` 文件（放在可执行文件同目录），程序启动时自动加载。

环境变量模板：[`ops-timer-backend/env.example`](ops-timer-backend/env.example)。

| 环境变量 | 说明 | 默认值 |
|--------|------|--------|
| `TASK_MANAGER_SERVER_PORT` | 服务端口 | 8080 |
| `TASK_MANAGER_DATABASE_DRIVER` | 数据库驱动 | sqlite |
| `TASK_MANAGER_DATABASE_DSN` | 数据库连接 | ./data/task_manager.db |
| `TASK_MANAGER_AUTH_JWT_SECRET` | JWT 密钥 | 需修改 |
| `TASK_MANAGER_AUTH_JWT_EXPIRY_HOURS` | JWT 过期时间（小时） | 24 |
| `TASK_MANAGER_SCHEDULER_NOTIFICATION_SCAN_INTERVAL` | 通知扫描间隔 | 10m |
| `TASK_MANAGER_MCP_ENABLED` | 是否启用 MCP | true |
| `TASK_MANAGER_MCP_PATH` | MCP 路由路径 | /mcp |
| `TASK_MANAGER_MCP_EXTERNAL_URL` | MCP 对外地址（用于配置端点自动生成 URL） | 自动检测 |
