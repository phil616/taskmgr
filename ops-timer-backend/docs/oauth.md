# 外部智能体 OAuth 接入

后端同时作为 OAuth 授权服务器和资源服务器。用户在后端提供的浏览器页面使用现有账号登录并确认授权；智能体只取得独立、有限权限的令牌，无需保存账号密码或长期 API Token。现有浏览器 Cookie、JWT 和 API Token 继续可用。

采用 Authorization Code + PKCE（仅 S256）、公共客户端（`token_endpoint_auth_method=none`）、不透明 Bearer Token、刷新轮换及重放检测。提供 RFC 8414 / RFC 9728 自动发现、RFC 7591 动态注册、RFC 7009 撤销和 RFC 8707 resource 绑定。不提供隐式授权、密码授权、client_credentials 或 OIDC ID Token。

## 部署

管理员登录「系统设置 → MCP 与 OAuth 管理」，启用 OAuth 并填写授权服务器地址（issuer），按需启用动态注册，保存后立即生效。后端自动创建 OAuth 表和 `mcp_settings` 单例配置表，不再通过修改 `.env` 或重启应用管理这些设置。

issuer 必须为后端的外部 HTTPS 来源，不带路径；本地开发允许 `http://localhost:8080` 或 loopback IP。服务不会从 Host 或转发请求头推断可信 issuer。`TASK_MANAGER_AUTH_JWT_SECRET` 仍为网站登录使用的启动密钥（至少 32 字节），也用于签名十分钟有效的授权确认表单；多副本须使用相同密钥和 PostgreSQL。

升级时只在配置表没有记录时导入旧 `TASK_MANAGER_OAUTH_*` 和 `TASK_MANAGER_MCP_*` 值，之后数据库是唯一运行时配置来源。首次安装默认启用 MCP/API Token，OAuth 关闭。管理员角色和迁移细节见 [MCP 管理说明](mcp-admin-settings.md)。

反向代理须原样转发 `/oauth/`、`/.well-known/`、`/mcp` 和 `/api/`，包含查询参数、请求体、Authorization、Cookie、Set-Cookie、Location 和 WWW-Authenticate 响应头。仓库内 Nginx 已包含这些规则。外部 HTTPS 可在代理处终止；OAuth Cookie 是否 Secure 由 issuer 决定。授权页面直接由后端渲染，无需部署新版前端。

默认规范资源地址是 `<issuer>/mcp`，管理员自定义 MCP 接入路径时随之改变；它表示同一任务管理器的 MCP 和 REST API，授权令牌可用于两种入口。部署时不要改变或隐藏这个地址。管理员设置的“外部服务地址”是 API Token 配置的展示选项，OAuth 使用固定 issuer。

## MCP 客户端

支持 OAuth 自动发现和动态注册的客户端只需填写 MCP URL：

```json
{
  "mcpServers": {
    "ops-timer-mcp": {
      "url": "https://tasks.example.com/mcp"
    }
  }
}
```

未认证调用返回 `401`，`WWW-Authenticate` 指向资源元数据，客户端发现授权服务器后注册、打开浏览器并完成授权。`GET /mcp/config` 在启用 OAuth 后返回上述配置及 OAuth 元数据。首次默认只请求 `data:read`；需要创建或修改数据时，客户端应在授权请求中显式包含 `data:write`。MCP 工具列表按权限过滤；调用未开放的工具会返回 SDK 的协议错误，已执行工具的业务/权限失败以 isError=true 表达。REST 的越权请求返回 `403 insufficient_scope` 和对应挑战头。OAuth 客户端不开放通用 `backend_request`；不能用附加请求头替换授权身份。协议和敏感工具策略见 [MCP 改造说明](mcp-reliability.md)。

## 权限

| Scope | 能力 |
|---|---|
| `data:read` | 读取任务、项目、待办、笔记/草稿、通知、日程、钱包、财务数据及个人资料 |
| `data:write` | 创建、修改、删除上述业务数据；不隐含读取权限 |
| `secrets:read` | 读取密钥（含明文）及密钥审计日志，须单独授权 |
| `secrets:write` | 创建、修改、删除密钥；不隐含读取权限 |

无论申请哪些 scope，OAuth 都不能修改账号/密码、获取或重置 API Token、管理 OAuth 客户端或授权、执行整库导入导出。每次授权均明确显示应用名（注册者自行声明）、精确回调地址、资源和完整权限，并要求用户登录确认。业务数据沿用项目现有的共享工作区模型，没有增加租户或按项目隔离。

## 手动 REST 对接

以下流程适用于自定义智能体；凭据仅在用户浏览器提交给任务管理器。

1. `GET /.well-known/oauth-protected-resource`（也支持 `/.well-known/oauth-protected-resource/mcp`）获取 resource 和授权服务器。
2. `GET /.well-known/oauth-authorization-server` 获取各端点和支持的能力。
3. 注册客户端：

```http
POST /oauth/register
Content-Type: application/json

{"client_name":"My Agent","redirect_uris":["http://127.0.0.1:43123/callback"],"token_endpoint_auth_method":"none","grant_types":["authorization_code","refresh_token"],"response_types":["code"]}
```

保存返回的 `client_id`。回调必须为 HTTPS 或本机 loopback HTTP，无 fragment、用户信息或通配符。授权时采用完全匹配（包括端口和原查询参数）。每个直接网络来源每小时最多注册 20 个客户端；代理后的请求共享此额度，避免信任可伪造的转发 IP。可关闭动态注册，并通过已登录用户的 `POST /api/v1/oauth/clients` 提前注册（相同 JSON 请求）；此时发现文档不公布公共注册端点。

4. 生成随机 `state` 和 43～128 个字符的 `code_verifier`，计算 `BASE64URL(SHA256(code_verifier))` 作为 `code_challenge`，不带 `=`。在浏览器打开：

```text
https://tasks.example.com/oauth/authorize?response_type=code&client_id=CLIENT_ID&redirect_uri=ENCODED_CALLBACK&code_challenge=CHALLENGE&code_challenge_method=S256&scope=data%3Aread%20data%3Awrite&state=RANDOM_STATE&resource=https%3A%2F%2Ftasks.example.com%2Fmcp
```

授权页面提交后会 `303` 回调。客户端必须校验返回的 `state` 与 `iss`（应等于配置的 issuer），并处理用户拒绝时的 `error=access_denied`。授权码五分钟有效、单次使用。无效 client_id/回调地址等请求在本地返回错误，不重定向。

5. 用表单编码请求兑换令牌，不使用 JSON 或 client secret：

```bash
curl https://tasks.example.com/oauth/token \
  --data-urlencode grant_type=authorization_code \
  --data-urlencode client_id=CLIENT_ID \
  --data-urlencode code=CODE \
  --data-urlencode redirect_uri=http://127.0.0.1:43123/callback \
  --data-urlencode code_verifier=VERIFIER \
  --data-urlencode resource=https://tasks.example.com/mcp
```

成功返回裸 OAuth JSON（没有业务 API 的 `data` 包装）：

```json
{"access_token":"ot_at_...","token_type":"Bearer","expires_in":900,"refresh_token":"ot_rt_...","scope":"data:read data:write"}
```

6. 调用业务接口或 MCP：

```bash
curl https://tasks.example.com/api/v1/todos \
  -H 'Authorization: Bearer ACCESS_TOKEN'

curl https://tasks.example.com/mcp \
  -H 'Authorization: Bearer ACCESS_TOKEN' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -H 'MCP-Protocol-Version: 2025-11-25' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"todo_list","arguments":{}}}'
```

## 刷新与撤销

Access Token 有效期 15 分钟，授权和 Refresh Token 家族自首次兑换起固定 30 天。刷新不会延长这个期限。

```bash
curl https://tasks.example.com/oauth/token \
  --data-urlencode grant_type=refresh_token \
  --data-urlencode client_id=CLIENT_ID \
  --data-urlencode refresh_token=REFRESH_TOKEN \
  --data-urlencode resource=https://tasks.example.com/mcp
```

每次成功刷新都返回新的 access/refresh token，客户端须原子替换并串行刷新。旧 Refresh Token 再次使用会撤销整个家族，包括已签发的新 Access Token；不要重试已成功消费的刷新请求。可提交 `scope` 缩小权限，不能扩大权限或切换 resource。授权码成功兑换后再次使用（且 PKCE 和其他绑定均正确）也会撤销相应家族。无效绑定不会撤销合法授权。

智能体撤销整次授权：

```bash
curl https://tasks.example.com/oauth/revoke \
  --data-urlencode client_id=CLIENT_ID \
  --data-urlencode token=ACCESS_OR_REFRESH_TOKEN
```

用户使用原有 Cookie/JWT/API Token 认证访问 `GET /api/v1/oauth/grants` 查看自己的有效授权，`DELETE /api/v1/oauth/grants/{id}` 撤销。此管理接口不接受智能体 OAuth 凭据。改密后该账号的 OAuth Access/Refresh Token 及尚未兑换的授权码立即失效。

数据库仅保存授权码、Access Token、Refresh Token 的 SHA-256 摘要。授权查询参数不写入应用访问日志；代理层也应避免记录 OAuth 查询参数、请求体或 Authorization。服务每小时清理到期授权和无效令牌；已使用 Refresh Token 保留到家族失效，供重放检测。OAuth 状态不随业务数据备份导出。

## 验证

```bash
cd ops-timer-backend
go test ./...
TASK_MANAGER_TEST_DATABASE_DSN='postgres://USER:PASSWORD@localhost:5432/TEST_DB?sslmode=disable' go test ./... -count=1
go vet ./...
```

集成测试使用随机 PostgreSQL schema 并在结束后清理，覆盖登录确认/拒绝、CSRF/篡改、发现、注册、PKCE、令牌摘要、REST/MCP 权限、代理绕过、刷新并发及重放、撤销、过期和改密失效。未设置测试 DSN 时集成测试跳过。

参考：[MCP 授权规范](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)、[OAuth 安全最佳实践 RFC 9700](https://www.rfc-editor.org/rfc/rfc9700.html)、[受保护资源元数据 RFC 9728](https://www.rfc-editor.org/rfc/rfc9728.html)。
