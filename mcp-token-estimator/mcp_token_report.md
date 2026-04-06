# ops-timer MCP Token 消耗报告（实测）

**数据来源**：`--live` 拉取运行中后端 `tools/list`  
**测算方式**：`tiktoken(cl100k_base)`  
**生成说明**：数值来自一次完整 `estimate` 运行输出；与模型厂商实际计费 tokenizer 可能略有差异。

---

## 核心结论

| 指标 | 数值 |
|------|------:|
| 服务端返回的工具数量 | **63** |
| **完整 JSON-RPC `tools/list` 响应**（推荐作为主指标） | **6,655** tokens |
| 仅 `result.tools` 紧凑 JSON（本地重序列化） | 6,703 tokens |

> 注：完整响应为 HTTP 原文；「仅 tools」为脚本对 `result.tools` 再 `json.dumps` 的结果。二者若出现与直觉不符的大小关系，以 **完整 JSON-RPC 响应** 为主指标。
| 握手阶段小消息（initialize / notifications / tools/list 请求 / ping 等） | 158 tokens |
| 各工具 schema token 合计 | 8,342 tokens |
| 平均每工具在列表中的 schema 体量 | **约 132** tokens / 工具 |

> 说明：`tools/list` 的大响应一般会进入模型上下文，是**会话级固定开销**；若客户端支持工具 / 系统提示缓存，多轮对话中这部分可被摊薄。

---

## 各工具 Schema 体量（Top 20，降序）

| 排名 | 工具名 | schema (tokens) | 典型调用 input | 典型响应 output |
|:---:|:---|---:|---:|---:|
| 1 | `unit_create` | 607 | 36 | 31 |
| 2 | `unit_update` | 453 | 18 | 29 |
| 3 | `transaction_list` | 345 | 13 | 87 |
| 4 | `transaction_create` | 342 | 42 | 31 |
| 5 | `schedule_create` | 304 | 46 | 31 |
| 6 | `schedule_update` | 297 | 22 | 29 |
| 7 | `wallet_create` | 242 | 11 | 31 |
| 8 | `transaction_update` | 224 | 17 | 29 |
| 9 | `unit_list` | 216 | 13 | 87 |
| 10 | `todo_update` | 209 | 22 | 29 |
| 11 | `backend_request` | 204 | 16 | 11 |
| 12 | `project_update` | 196 | 16 | 29 |
| 13 | `todo_list` | 196 | 13 | 87 |
| 14 | `wallet_update` | 196 | 23 | 29 |
| 15 | `todo_create` | 188 | 9 | 31 |
| 16 | `project_create` | 178 | 7 | 31 |
| 17 | `schedule_list` | 161 | 24 | 87 |
| 18 | `schedule_add_resource` | 148 | 26 | 29 |
| 19 | `budget_category_update` | 148 | 23 | 29 |
| 20 | `budget_category_create` | 147 | 17 | 31 |

（其余 43 个工具见完整 JSON 输出 `mcp_token_estimate.json` 或终端全文。）

---

## 典型场景估算

假设：**每轮对话已包含一次完整 `tools/list` 进入上下文**，下表再叠加「工具调用参数 input」与「典型 HTTP 结果 output」。

| 场景 | 调用链 | 合计 input | 合计 output | **总计** |
|:---|:---|---:|---:|---:|
| 查看今日概览 | project_list → todo_list → notification_unread_count | 6,682 | 185 | **6,867** |
| 创建任务并关联计时 | todo_create → unit_create → unit_update_status | 6,715 | 91 | **6,806** |
| 记账（收入+支出） | wallet_list → transaction_create ×2 → budget_stats | 6,764 | 206 | **6,970** |
| 安排日程 | schedule_list → schedule_create → schedule_add_resource | 6,751 | 147 | **6,898** |
| 密钥管理 | secret_list → secret_get_value → secret_audit_logs | 6,655 | 0 | **6,655** |
| **完整工作流（复杂）** | 9 步混合调用 | **6,852** | **453** | **7,305** |

其中：

- **input** = `tools/list` 固定量（6,655） + 各步典型参数 JSON 的 tokens。
- **output** = 各步典型后端响应 JSON 的 tokens（粗算）。

**关于「密钥管理」场景为 0**：当前服务端返回的 63 个工具中若**未包含** `secret_*` 系列，测算脚本无法匹配 `TYPICAL_ARGS`/典型响应，会显示 0；以实际工具清单为准或补齐静态映射后再跑。

---

## 费用参考（完整工作流场景）

**该场景 token 量**：input = **6,852** output = **453**

| 模型 | input 费用 (USD) | output 费用 (USD) | 合计 (USD) |
|:---|---:|---:|---:|
| Claude Sonnet 4.5 | 0.020556 | 0.006795 | 0.027351 |
| Claude Opus 4.6 | 0.102780 | 0.033975 | 0.136755 |
| Claude Haiku 4.5 | 0.005482 | 0.001812 | 0.007294 |
| GPT-4o | 0.017130 | 0.004530 | 0.021660 |
| GPT-4o-mini | 0.001028 | 0.000272 | 0.001300 |

> 单价为脚本内参考标价（$/1M tokens），仅作量级对比；以服务商实时定价为准。

---

## 建议

1. **工具缓存**：`tools/list` 约 **6.6k** tokens 量级，建议启用客户端 / 模型侧 **prompt caching** 或等价能力，多轮可显著降费。
2. **实测与编码**：若主用 **GPT-4o**，可再跑 `--encoding o200k_base` 对照；与 **Claude** 实际 tokenizer 不同，本报告为 **cl100k_base 估算**。
3. **与静态快照对比**：仓库内嵌 Python 工具定义与线上一致性以 **`--live`** 为准；工具数量 63 与文档中「70」若不一致，以后端部署版本为准。

---

## 复现命令

```bash
cd mcp-token-estimator
uv run estimate --live http://127.0.0.1:8080 --token <你的_API_Token>
```

原始结构化数据：`mcp_token_estimate.json`（同目录，由脚本生成）。
