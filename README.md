# 任务管理器 — 支持MCP的数据管理平台

面向AI智能体的专项任务管理系统，提供倒计时/正计时、项目聚合、TODO待办、Markdown 笔记、通知提醒、日程管理等功能。

## 功能特性

- **计时单元管理**：支持时间型（倒计时/正计时）和数值型（目标计数/累计计数）
- **项目管理**：将相关计时单元归属到项目中统一管理
- **TODO待办**：轻量级待办事项管理，支持分组、批量操作，以及 [0,1] 范围的重要程度和困难程度
- **笔记管理**：Markdown 笔记 CRUD、分组、标签与全文关键词搜索；随时保存草稿、恢复继续编辑、全文复制
- **通知提醒**：后台定时扫描，自动生成到期/超期提醒
- **认证方式**：浏览器会话、JWT、API Token，以及面向外部智能体的 OAuth 授权码 + PKCE
- **RESTful API**：完整的 REST API，支持脚本和第三方集成
- **MCP 接入**：支持 AI 智能体通过 MCP + API Token 调用全部后端能力

## 技术栈

| 层面 | 技术 |
|------|------|
| 后端 | Go 1.25+, Gin, GORM, PostgreSQL |
| 前端 | Vue 3, Vuetify 3, Pinia, TypeScript |
| 桌面端 | Python, PySide6 / Qt 6 WebEngine |
| 构建 | Vite 5, Docker |

## 快速开始

### For Ops:

indev: https://cnb.cool/greenshadecapital/dreamreflex/devops-taskmgr
 
OpenSource: https://github.com/phil616/taskmgr

### 开发模式

**后端：**

```bash
docker run -d --name taskmgr-postgres -p 127.0.0.1:5432:5432 \
  -e POSTGRES_USER=taskmgr -e POSTGRES_PASSWORD=taskmgr -e POSTGRES_DB=taskmgr \
  -v taskmgr-postgres-data:/var/lib/postgresql/data postgres:17
cd ops-timer-backend
cp env.example .env
# 将 .env 中 TASK_MANAGER_AUTH_JWT_SECRET 替换为 openssl rand -hex 32 生成的值
go mod tidy
go run ./cmd/server/
```

初始管理员用户名默认 `admin`，密码默认 `Admin@12345`。首次启动前请在 `.env` 中设置 `TASK_MANAGER_INIT_ADMIN_USERNAME` 和 `TASK_MANAGER_INIT_ADMIN_PASSWORD`。仅数据库没有用户时创建账户，修改这些变量不会重置已有账户。

根目录 `.env.example` 与 `ops-timer-backend/env.example` 均为完整、相同的后端模板。`.env` 从**启动命令的当前工作目录**读取；例如在 `ops-timer-backend` 中执行 `go run ./cmd/server` 时，应将 `.env` 放在该目录。优先级为：管理员命令行参数 `-admin-user` / `-admin-pass` > 已导出的环境变量 > `.env` > 默认值。

**前端：**

```bash
cd ops-timer-frontend
npm install
npm run dev
```

前端开发服务器运行在 `http://localhost:5173`，API 请求自动代理到后端 `http://localhost:8080`。

### Qt 桌面端

新增 [Qt 任务工作台](ops-timer-desktop/README.md)，内置 Web 业务界面与桌面专用布局，支持持久保存后端 URL、按后端隔离登录会话、原生文件保存及缩放快捷键。仅需连接 Go 后端，无需部署 Web 站点。

Windows 支持构建 ZIP 免安装版：解压后运行 `TaskmgrDesktop.exe`，随包携带动态 Qt DLL、WebEngine 和 Python 运行时，无需另装依赖。构建命令见桌面端文档。

Web 与 Qt 共用笔记编辑器。编辑器使用本地打包的 CodeMirror 6、highlight.js、Prettier、KaTeX 和 Mermaid，工具栏图标也随应用分发，不依赖外部 CDN。Markdown 的 Enter 续写列表与引用，Tab / Shift+Tab 使用两空格缩进。全文复制保留完整 Markdown 原文。

编辑时点击“保存草稿”（或 Ctrl+S），也可选择“存草稿并关闭”；草稿允许空标题、空正文。草稿保存在后端，连接同一后端的 Web 与 Qt 可以相互恢复，提交成功后移除对应草稿，提交失败则保留。草稿包含在数据备份中。升级此功能需同时更新后端，启动时自动创建草稿表。

```bash
cd ops-timer-desktop
python -m venv .venv
source .venv/bin/activate  # Windows: .venv\Scripts\Activate.ps1
pip install -e .
python build_web.py
python -m taskmgr_desktop
```

### Docker 部署

后端镜像使用 `ops-timer-backend/Dockerfile` 构建，通过 `TASK_MANAGER_DATABASE_DSN` 连接已创建的 PostgreSQL 数据库；启动时自动建表。仅支持 PostgreSQL，不提供 SQLite 迁移或兼容路径。

当前 CNB 流水线执行 `baoctl sync phil616-taskmgr`，并通过 Docker `--env-file /run/baoctl/phil616-taskmgr.env` 注入配置。请在对应配置源中设置 `TASK_MANAGER_DATABASE_DSN` 等后端变量；跨源部署还需设置 `TASK_MANAGER_SERVER_CORS_ORIGINS`。修改仓库中的示例文件不会自动改变容器环境。当前端口映射为 `5003:5003`，实际配置需设置 `TASK_MANAGER_SERVER_PORT=5003`。生产连接串请按数据库服务要求配置 TLS。

## 构建命令

### 本地构建

**后端：构建可执行文件**

```bash
cd ops-timer-backend
go build -o bin/taskmgr-server ./cmd/server
```

构建产物默认输出到 `ops-timer-backend/bin/taskmgr-server`。

**前端：构建生产静态资源**

```bash
cd ops-timer-frontend
npm install
npm run build
```

构建产物默认输出到 `ops-timer-frontend/dist/`。

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
├── ops-timer-desktop/       # Qt 桌面端（内置界面、连接设置与本地代理）
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

```

## API 认证

**外部智能体 OAuth：** 设置 `TASK_MANAGER_OAUTH_ISSUER=https://你的后端域名` 并重启，即可通过浏览器登录确认，授权 MCP/REST 客户端。支持自动发现、动态注册、读写权限、15 分钟 Access Token、刷新轮换和撤销；原有登录方式继续可用。配置和对接步骤见 [OAuth 接入文档](ops-timer-backend/docs/oauth.md)。

**JWT 认证：**

```bash
# 登录获取 Token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@12345"}'

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

后端内置 MCP 服务器（默认 `POST /mcp`），提供 **80 个专用工具**，覆盖全部后端功能。智能体通过一个 MCP 配置即可完全操控所有数据。

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

### 可用工具分类（80 个）

| 模块 | 工具数 | 说明 |
|------|--------|------|
| 认证 | 7 | 登录态资料、密码、API Token、SMTP 测试 |
| 计时单元 | 10 | 完整 CRUD + 状态变更 + 步进/设值 + 日志 + 汇总 |
| 项目 | 7 | 完整 CRUD + 查看项目下的单元 + 项目预算统计 |
| 待办 | 11 | 完整 CRUD + 状态 + 批量操作 + 分组管理（4 个） |
| 笔记 | 10 | Markdown 笔记 CRUD + 全局搜索 + 分组管理（4 个） |
| 通知 | 5 | 列表 + 标记已读 + 全部已读 + 未读数 + 删除 |
| 日程 | 7 | 完整 CRUD + 关联/移除资源 |
| 钱包 | 5 | 完整 CRUD + 详情 |
| 收支分类 | 4 | 完整 CRUD |
| 收支记录 | 5 | 完整 CRUD + 详情 |
| 密钥管理 | 7 | 完整 CRUD + 明文读取 + 审计日志 |
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

所有配置通过 `TASK_MANAGER_*` 环境变量读取，也可将变量写入 `.env` 文件（放在启动命令的当前工作目录），程序启动时自动加载。

环境变量模板：[`ops-timer-backend/env.example`](ops-timer-backend/env.example)。

| 环境变量 | 说明 | 默认值 |
|--------|------|--------|
| `TASK_MANAGER_SERVER_PORT` | 服务端口 | 8080 |
| `TASK_MANAGER_SERVER_CORS_ORIGINS` | 允许跨源访问的前端来源，逗号分隔；同源部署可留空 | 空 |
| `TASK_MANAGER_DATABASE_DSN` | PostgreSQL 连接串 | postgres://taskmgr:taskmgr@localhost:5432/taskmgr?sslmode=disable |
| `TASK_MANAGER_INIT_ADMIN_USERNAME` | 初始管理员用户名，仅首次创建 | admin |
| `TASK_MANAGER_INIT_ADMIN_PASSWORD` | 初始管理员密码，仅首次创建 | Admin@12345（请替换） |
| `TASK_MANAGER_AUTH_JWT_SECRET` | JWT 密钥 | 需修改 |
| `TASK_MANAGER_AUTH_JWT_EXPIRY_HOURS` | JWT 过期时间（小时） | 24 |
| `TASK_MANAGER_OAUTH_ISSUER` | OAuth 外部 HTTPS 来源，无路径；空值不启用 | 空 |
| `TASK_MANAGER_OAUTH_DYNAMIC_REGISTRATION` | 允许智能体自行注册 OAuth 公共客户端 | true |
| `TASK_MANAGER_SCHEDULER_NOTIFICATION_SCAN_INTERVAL` | 通知扫描间隔 | 10m |
| `TASK_MANAGER_MCP_ENABLED` | 是否启用 MCP | true |
| `TASK_MANAGER_MCP_PATH` | MCP 路由路径 | /mcp |
| `TASK_MANAGER_MCP_EXTERNAL_URL` | MCP 对外地址（用于配置端点自动生成 URL） | 自动检测 |

## TODO 评分字段

创建和更新 TODO 的 REST API、MCP 工具均接受 `importance`（重要程度）和 `difficulty`（困难程度），返回值也包含这两个字段。两者为 `[0,1]` 浮点数，0 表示最不重要／最不困难，1 表示最重要／最困难。创建时省略默认为 0.5；更新时省略保持原值，传 0 可重置。原有 `priority` 字段独立保留。Web 和 Qt 桌面端均支持显示和编辑评分。

## 测试

```bash
cd ops-timer-backend
go test ./...
# 使用专用测试数据库；测试在随机 schema 中建表，结束后删除该 schema
TASK_MANAGER_TEST_DATABASE_DSN='postgres://taskmgr:taskmgr@localhost:5432/taskmgr?sslmode=disable' go test ./... -count=1
cd ../ops-timer-frontend
npm ci
npm run build
```

## 浏览器登录与待办坐标图

Web 登录支持“记住我（14 天）”，默认不勾选。勾选后使用固定 14 天有效期的 HttpOnly、SameSite=Strict 会话 Cookie；不勾选则使用浏览器会话 Cookie，服务端最长有效期 24 小时。浏览器的“恢复上次会话”功能可能保留会话 Cookie，公共设备请主动退出。密码与长期令牌不存入 localStorage。会话凭据使用随机 256 位值，数据库仅保存其 SHA-256 摘要。退出会撤销当前会话；修改密码后所有浏览器会话立即失效。旧版 JWT/API Token 客户端仍使用原有认证方式。

生产环境必须通过 HTTPS 访问，`TASK_MANAGER_AUTH_COOKIE_SECURE` 默认为 `true`。本地 HTTP 开发可设为 `false`（`env.example` 已配置）。推荐前端与 `/api` 同源反向代理；跨源且同站部署须在 `TASK_MANAGER_SERVER_CORS_ORIGINS` 中列出精确来源，不支持凭据请求的通配来源或跨站 Cookie。浏览器请求需 `X-Taskmgr-Browser: 1`，后端验证 Origin，防止 CSRF。CNB 默认使用生产 Secure Cookie，无需额外配置。

TODO 坐标图使用 [interact.js](https://interactjs.io/docs/draggable/)。横轴重要程度、纵轴困难程度，范围均为 0～1，中心交点是 (0.5, 0.5)，新建 TODO 默认位于中心。图中显示当前筛选当前页的任务，拖动预览、松手保存，失败恢复原位置。通过选择器选取重叠圆点；方向键每次调整 0.01，Enter 保存，Esc 取消。

浏览器端到端测试（使用专用测试数据库运行后端，管理员密码默认 `BrowserTestPassword123`，可通过 `TASK_MANAGER_E2E_PASSWORD` 指定；后端本地 HTTP 配置 `TASK_MANAGER_AUTH_COOKIE_SECURE=false`）：

```bash
cd ops-timer-frontend
npm ci
npx playwright install chromium
# 后端已在 8080 运行时可直接执行；测试自动启动 5177 端口的 Vite
npm run test:e2e
# 若后端使用其他端口：
TASK_MANAGER_DEV_BACKEND_URL=http://127.0.0.1:18087 npm run test:e2e
```

测试覆盖持久 Cookie 恢复登录、未勾选时的会话 Cookie、退出撤销，以及坐标图拖动保存、键盘微调、失败回退和移动端布局。后端 PostgreSQL 集成测试另覆盖 14 天有效期、Cookie 安全属性、CSRF 拦截、到期失效和改密后所有浏览器会话失效。
