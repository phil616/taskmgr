# MCP 分析、改造与验收

依据 [Issue #8](https://github.com/phil616/taskmgr/issues/8) 对当前实现逐项核查。下列结论区分源码可确认的缺陷、已完成的本地验证，以及仍需要生产证据的现象。

## 问题和处理结果

| 项目 | 源码分析 | 本次改造 |
|---|---|---|
| 传输与版本 | 固定声明 2024-11-05，却没有对应的旧 HTTP+SSE 传输 | 使用官方 Go SDK v1.8.0；明确限制协商版本为 2025-11-25、2025-06-18、2025-03-26 |
| GET 入口 | 协议入口返回 HTML，客户端探测时无法解析 | GET/DELETE 在认证后返回 405 + Allow: POST；说明页迁到 `/info` |
| 通知与 JSON-RPC | initialized 返回 200 `{}`，错误和 ID 处理不完整 | SDK 接管消息解析、ID、通知 202、HTTP/JSON-RPC 错误及版本头校验；额外验证 initialize 必填参数 |
| 工具输出 | 文本只有 HTTP 摘要，真正数据只在 structuredContent | 文本包含完整 JSON 结果，同时提供相同的结构化结果；SDK 校验 inputSchema/outputSchema |
| 空值和异常 | 空正文、null、非 JSON、截断没有清楚区分 | 204 用 no_content；JSON null/数组/对象保留；异常成功空正文、非 JSON、超限返回工具错误 |
| 业务调用 | MCP 依赖固定 loopback IP 和监听端口，重复走网络 | 启动时安装进程内 REST 适配器，调用相同认证、DTO、Handler、Service；不再建立回环连接 |
| 全局设置依赖 | 普通 REST 请求也查询 MCP 设置，可被配置故障拖垮 | 动态 MCP 放到 NoRoute；OAuth 端点按需读取；REST 仅实际携带 OAuth 凭据时查询 |
| 动态路径 | Nginx 固定 `/mcp`，其他路径可能得到 SPA HTML 200 | 未命中静态文件先到后端，只有 GET 且 Accept 含 text/html 的后端 404 才回退 SPA |
| Origin | CORS 不出头不能阻止服务端执行非法来源请求 | MCP 对非法 Origin 直接 403；允许无 Origin 的非浏览器客户端，只信任显式配置来源 |
| 代理与敏感能力 | 通用代理、凭据读取和明文密钥暴露过宽 | 两项独立开关默认关闭；敏感路径检查同时作用于代理；OAuth 完全隐藏通用代理，保持 REST scope/管理员校验 |
| 工具发现 | 使用伪造 scope-check 参数推导权限 | 明确 scope 元数据；注册表测试逐项与共享 REST 权限函数核对 |
| 可观测性 | 无法区分 HTTP 零字节、HTML 和工具数据缺失 | 响应字节数、Content-Type、请求 ID、MCP 方法/工具、业务状态、耗时和错误类别；不记录凭据、参数、正文或查询字符串 |
| 测试和文档 | 缺少真实客户端，工具数量不同步 | 官方 SDK Client、PostgreSQL、真实 Nginx HTTPS、故障注入、权限和工具文档生成校验；新增 CI |

源码确认的是“客户端只消费 content 时丢失业务结果”，以及“自定义路径被旧 Nginx SPA fallback 返回 HTML”的故障路径。尚未取得生产环境有效 MCP POST 原始响应为零字节的抓包，因此不能声称已确认该线上现象的唯一根因。测试中的通知 202 空正文是标准行为。

## 协议契约

默认端点为 `/mcp`，后台可更改，保留 api、oauth、health 和 assets 命名空间以避免业务及静态资源冲突。客户端 POST 请求必须包含：

```http
Content-Type: application/json
Accept: application/json, text/event-stream
Authorization: Bearer <api-token-or-oauth-token>
```

API Token 也可通过 `X-API-Token` 传递。后续请求应携带协商出的 `MCP-Protocol-Version`。initialize 必须提供 protocolVersion、capabilities 对象和 clientInfo 的 name/version。标准客户端应先 initialize，再发送 notifications/initialized，然后发现和调用工具。

采用 SDK 的无状态模式：没有持久 Mcp-Session-Id，不保存客户端握手、订阅或能力状态；每个 HTTP 请求都重新验证身份、读取相关策略并建立临时 SDK 会话。服务端不以“曾初始化”作为授权条件，不拒绝独立业务请求。该模式支持多副本及配置即时变更，不能依赖会话存储来执行多轮服务端请求。初始化只声明 tools 能力，不声明工具列表变更推送或日志订阅。

本次不提供 2024-11-05 的旧 HTTP+SSE，也不开放 SDK 中的 2026-07-28 新模型。SDK 默认版本较新的客户端可协商降级，测试覆盖该路径。

| 情形 | 行为 |
|---|---|
| 成功的请求 | HTTP 200，application/json，单个 JSON-RPC 响应及对应 ID |
| 被接受的通知/客户端响应 | HTTP 202，零字节正文 |
| GET/DELETE 协议入口 | 认证后 405，Allow: POST |
| 非法 Origin | 403，认证/工具执行之前拒绝 |
| 无效 Token | 401；OAuth 启用时提供 WWW-Authenticate |
| 错误 Content-Type / Accept | SDK 分别拒绝为 415 / 400 |
| 超过 4 MiB 的 MCP 请求 | 413（按读取字节检查，适用 chunked 请求） |
| 无效 JSON、JSON-RPC 结构/版本 | SDK 处理 HTTP 输入错误或对应 JSON-RPC 错误 |
| 业务失败、DTO/Schema 错误 | JSON-RPC 工具结果 isError=true；无可用工具等协议错误由 SDK 返回 error |

## 结果契约

`content[0].text` 是以下对象的 JSON，`structuredContent` 提供相同对象：

```json
{
  "status": 200,
  "method": "GET",
  "path": "/api/v1/todos",
  "kind": "json",
  "data": {"code": 0, "message": "success", "data": []}
}
```

保留 REST 的业务包裹、分页 meta、空数组、空对象和 JSON null。204 返回 `kind=no_content, data=null`。空的成功响应会明确报 empty_backend_response；非 JSON（包括 HTML）、损坏 JSON 或被截断的响应报 invalid_backend_response；超过 10 MiB 报 response_too_large，不返回不完整业务数据。HTTP 非 2xx 或 REST 非零业务 code 标记为工具失败。

不再返回内部 URL、原始响应头或 Cookie 等元数据。敏感工具显式启用后，业务值会按工具职责返回给获授权客户端，不能把它们写入日志。

## 权限和升级

新增 `generic_tools_enabled` 和 `sensitive_tools_enabled`，数据库升级后均默认关闭，管理员可在设置页独立启用。原有调用这些工具的自动化需调整；OAuth 不开放通用代理，即使管理员启用该开关。

敏感开关覆盖 auth_get_token、auth_regenerate_token、auth_change_password、secret_get、secret_get_value，并阻止通用代理绕道调用凭据、备份、授权管理和密钥明文路径。密钥创建/更新的 REST DTO 也包含 value；敏感开关关闭时，MCP 会从这些写入结果中移除该字段，避免通过只改名称读取旧值。OAuth 缺少 secrets:read 时，即使敏感开关开启，密钥写入结果也移除 value。其他工具仍使用原有 REST 身份/业务权限。代理无法注入 Authorization、X-API-Token 或 Cookie 来替换调用者。管理 API 仍拒绝 API Token/OAuth 身份。

升级流程：部署后端与仓库 Nginx 配置，检查外部代理转发自定义路径，更新客户端 URL，按需开启能力。GET 说明页 URL 从 `<MCP路径>` 改为 `<MCP路径>/info`。配置 JSON 仍在 `<MCP路径>/config`。工具注册表共 80 个，默认启用 74 个；OAuth 的实际数量依 scope 而定。完整表见 [自动生成清单](mcp-tools.md)。`mcp-token-estimator` 的静态表及已有报告保留历史快照含义，当前工具和完整 schema 应使用其 `--live` 模式测量；实时请求已更新为标准 Accept/协议头。

Nginx 测试直接加载仓库 nginx.conf，仅替换测试监听端口、TLS 证书和上游地址；覆盖自定义路径无需重新生成 Nginx 配置、真实 TLS 校验及 SPA 导航保留。采用其他网关或独立前端托管时，应对真实 API 域名执行下面的逐字节检查。

## 复现与验收

```bash
cd ops-timer-backend
# DSN 指向专用测试数据库；每个集成测试创建独立 schema。
export TASK_MANAGER_TEST_DATABASE_DSN='postgres://USER:PASSWORD@127.0.0.1:5432/TEST_DB?sslmode=disable'
export TASK_MANAGER_TEST_NGINX_IMAGE=nginx:alpine
# HTTPS 测试要求本机 Docker，临时 Nginx 使用 Linux host 网络。
docker pull "$TASK_MANAGER_TEST_NGINX_IMAGE"
go test -race ./... -count=1 -timeout=5m
go vet ./...
# 修改工具后更新清单；普通 go test 会检查清单是否过期。
go generate ./internal/api/handler
```

未设置 DSN 时数据库集成测试明确 skip；未设置 Nginx 镜像时仅 HTTPS 集成测试 skip。CI 提供两项配置，完整运行，不依赖生产令牌。官方 SDK 客户端针对三个明确版本和默认协商各重复 tools/list、tools/call 五次，分别覆盖直连、默认 Nginx HTTPS 入口和动态路径。

生产排障时，用同一请求和有效凭据比较本机与外部入口，保存 headers/body 到本机受控位置：

```bash
curl -sS -D headers.txt -o body.txt \
  -w 'HTTP=%{http_code} TYPE=%{content_type} BYTES=%{size_download}\n' \
  -H 'Authorization: Bearer <TOKEN>' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -H 'MCP-Protocol-Version: 2025-11-25' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"todo_list","arguments":{}}}' \
  'https://<API_HOST>/<MCP_PATH>'
wc -c body.txt
```

请求业务结果应为 200 + JSON + 非零字节；202 空正文只应出现在通知/响应消息。200 HTML 表示入口路由错误。200 JSON 若客户端仍报告空结果，应检查 content 和客户端日志。使用响应 X-Request-ID 关联应用日志及代理日志，比较 response_bytes、Content-Type、upstream_status 和分段耗时。不要把凭据或 body.txt 的业务数据粘贴到公开 Issue。

## 架构边界与后续演进

协议层已交给 SDK，注册表与参数映射分离到 mcp_tools.go，进程内适配器在 mcp_dispatch.go。现有 REST 验证及业务行为得到保留；暂未为全部 80 个工具重写独立的 Service/UseCase 适配器，当前仍存在进程内的 HTTP 请求/响应序列化。这是可继续替换的过渡边界，但生产路径已经没有回环网络、监听地址或连接超时依赖。

调用携带超时/取消 Context，并在分发前后检查；但原有部分 Service/Repository 未使用请求 Context，因此不能强行中断已经进入数据库的操作，超时也不能回滚已完成的副作用。后续统一为 Repository 增加 Context 参数后，可进一步落实数据库取消。当前不会因超时自动重试写操作。

本地 HTTPS 测试验证仓库提供的代理配置，不能代替生产 CDN/负载均衡/网关的验证。上线后仍需对真实域名重复初始化、发现和调用并检查原始字节；本次未执行生产部署。

参考：[官方 Go SDK](https://github.com/modelcontextprotocol/go-sdk)、[2025-11-25 传输规范](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)、[生命周期](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle)、[工具规范](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)。
