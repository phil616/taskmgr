# 更新日志

## v1.2.0 — 2026-10-27

### MCP 标准化与可靠性

- 使用官方 Go MCP SDK v1.8.0 替换手写协议分发，提供无状态 Streamable HTTP，明确支持 `2025-11-25`、`2025-06-18`、`2025-03-26`。
- 修正版本协商、初始化参数校验、通知 `202` 空正文、协议入口 `GET/DELETE 405`、请求头和 JSON-RPC 错误处理。
- 工具 `content` 返回完整 JSON 业务结果，同时提供结构化结果及输入/输出 Schema 校验；明确区分 JSON null、空集合、204、业务失败、非 JSON、异常空正文和响应超限。
- 移除固定监听端口的内部 HTTP 回环，改用进程内 REST 适配器，共享现有身份认证、DTO 校验和业务服务。
- 修复自定义 MCP 路径被前端 SPA fallback 返回 HTML 的问题；普通 REST 请求不再依赖 MCP 设置表查询。

### 权限与可观测性

- MCP 严格校验 Origin，使用明确的 OAuth scope 元数据筛选工具，并保留 REST 的权限与管理员校验。
- 新增通用代理和敏感工具开关，默认关闭；OAuth 不开放通用代理，密钥写入结果按明文读取权限脱敏。
- 增加请求关联 ID、响应字节数、Content-Type、MCP 方法与工具、业务状态、耗时和错误类别，避免记录凭据和敏感正文。
- 新增官方 SDK Client、PostgreSQL、真实 Nginx HTTPS、故障与权限回归，以及 MCP CI 工作流；工具清单由注册表生成并自动校验。

### 升级说明

- 协议入口继续使用后台配置的路径，默认 `/mcp`；人类说明页迁至 `<MCP路径>/info`，配置 JSON 仍在 `<MCP路径>/config`。
- 不再提供旧 `2024-11-05` HTTP+SSE 兼容入口。标准客户端须使用 Streamable HTTP，并发送 JSON Content-Type、JSON/SSE Accept 和协商后的协议版本头。
- `generic_tools_enabled` 与 `sensitive_tools_enabled` 升级后默认关闭；原有依赖通用代理、凭据操作或密钥明文读取的自动化需由管理员按需开启。注册工具 80 个，默认启用 74 个，OAuth 列表还受 scope 限制。
- 部署时同步更新仓库 Nginx 配置；使用其他网关时确认自定义 MCP 路径被转发到后端。`api`、`oauth`、`health`、`assets` 为保留路径。
- 业务调用目前保留进程内 REST 适配边界；部分原有数据库操作仍不能被请求取消强制中断。线上“200 零字节”的唯一根因尚需生产抓包确认。
- 本次 Release 提供源码，不附带预编译二进制；推送 main 后由现有 CNB 流水线构建和部署前后端。

### 验证

- 本地全量 `go test -race ./...`（包含 PostgreSQL、官方 SDK Client 和 Nginx HTTPS）及 `go vet ./...` 通过。
- 前端生产构建、MCP 设置页两项端到端测试、CI 语法检查及工具文档一致性检查通过。

详细说明见 [MCP 分析、改造与验收](ops-timer-backend/docs/mcp-reliability.md)、[工具注册表](ops-timer-backend/docs/mcp-tools.md) 和 [管理员配置](ops-timer-backend/docs/mcp-admin-settings.md)。

## v1.1.0 — 2026-10-26

### 新增与改进

- 管理员可在「系统设置 → MCP 与 OAuth 管理」设置服务开关、接入路径、外部地址、服务信息、超时、API Token 接入与 OAuth 动态注册，保存后立即生效。
- MCP/OAuth 设置持久化到 PostgreSQL，支持预览、复制已保存的客户端 JSON 配置。
- 增加管理员身份和服务端权限校验，阻止普通用户、API Token、OAuth 令牌及 MCP 通用代理修改系统认证策略。
- 外部智能体支持 OAuth 授权码 + PKCE、浏览器授权确认、scope 限制、刷新令牌轮换及授权撤销。
- 增加 Qt 桌面客户端及 Windows 打包支持，改进笔记编辑与草稿恢复。
- 修正前端生产 API 地址、构建提交信息展示及后端配置说明。

### 升级说明

- 后端现在仅支持 PostgreSQL。由 v1.0.0 使用 SQLite 的实例升级时，需准备 PostgreSQL；本版本不提供 SQLite 自动迁移。
- 首次升级自动创建 MCP 设置表，并导入旧 `TASK_MANAGER_MCP_*` / `TASK_MANAGER_OAUTH_*` 配置；后续启动以数据库设置为准，环境变量不再覆盖管理员设置。
- 旧实例尚无管理员时，将创建时间最早的现有账户设为管理员；新实例的初始账户默认是管理员。
- 数据库连接、JWT 密钥及网站登录策略仍由部署环境配置。修改 MCP 路径时需同步反向代理和客户端 URL。
- 本次 Release 提供源码，不附带预编译二进制；线上前后端由 CNB 现有流水线发布。

认证机制与迁移细节见 [MCP 管理说明](ops-timer-backend/docs/mcp-admin-settings.md) 和 [OAuth 接入说明](ops-timer-backend/docs/oauth.md)。
