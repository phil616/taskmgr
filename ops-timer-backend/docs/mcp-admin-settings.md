# MCP 认证与管理员配置

## 认证流程

MCP 通过 HTTP JSON-RPC 接收请求，默认路径 `/mcp`。认证在解析和执行工具之前完成：

1. `Authorization: Bearer ot_at_...` 使用 OAuth Access Token。后端核对数据库中的令牌摘要、有效期、授权撤销状态、resource 绑定及用户密码指纹，再按 scope 筛选工具、检查调用权限。OAuth 令牌不接受降级为 API Token。
2. 启用 API Token 接入时，`X-API-Token` 或 `Authorization: Bearer <api_token>` 核对用户表中的个人 API Token。此方式拥有账号的完整业务权限。
3. 网站登录 JWT 与浏览器 Cookie 不用于 MCP 入口认证。它们用于网站和管理 API；MCP 的内部 HTTP 代理保留传入身份，不能替换为其他认证头。

OAuth 使用授权码 + PKCE S256、浏览器确认、15 分钟 Access Token、固定 30 天授权家族、刷新轮换和重放撤销。数据库仅存 OAuth 凭据摘要。具体流程、scope 及撤销接口见 [oauth.md](oauth.md)。自动发现及 resource 绑定继续遵循 [MCP HTTP 授权规范](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)。

## 管理入口

管理员登录后进入「系统设置 → MCP 与 OAuth 管理」：

| 设置 | 生效范围 |
|---|---|
| MCP 开关、接入路径 | MCP 说明页、配置 JSON 和 JSON-RPC 入口 |
| 外部服务地址 | API Token 客户端配置展示；须为不带路径的 HTTPS 来源，本机开发可使用 HTTP |
| 服务名称、版本、请求超时 | 客户端配置、initialize 响应和后端代理请求 |
| API Token 接入开关 | 仅控制 MCP；REST API 的原有 Token 认证保持可用 |
| OAuth 开关、issuer | OAuth 发现、注册、授权、令牌端点，以及 MCP/REST 的 OAuth 令牌认证 |
| 动态注册开关 | 公共 `/oauth/register` 端点及发现文档；关闭时可通过已登录账户预注册 |

配置保存到 PostgreSQL 的 `mcp_settings` 单例记录，记录修改者和修改时间。每个请求读取一次数据库快照，不修改共享 OAuth 对象，多副本使用同一数据库即可在后续请求读取新设置。读取失败返回 503，不会回退到旧环境配置或放开认证。

启用 MCP 时必须保留 API Token 或 OAuth 中至少一种认证方式。关闭 MCP 后，管理入口仍可使用。关闭 OAuth 后，已签发的 OAuth 凭据立即停止认证；重新启用相同 issuer/path 时尚未过期且未撤销的授权可继续使用。更改 issuer/path 后，旧 resource 的授权被拒绝；客户端需要重新授权。更改接入路径还需同步反向代理规则和客户端 URL。

管理员可预览并复制已保存配置，同时启用两种认证时可选择生成 OAuth 或 API Token 配置。API Token 配置只包含占位符，不会嵌入真实个人凭据。

## 管理员与接口权限

用户的 `is_admin` 标记存储在数据库，个人资料接口不能修改它。首次创建的账户标记为管理员；旧实例升级时，如果尚无管理员，则将数据库中创建时间最早的现有账户标记为管理员。识别依据是稳定的用户 ID，不依赖可编辑用户名。已存在管理员时不会提升其他账户。

`GET /api/v1/admin/mcp-settings` 读取设置，`PUT /api/v1/admin/mcp-settings` 提交完整设置文档：

```json
{
  "enabled": true,
  "path": "/mcp",
  "external_url": "https://tasks.example.com",
  "server_name": "ops-timer-mcp",
  "server_version": "2.0.0",
  "timeout_seconds": 30,
  "api_token_enabled": true,
  "oauth_enabled": true,
  "oauth_issuer": "https://tasks.example.com",
  "oauth_dynamic_registration": true
}
```

管理接口要求管理员的浏览器 Cookie（沿用网站 Origin、自定义请求头检查）或网站登录 JWT。API Token 和 OAuth 凭据均不能访问，即使所属账户是管理员，也不能通过 MCP `backend_request` 修改服务器策略。未登录返回 401，权限不足返回 403，无效或不完整配置返回 400。

## 升级与部署

升级首次启动自动创建配置表并导入旧 `TASK_MANAGER_MCP_*` / `TASK_MANAGER_OAUTH_*` 值；没有旧配置则采用 MCP/API Token 开启、OAuth 关闭、路径 `/mcp` 的默认值。数据库已经存在配置时，之后的启动不再应用这些环境变量，旧值可从部署配置中移除。

数据库连接、JWT 签名密钥、网站登录策略等仍是启动配置，本次迁移只覆盖 MCP/OAuth。管理员设置不包含密钥，也不包含真实 API Token。业务 JSON 导入导出不包含此系统配置表或管理员身份；完整灾备需备份 PostgreSQL 和部署密钥。

运行验证：

```bash
cd ops-timer-backend
TASK_MANAGER_TEST_DATABASE_DSN='postgres://USER:PASSWORD@localhost:5432/TEST_DB?sslmode=disable' go test ./... -count=1
go vet ./...
```

集成测试创建独立 PostgreSQL schema，覆盖旧账户管理员迁移、配置持久化、环境值不覆盖、普通账户/集成凭据越权阻断、浏览器会话校验、MCP 动态路径和服务信息、API Token 开关、OAuth 注册/发现/认证关闭，以及数据库读取失败时拒绝请求。
