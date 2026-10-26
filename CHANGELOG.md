# 更新日志

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
