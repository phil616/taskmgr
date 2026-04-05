# 运维任务管理系统 — 后端文档

本目录包含 **HTTP API** 的规范说明，供前端、脚本、第三方集成与自动化工具使用。

## OpenAPI 规范

| 文件 | 说明 |
|------|------|
| [`openapi.yaml`](./openapi.yaml) | OpenAPI **3.0.3** 完整描述（路径、请求/响应模型、认证方式、错误码） |

### 使用方式

1. **导入 Postman / Insomnia / Apifox**  
   选择「Import」→「OpenAPI」→ 选择本仓库的 `ops-timer-backend/docs/openapi.yaml`。

2. **Swagger UI（本地预览）**  
   ```bash
   npx @redocly/cli preview-docs ops-timer-backend/docs/openapi.yaml
   ```
   或使用 Docker：
   ```bash
   docker run -p 8080:8080 -e SWAGGER_JSON=/docs/openapi.yaml -v "%cd%/ops-timer-backend/docs:/docs" swaggerapi/swagger-ui
   ```
   浏览器访问 `http://localhost:8080`（Windows 下路径请按实际调整）。

3. **代码生成**  
   使用 [OpenAPI Generator](https://openapi-generator.tech/) 等工具，根据 `openapi.yaml` 生成各语言客户端 SDK。

   ```bash
   docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli generate \
     -i /local/ops-timer-backend/docs/openapi.yaml \
     -g go \
     -o /local/out/go-client
   ```

4. **CI 校验**  
   可在流水线中执行 `npx @redocly/cli lint ops-timer-backend/docs/openapi.yaml` 校验规范文件。

## 基础约定

- **Base URL**：默认 `http://localhost:8080`（部署时替换为实际域名与端口）。
- **API 前缀**：业务接口为 `/api/v1`；健康检查为 `/health`。
- **认证**（需登录的接口）：
  - **Bearer JWT**：`Authorization: Bearer <access_token>`（登录接口返回的 `token`）。
  - **API Token**：`X-API-Token: <api_token>`（用户可在系统内生成/轮换，适合脚本与集成）。
- **响应格式**：JSON。成功时 `code` 为 `0`；失败时 HTTP 状态码与业务 `code` 见 OpenAPI 文档中的「错误码」说明。
- **分页列表**：查询参数 `page`（默认 1）、`page_size`（默认 20）；响应体在 `meta` 中携带 `total`、`total_pages` 等。

## MCP 约定（AI Agent）

### 概览

MCP 服务器 v2.0 提供 **62 个专用工具**，100% 覆盖后端所有 API，智能体通过一个 JSON 配置即可获得全部能力。

### 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/mcp` | MCP JSON-RPC 入口（`initialize`、`ping`、`tools/list`、`tools/call`、`notifications/initialized`） |
| `GET` | `/mcp/config` | 返回可直接粘贴到 MCP 客户端的 JSON 配置片段 |

### 认证

- `X-API-Token: <api_token>`（推荐）
- 或 `Authorization: Bearer <api_token>`

### 工具清单（62 个）

| 模块 | 工具 |
|------|------|
| **计时单元** | `unit_list` `unit_get` `unit_create` `unit_update` `unit_delete` `unit_update_status` `unit_step` `unit_set_value` `unit_logs` `unit_summary` |
| **项目** | `project_list` `project_get` `project_create` `project_update` `project_delete` `project_units` |
| **待办** | `todo_list` `todo_get` `todo_create` `todo_update` `todo_delete` `todo_update_status` `todo_batch` |
| **待办分组** | `todo_group_list` `todo_group_create` `todo_group_update` `todo_group_delete` |
| **通知** | `notification_list` `notification_mark_read` `notification_mark_all_read` `notification_unread_count` `notification_delete` |
| **日程** | `schedule_list` `schedule_get` `schedule_create` `schedule_update` `schedule_delete` `schedule_add_resource` `schedule_remove_resource` |
| **钱包** | `wallet_list` `wallet_get` `wallet_create` `wallet_update` `wallet_delete` |
| **收支分类** | `budget_category_list` `budget_category_create` `budget_category_update` `budget_category_delete` |
| **收支记录** | `transaction_list` `transaction_get` `transaction_create` `transaction_update` `transaction_delete` |
| **预算统计** | `budget_stats` |
| **密钥管理** | `secret_list` `secret_get` `secret_get_value` `secret_create` `secret_update` `secret_delete` `secret_audit_logs` |
| **通用** | `backend_request`（兜底，可代理调用 `/api/v1/*` 与 `/health`） |

### 快速接入

```bash
# 1. 获取配置片段
curl http://localhost:8080/mcp/config

# 2. 将返回的 mcpServers 粘贴到客户端配置，替换 Token
# 3. 智能体即可通过 62 个工具操控全部后端数据
```

## 与配置的关系

- 服务监听地址、CORS、JWT、数据库等见项目根目录 `README.md` 与 `env.example` / 环境变量前缀 `TASK_MANAGER_`。

## 维护说明

- 新增或变更 API 时，请**同步更新** `openapi.yaml`，并保持与 `internal/api/router` 及 DTO 一致。
- 版本号：在 `openapi.yaml` 的 `info.version` 中与发布版本对齐（或采用独立 API 版本号策略并在文档中说明）。
