"""
ops-timer MCP Token 消耗测算工具
=====================================
基于 mcp.go 中定义的工具 schema，静态估算 AI 客户端在一次 MCP 会话中的 token 开销。

测算维度：
1. tools/list 完整 JSON-RPC 响应（与 Gin 紧凑 JSON 一致，贴近真实服务端）
2. 可选：HTTP 拉取运行中后端的 tools/list，用真实响应测算
3. 每个工具调用的 input/output token 典型值
4. 常见使用场景的端到端 token 消耗
5. 握手阶段（initialize / tools/list 请求等小消息）的 token 粗算
"""

import argparse
import io
import json
import os
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Callable, Optional

# Windows 下强制 stdout/stderr 使用 UTF-8
if sys.stdout.encoding and sys.stdout.encoding.lower() != "utf-8":
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")
if sys.stderr.encoding and sys.stderr.encoding.lower() != "utf-8":
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding="utf-8", errors="replace")

# ---------------------------------------------------------------------------
# tiktoken / 降级计数
# ---------------------------------------------------------------------------
def make_token_counter(encoding_name: str) -> tuple[Callable[[str], int], str]:
    """encoding_name: cl100k_base（GPT-3.5/4 系常用）或 o200k_base（GPT-4o 系更接近）。"""
    try:
        import tiktoken

        enc = tiktoken.get_encoding(encoding_name)

        def count_tokens(text: str) -> int:
            return len(enc.encode(text))

        return count_tokens, f"tiktoken({encoding_name})"
    except ImportError:
        def count_tokens(text: str) -> int:
            return max(1, int(len(text) / 3.5))

        return count_tokens, "approximate(chars/3.5)"


# ---------------------------------------------------------------------------
# MCP 工具定义（从 mcp.go allTools() 提取）
# ---------------------------------------------------------------------------

TOOLS: list[dict] = [
    # ── 通用 ──────────────────────────────────────────────────────────────
    {
        "name": "backend_request",
        "description": "调用任务管理器后端 HTTP API（支持 GET/POST/PUT/PATCH/DELETE）。当其他专用工具不满足需求时使用此通用工具。",
        "inputSchema": {
            "type": "object",
            "properties": {
                "method":  {"type": "string", "description": "HTTP 方法：GET/POST/PUT/PATCH/DELETE"},
                "path":    {"type": "string", "description": "接口路径，必须以 /api/v1/ 或 /health 开头"},
                "query":   {"type": "object", "description": "查询参数对象（可选）"},
                "body":    {"description": "JSON 请求体（可选）"},
                "headers": {"type": "object", "additionalProperties": {"type": "string"}, "description": "额外请求头（可选）"},
            },
            "required": ["method", "path"],
        },
    },
    # ── Auth ──────────────────────────────────────────────────────────────
    {"name": "auth_get_profile",     "description": "获取当前用户的个人资料信息（用户名、邮箱、昵称等）。",                                        "inputSchema": {"type": "object", "properties": {}}},
    {"name": "auth_update_profile",  "description": "更新当前用户的个人资料（昵称、邮箱等）。",                                                     "inputSchema": {"type": "object", "properties": {"nickname": {"type": "string"}, "email": {"type": "string"}}}},
    {"name": "auth_change_password", "description": "修改当前用户的登录密码。",                                                                     "inputSchema": {"type": "object", "properties": {"old_password": {"type": "string"}, "new_password": {"type": "string"}}, "required": ["old_password", "new_password"]}},
    {"name": "auth_get_token",       "description": "获取当前用户的 API Token 信息。",                                                              "inputSchema": {"type": "object", "properties": {}}},
    {"name": "auth_regenerate_token","description": "重新生成当前用户的 API Token（旧 Token 将失效）。",                                             "inputSchema": {"type": "object", "properties": {}}},
    {"name": "auth_test_email",      "description": "发送测试邮件，验证 SMTP 邮件配置是否正常。",                                                   "inputSchema": {"type": "object", "properties": {"to": {"type": "string"}}, "required": ["to"]}},
    {"name": "auth_smtp_status",     "description": "查询 SMTP 邮件服务的配置状态。",                                                               "inputSchema": {"type": "object", "properties": {}}},
    # ── Units ─────────────────────────────────────────────────────────────
    {"name": "unit_list",          "description": "获取计时单元列表，支持按项目 ID、状态、类型等筛选。",                                               "inputSchema": {"type": "object", "properties": {"project_id": {}, "type": {}, "status": {}, "priority": {}, "page": {}, "page_size": {}}}},
    {"name": "unit_get",           "description": "获取单个计时单元的详细信息。",                                                                     "inputSchema": {"type": "object", "properties": {"id": {"type": "string"}}, "required": ["id"]}},
    {"name": "unit_create",        "description": "创建一个新的计时单元。type 为 time_countdown（倒计时）或 time_countup（正计时）时使用 target_time/start_time；type 为 count_countdown（倒数计数）或 count_countup（正向计数）时使用 target_value/current_value/step。", "inputSchema": {"type": "object", "properties": {"title": {}, "type": {}, "description": {}, "project_id": {}, "status": {}, "priority": {}, "tags": {}, "color": {}, "target_time": {}, "start_time": {}, "display_unit": {}, "current_value": {}, "target_value": {}, "step": {}, "unit_label": {}, "remind_before_days": {}, "remind_after_days": {}}, "required": ["title", "type"]}},
    {"name": "unit_update",        "description": "更新计时单元的基本信息（标题、描述、标签、颜色、优先级、目标时间等）。",                             "inputSchema": {"type": "object", "properties": {"id": {}, "title": {}, "description": {}, "project_id": {}, "priority": {}, "tags": {}, "color": {}, "target_time": {}, "start_time": {}, "display_unit": {}, "target_value": {}, "step": {}, "unit_label": {}, "remind_before_days": {}, "remind_after_days": {}}, "required": ["id"]}},
    {"name": "unit_delete",        "description": "删除指定计时单元。",                                                                               "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "unit_update_status", "description": "更新计时单元的状态（激活/暂停/完成/归档）。",                                                      "inputSchema": {"type": "object", "properties": {"id": {}, "status": {}}, "required": ["id", "status"]}},
    {"name": "unit_step",          "description": "对计数型单元执行一次步进操作（+step 或 -step）。仅适用于 count_countdown / count_countup 类型。", "inputSchema": {"type": "object", "properties": {"id": {}, "direction": {}, "note": {}}, "required": ["id"]}},
    {"name": "unit_set_value",     "description": "直接设置计数型单元的当前值。仅适用于 count_countdown / count_countup 类型。",                    "inputSchema": {"type": "object", "properties": {"id": {}, "value": {}}, "required": ["id", "value"]}},
    {"name": "unit_logs",          "description": "获取计时单元的操作日志/历史记录。",                                                                "inputSchema": {"type": "object", "properties": {"id": {}, "page": {}, "page_size": {}}, "required": ["id"]}},
    {"name": "unit_summary",       "description": "获取计时单元的汇总统计信息（各状态数量等）。",                                                     "inputSchema": {"type": "object", "properties": {}}},
    # ── Projects ──────────────────────────────────────────────────────────
    {"name": "project_list",         "description": "获取所有项目列表（含状态、描述、颜色等信息）。",                                                 "inputSchema": {"type": "object", "properties": {"status": {}, "page": {}, "page_size": {}}}},
    {"name": "project_get",          "description": "获取单个项目的详细信息。",                                                                      "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "project_create",       "description": "创建新项目。",                                                                                   "inputSchema": {"type": "object", "properties": {"title": {}, "description": {}, "color": {}, "icon": {}, "status": {}, "max_budget": {}}, "required": ["title"]}},
    {"name": "project_update",       "description": "更新项目信息（标题、描述、颜色、预算等）。",                                                    "inputSchema": {"type": "object", "properties": {"id": {}, "title": {}, "description": {}, "color": {}, "icon": {}, "status": {}, "max_budget": {}}, "required": ["id"]}},
    {"name": "project_delete",       "description": "删除指定项目。",                                                                                 "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "project_units",        "description": "获取指定项目下的所有计时单元。",                                                                 "inputSchema": {"type": "object", "properties": {"id": {}, "page": {}, "page_size": {}}, "required": ["id"]}},
    {"name": "project_budget_stats", "description": "获取指定项目的预算统计信息，包括总收入、总支出、净额、剩余预算、使用率和关联交易数。",           "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    # ── Todos ─────────────────────────────────────────────────────────────
    {"name": "todo_list",          "description": "查询待办事项列表，支持按分组、状态、优先级筛选。",                                                  "inputSchema": {"type": "object", "properties": {"group_id": {}, "status": {}, "priority": {}, "due_date": {}, "page": {}, "page_size": {}}}},
    {"name": "todo_get",           "description": "获取单条待办事项的详细信息。",                                                                     "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "todo_create",        "description": "创建一条新待办事项。",                                                                             "inputSchema": {"type": "object", "properties": {"title": {}, "description": {}, "group_id": {}, "priority": {}, "due_date": {}, "status": {}}, "required": ["title"]}},
    {"name": "todo_update",        "description": "更新待办事项的信息（标题、描述、优先级、截止日期等）。",                                           "inputSchema": {"type": "object", "properties": {"id": {}, "title": {}, "description": {}, "group_id": {}, "priority": {}, "due_date": {}, "status": {}}, "required": ["id"]}},
    {"name": "todo_delete",        "description": "删除指定待办事项。",                                                                               "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "todo_update_status", "description": "更新待办事项的完成状态。",                                                                         "inputSchema": {"type": "object", "properties": {"id": {}, "status": {}}, "required": ["id", "status"]}},
    {"name": "todo_batch",         "description": "批量操作待办事项（批量完成、删除等）。",                                                           "inputSchema": {"type": "object", "properties": {"action": {}, "ids": {}}, "required": ["action", "ids"]}},
    {"name": "todo_group_list",    "description": "获取所有待办分组列表。",                                                                           "inputSchema": {"type": "object", "properties": {}}},
    {"name": "todo_group_create",  "description": "创建新的待办分组。",                                                                               "inputSchema": {"type": "object", "properties": {"name": {}, "color": {}, "icon": {}}, "required": ["name"]}},
    {"name": "todo_group_update",  "description": "更新待办分组信息。",                                                                               "inputSchema": {"type": "object", "properties": {"id": {}, "name": {}, "color": {}, "icon": {}}, "required": ["id"]}},
    {"name": "todo_group_delete",  "description": "删除待办分组（不会删除分组内的待办事项，待办将移至无分组）。",                                     "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    # ── Notifications ─────────────────────────────────────────────────────
    {"name": "notification_list",          "description": "获取通知消息列表，支持分页。",                                                             "inputSchema": {"type": "object", "properties": {"page": {}, "page_size": {}}}},
    {"name": "notification_mark_read",     "description": "将单条通知标记为已读。",                                                                  "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "notification_mark_all_read", "description": "将所有未读通知标记为已读。",                                                              "inputSchema": {"type": "object", "properties": {}}},
    {"name": "notification_unread_count",  "description": "获取未读通知数量。",                                                                      "inputSchema": {"type": "object", "properties": {}}},
    {"name": "notification_delete",        "description": "删除指定通知。",                                                                           "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    # ── Schedules ─────────────────────────────────────────────────────────
    {"name": "schedule_list",            "description": "查询日程列表，可按日期范围和状态筛选。",                                                     "inputSchema": {"type": "object", "properties": {"start_date": {}, "end_date": {}, "status": {}, "page": {}, "page_size": {}}}},
    {"name": "schedule_get",             "description": "获取某条日程的详细信息（含关联资源）。",                                                     "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "schedule_create",          "description": "创建一条新日程安排。",                                                                       "inputSchema": {"type": "object", "properties": {"title": {}, "description": {}, "start_time": {}, "end_time": {}, "all_day": {}, "color": {}, "location": {}, "status": {}, "recurrence_type": {}, "tags": {}}, "required": ["title", "start_time", "end_time"]}},
    {"name": "schedule_update",          "description": "更新已有日程的信息（标题、时间、状态等）。",                                                 "inputSchema": {"type": "object", "properties": {"id": {}, "title": {}, "description": {}, "start_time": {}, "end_time": {}, "all_day": {}, "status": {}, "location": {}, "recurrence_type": {}, "color": {}, "tags": {}}, "required": ["id"]}},
    {"name": "schedule_delete",          "description": "删除指定日程。",                                                                             "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "schedule_add_resource",    "description": "为日程关联一个已有资源（项目、待办或计时单元）。",                                           "inputSchema": {"type": "object", "properties": {"id": {}, "resource_type": {}, "resource_id": {}, "note": {}}, "required": ["id", "resource_type", "resource_id"]}},
    {"name": "schedule_remove_resource", "description": "从日程中移除一个关联资源。",                                                                 "inputSchema": {"type": "object", "properties": {"id": {}, "resource_id": {}}, "required": ["id", "resource_id"]}},
    # ── Wallets ───────────────────────────────────────────────────────────
    {"name": "wallet_list",   "description": "获取所有钱包/账户列表（含余额、本月收支汇总）。",                                                       "inputSchema": {"type": "object", "properties": {}}},
    {"name": "wallet_get",    "description": "获取单个钱包的详细信息。",                                                                              "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "wallet_create", "description": "创建新钱包/账户（银行卡、现金、支付宝、微信等）。",                                                    "inputSchema": {"type": "object", "properties": {"name": {}, "type": {}, "balance": {}, "currency": {}, "color": {}, "icon": {}, "description": {}, "is_default": {}}, "required": ["name"]}},
    {"name": "wallet_update", "description": "更新钱包信息（名称、颜色、默认设置等）。",                                                              "inputSchema": {"type": "object", "properties": {"id": {}, "name": {}, "type": {}, "color": {}, "icon": {}, "description": {}, "is_default": {}}, "required": ["id"]}},
    {"name": "wallet_delete", "description": "删除指定钱包（软删除，已有交易记录不受影响）。",                                                        "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    # ── Budget Categories ─────────────────────────────────────────────────
    {"name": "budget_category_list",   "description": "获取收支分类列表，可按类型筛选（income/expense/both）。",                                      "inputSchema": {"type": "object", "properties": {"type": {}}}},
    {"name": "budget_category_create", "description": "创建自定义收支分类。",                                                                        "inputSchema": {"type": "object", "properties": {"name": {}, "type": {}, "color": {}, "icon": {}}, "required": ["name", "type"]}},
    {"name": "budget_category_update", "description": "更新分类信息（系统内置分类不可修改）。",                                                      "inputSchema": {"type": "object", "properties": {"id": {}, "name": {}, "type": {}, "color": {}, "icon": {}}, "required": ["id"]}},
    {"name": "budget_category_delete", "description": "删除自定义分类（系统内置分类不可删除，有关联交易的分类不可删除）。",                          "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    # ── Transactions ──────────────────────────────────────────────────────
    {"name": "transaction_list",   "description": "查询收支记录列表，支持按钱包、类型、日期范围、分类、项目等筛选。结果按交易时间倒序排列。",          "inputSchema": {"type": "object", "properties": {"wallet_id": {}, "category_id": {}, "project_id": {}, "type": {}, "start_date": {}, "end_date": {}, "min_amount": {}, "max_amount": {}, "keyword": {}, "page": {}, "page_size": {}}}},
    {"name": "transaction_get",    "description": "获取单条收支记录的详细信息。",                                                                     "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "transaction_create", "description": "新增一条收支记录（收入/支出/转账）。新增后自动更新对应钱包余额。可关联到某个项目。",               "inputSchema": {"type": "object", "properties": {"wallet_id": {}, "type": {}, "amount": {}, "category_id": {}, "project_id": {}, "note": {}, "tags": {}, "transaction_at": {}, "to_wallet_id": {}}, "required": ["wallet_id", "type", "amount", "transaction_at"]}},
    {"name": "transaction_update", "description": "更新收支记录的金额、分类、备注、标签、交易时间或关联项目。",                                       "inputSchema": {"type": "object", "properties": {"id": {}, "category_id": {}, "project_id": {}, "amount": {}, "note": {}, "tags": {}, "transaction_at": {}}, "required": ["id"]}},
    {"name": "transaction_delete", "description": "删除收支记录（软删除），删除后自动回滚对应钱包余额。",                                             "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    # ── Budget Stats ──────────────────────────────────────────────────────
    {"name": "budget_stats", "description": "获取预算统计汇总，包括总收入、总支出、净余额和各分类消费占比。",                                          "inputSchema": {"type": "object", "properties": {"wallet_id": {}, "start_date": {}, "end_date": {}}}},
    # ── Secrets ───────────────────────────────────────────────────────────
    {"name": "secret_list",       "description": "获取密钥列表（不含密钥值），支持按名称、标签、项目筛选。",                                          "inputSchema": {"type": "object", "properties": {"name": {}, "tag": {}, "project_id": {}, "page": {}, "page_size": {}}}},
    {"name": "secret_get",        "description": "获取单个密钥的详细信息（含密钥值）。此操作会被审计记录。",                                          "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "secret_get_value",  "description": "仅获取密钥的值（明文）。此操作会被审计记录为 value_read。",                                        "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "secret_create",     "description": "创建一个新密钥。密钥名称必须唯一。可关联到项目并添加标签。",                                        "inputSchema": {"type": "object", "properties": {"name": {}, "value": {}, "description": {}, "tags": {}, "project_id": {}}, "required": ["name", "value"]}},
    {"name": "secret_update",     "description": "更新密钥的名称、值、描述、标签或关联项目。",                                                        "inputSchema": {"type": "object", "properties": {"id": {}, "name": {}, "value": {}, "description": {}, "tags": {}, "project_id": {}}, "required": ["id"]}},
    {"name": "secret_delete",     "description": "删除指定密钥（软删除）。",                                                                          "inputSchema": {"type": "object", "properties": {"id": {}}, "required": ["id"]}},
    {"name": "secret_audit_logs", "description": "查询密钥的审计日志，记录了每次密钥的访问历史（谁访问的、什么操作、IP 地址等）。",                  "inputSchema": {"type": "object", "properties": {"id": {}, "action": {}, "page": {}, "page_size": {}}}},
]

# ---------------------------------------------------------------------------
# 典型工具调用参数 & 响应（用于 per-call 估算）
# ---------------------------------------------------------------------------

TYPICAL_ARGS: dict[str, dict] = {
    "backend_request":              {"method": "GET", "path": "/api/v1/units"},
    "auth_get_profile":             {},
    "auth_update_profile":          {"nickname": "Alice"},
    "auth_change_password":         {"old_password": "old123", "new_password": "new456"},
    "auth_get_token":               {},
    "auth_regenerate_token":        {},
    "auth_test_email":              {"to": "test@example.com"},
    "auth_smtp_status":             {},
    "unit_list":                    {"page": 1, "page_size": 20},
    "unit_get":                     {"id": "unit-abc123"},
    "unit_create":                  {"title": "完成项目报告", "type": "time_countdown", "target_time": "2025-12-31T00:00:00Z"},
    "unit_update":                  {"id": "unit-abc123", "title": "更新后的标题"},
    "unit_delete":                  {"id": "unit-abc123"},
    "unit_update_status":           {"id": "unit-abc123", "status": "completed"},
    "unit_step":                    {"id": "unit-abc123", "direction": "up"},
    "unit_set_value":               {"id": "unit-abc123", "value": 42},
    "unit_logs":                    {"id": "unit-abc123", "page": 1, "page_size": 10},
    "unit_summary":                 {},
    "project_list":                 {"page": 1, "page_size": 20},
    "project_get":                  {"id": "proj-abc123"},
    "project_create":               {"title": "新项目"},
    "project_update":               {"id": "proj-abc123", "title": "更新项目"},
    "project_delete":               {"id": "proj-abc123"},
    "project_units":                {"id": "proj-abc123"},
    "project_budget_stats":         {"id": "proj-abc123"},
    "todo_list":                    {"page": 1, "page_size": 20},
    "todo_get":                     {"id": "todo-abc123"},
    "todo_create":                  {"title": "买菜"},
    "todo_update":                  {"id": "todo-abc123", "title": "买菜（已更新）"},
    "todo_delete":                  {"id": "todo-abc123"},
    "todo_update_status":           {"id": "todo-abc123", "status": "done"},
    "todo_batch":                   {"action": "done", "ids": ["id1", "id2", "id3"]},
    "todo_group_list":              {},
    "todo_group_create":            {"name": "工作"},
    "todo_group_update":            {"id": "grp-abc123", "name": "工作（更新）"},
    "todo_group_delete":            {"id": "grp-abc123"},
    "notification_list":            {"page": 1, "page_size": 20},
    "notification_mark_read":       {"id": "notif-abc123"},
    "notification_mark_all_read":   {},
    "notification_unread_count":    {},
    "notification_delete":          {"id": "notif-abc123"},
    "schedule_list":                {"start_date": "2025-01-01", "end_date": "2025-12-31"},
    "schedule_get":                 {"id": "sched-abc123"},
    "schedule_create":              {"title": "团队会议", "start_time": "2025-06-01T09:00:00", "end_time": "2025-06-01T10:00:00"},
    "schedule_update":              {"id": "sched-abc123", "title": "团队会议（更新）"},
    "schedule_delete":              {"id": "sched-abc123"},
    "schedule_add_resource":        {"id": "sched-abc123", "resource_type": "todo", "resource_id": "todo-abc123"},
    "schedule_remove_resource":     {"id": "sched-abc123", "resource_id": "res-abc123"},
    "wallet_list":                  {},
    "wallet_get":                   {"id": "wallet-abc123"},
    "wallet_create":                {"name": "招商银行"},
    "wallet_update":                {"id": "wallet-abc123", "name": "招商银行（更新）"},
    "wallet_delete":                {"id": "wallet-abc123"},
    "budget_category_list":         {},
    "budget_category_create":       {"name": "餐饮", "type": "expense"},
    "budget_category_update":       {"id": "cat-abc123", "name": "餐饮（更新）"},
    "budget_category_delete":       {"id": "cat-abc123"},
    "transaction_list":             {"page": 1, "page_size": 20},
    "transaction_get":              {"id": "tx-abc123"},
    "transaction_create":           {"wallet_id": "wallet-abc123", "type": "expense", "amount": 50.0, "transaction_at": "2025-06-01T12:00:00"},
    "transaction_update":           {"id": "tx-abc123", "amount": 60.0},
    "transaction_delete":           {"id": "tx-abc123"},
    "budget_stats":                 {"start_date": "2025-01-01", "end_date": "2025-12-31"},
    "secret_list":                  {"page": 1, "page_size": 20},
    "secret_get":                   {"id": "secret-abc123"},
    "secret_get_value":             {"id": "secret-abc123"},
    "secret_create":                {"name": "OPENAI_API_KEY", "value": "sk-..."},
    "secret_update":                {"id": "secret-abc123", "name": "OPENAI_API_KEY_v2"},
    "secret_delete":                {"id": "secret-abc123"},
    "secret_audit_logs":            {"id": "secret-abc123", "page": 1, "page_size": 20},
}

# 典型响应体（按工具分组）
TYPICAL_RESPONSES: dict[str, str] = {
    "list": '{"code":0,"data":{"items":[{"id":"abc123","title":"示例项目","status":"active","created_at":"2025-01-01T00:00:00Z"},{"id":"def456","title":"第二项","status":"active","created_at":"2025-01-02T00:00:00Z"}],"total":2,"page":1,"page_size":20}}',
    "get":  '{"code":0,"data":{"id":"abc123","title":"示例项目","description":"项目描述","status":"active","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}}',
    "create": '{"code":0,"data":{"id":"new-abc123","created_at":"2025-06-01T00:00:00Z"}}',
    "update": '{"code":0,"data":{"id":"abc123","updated_at":"2025-06-01T00:00:00Z"}}',
    "delete": '{"code":0,"message":"删除成功"}',
    "status": '{"code":0,"data":{"status":"ok"}}',
    "count":  '{"code":0,"data":{"count":5}}',
    "stats":  '{"code":0,"data":{"total_income":10000.0,"total_expense":5000.0,"net":5000.0,"by_category":[{"name":"餐饮","amount":1000.0,"percentage":20.0}]}}',
}

def get_typical_response(tool_name: str) -> str:
    """为工具名选择合适的典型响应体。"""
    name = tool_name.lower()
    if any(name.endswith(sfx) for sfx in ("_list", "_logs", "_audit_logs")):
        return TYPICAL_RESPONSES["list"]
    if any(name.endswith(sfx) for sfx in ("_get", "_get_value", "_summary", "_profile", "_token", "_smtp_status")):
        return TYPICAL_RESPONSES["get"]
    if name.endswith("_create"):
        return TYPICAL_RESPONSES["create"]
    if name.endswith(("_update", "_update_status", "_batch", "_mark_read", "_mark_all_read", "_add_resource", "_remove_resource", "_regenerate_token", "_test_email", "_change_password")):
        return TYPICAL_RESPONSES["update"]
    if name.endswith("_delete"):
        return TYPICAL_RESPONSES["delete"]
    if "unread_count" in name:
        return TYPICAL_RESPONSES["count"]
    if "stats" in name or "budget_stats" in name:
        return TYPICAL_RESPONSES["stats"]
    return TYPICAL_RESPONSES["status"]

# ---------------------------------------------------------------------------
# 核心测算
# ---------------------------------------------------------------------------

@dataclass
class ToolTokenInfo:
    name: str
    schema_tokens: int          # tools/list 中该工具的 schema token 数
    call_input_tokens: int      # 一次典型 tool call（AI→服务端）的额外 input tokens
    call_output_tokens: int     # 服务端返回结果（注入 AI context）的 output tokens

@dataclass
class SessionEstimate:
    # 完整 HTTP 响应体：{"jsonrpc":"2.0","id":...,"result":{"tools":[...]}}（与 Gin 紧凑 JSON 一致）
    tools_list_tokens: int
    # 仅 result 内 tools 数组的紧凑 JSON，便于对比「纯工具定义」体量
    tools_inner_tokens: int
    # initialize / notifications / tools/list 请求 / ping 等小消息合计（不含 tools/list 大响应）
    handshake_small_tokens: int
    per_tool: list[ToolTokenInfo]
    total_tools: int
    source: str = "static"
    live_note: str = ""


def _compact_json(obj: object) -> str:
    return json.dumps(obj, ensure_ascii=False, separators=(",", ":"))


def build_jsonrpc_tools_list_response(tools: list[dict], *, rpc_id: int = 1) -> str:
    """与后端 writeResult 一致：完整 JSON-RPC tools/list 响应（紧凑、无缩进）。"""
    payload = {
        "jsonrpc": "2.0",
        "id": rpc_id,
        "result": {
            "tools": [
                {"name": t["name"], "description": t["description"], "inputSchema": t["inputSchema"]}
                for t in tools
            ],
        },
    }
    return _compact_json(payload)


def build_tools_inner_only_compact(tools: list[dict]) -> str:
    return _compact_json({
        "tools": [
            {"name": t["name"], "description": t["description"], "inputSchema": t["inputSchema"]}
            for t in tools
        ],
    })


def handshake_small_messages_tokens(ct: Callable[[str], int]) -> int:
    """典型 MCP 握手链中除 tools/list 大响应外的小 JSON-RPC 消息。"""
    msgs = [
        '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"client","version":"1"}}}',
        '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"ops-timer-mcp","version":"2.0.0"}}}',
        '{"jsonrpc":"2.0","method":"notifications/initialized"}',
        '{"jsonrpc":"2.0","id":2,"method":"tools/list"}',
        '{"jsonrpc":"2.0","id":3,"method":"ping"}',
        '{"jsonrpc":"2.0","id":3,"result":{}}',
    ]
    return sum(ct(m) for m in msgs)


def fetch_tools_list_live(
    base_url: str,
    token: str,
    mcp_path: str = "/mcp",
    timeout: float = 30.0,
) -> tuple[str, list[dict]]:
    """POST tools/list，返回 (原始响应体, tools 列表)。"""
    path = mcp_path if mcp_path.startswith("/") else "/" + mcp_path
    url = base_url.rstrip("/") + path
    body = json.dumps({"jsonrpc": "2.0", "method": "tools/list", "id": 1}, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=body,
        method="POST",
        headers={"Content-Type": "application/json", "X-API-Token": token},
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {e.code}: {err_body}") from e
    except urllib.error.URLError as e:
        raise RuntimeError(f"请求失败: {e}") from e

    data = json.loads(raw)
    if data.get("error"):
        raise RuntimeError(f"MCP error: {data['error']}")
    tools = data.get("result", {}).get("tools")
    if not isinstance(tools, list):
        raise RuntimeError("tools/list 响应中缺少 result.tools")
    return raw, tools


def estimate(
    ct: Callable[[str], int],
    *,
    tools: Optional[list[dict]] = None,
    tools_list_rpc_body: Optional[str] = None,
    source: str = "static",
    live_note: str = "",
) -> SessionEstimate:
    tools = tools if tools is not None else TOOLS
    if tools_list_rpc_body is None:
        tools_list_rpc_body = build_jsonrpc_tools_list_response(tools)
    tools_inner_body = build_tools_inner_only_compact(tools)

    tools_list_tokens = ct(tools_list_rpc_body)
    tools_inner_tokens = ct(tools_inner_body)
    handshake_small_tokens = handshake_small_messages_tokens(ct)

    per_tool: list[ToolTokenInfo] = []
    for tool in tools:
        name = tool["name"]

        tool_entry = json.dumps(
            {"name": name, "description": tool["description"], "inputSchema": tool["inputSchema"]},
            ensure_ascii=False,
        )
        schema_tokens = ct(tool_entry)

        args = TYPICAL_ARGS.get(name, {})
        call_input_tokens = ct(json.dumps(args, ensure_ascii=False))

        resp = get_typical_response(name)
        call_output_tokens = ct(resp)

        per_tool.append(
            ToolTokenInfo(
                name=name,
                schema_tokens=schema_tokens,
                call_input_tokens=call_input_tokens,
                call_output_tokens=call_output_tokens,
            )
        )

    return SessionEstimate(
        tools_list_tokens=tools_list_tokens,
        tools_inner_tokens=tools_inner_tokens,
        handshake_small_tokens=handshake_small_tokens,
        per_tool=per_tool,
        total_tools=len(tools),
        source=source,
        live_note=live_note,
    )

# ---------------------------------------------------------------------------
# 场景模拟
# ---------------------------------------------------------------------------

@dataclass
class Scenario:
    name: str
    description: str
    tool_calls: list[str]   # 按顺序调用的工具名

SCENARIOS: list[Scenario] = [
    Scenario(
        name="查看今日概览",
        description="AI 读取项目列表、待办列表和通知数",
        tool_calls=["project_list", "todo_list", "notification_unread_count"],
    ),
    Scenario(
        name="创建任务并关联计时",
        description="创建待办 → 创建计时单元 → 更新状态",
        tool_calls=["todo_create", "unit_create", "unit_update_status"],
    ),
    Scenario(
        name="记账（收入+支出）",
        description="查看钱包 → 新增收入 → 新增支出 → 查看统计",
        tool_calls=["wallet_list", "transaction_create", "transaction_create", "budget_stats"],
    ),
    Scenario(
        name="安排日程",
        description="查看日程 → 创建日程 → 关联待办资源",
        tool_calls=["schedule_list", "schedule_create", "schedule_add_resource"],
    ),
    Scenario(
        name="密钥管理",
        description="列出密钥 → 读取密钥值 → 审计日志",
        tool_calls=["secret_list", "secret_get_value", "secret_audit_logs"],
    ),
    Scenario(
        name="完整工作流（复杂场景）",
        description="项目+计时+待办+记账+日程，典型重度使用",
        tool_calls=[
            "project_list", "unit_list", "todo_list",
            "unit_create", "todo_create", "schedule_create",
            "transaction_create", "budget_stats", "notification_unread_count",
        ],
    ),
]

# ---------------------------------------------------------------------------
# 报告输出
# ---------------------------------------------------------------------------

def print_report(est: SessionEstimate, token_counter_label: str) -> None:
    print(f"\n{'═'*72}")
    print(f"  ops-timer MCP Token 消耗测算报告")
    print(f"  数据来源：{est.source}" + (f" — {est.live_note}" if est.live_note else ""))
    print(f"  Token 计数器：{token_counter_label}")
    print(f"{'═'*72}\n")

    # ── 1. tools/list 开销 ──────────────────────────────────────────────
    print(f"【1】tools/list 与握手阶段 Token 开销")
    print(f"    • 共 {est.total_tools} 个工具")
    print(f"    • 完整 JSON-RPC 响应（推荐作为主指标）：{est.tools_list_tokens:,} tokens")
    print(f"    • 仅 result.tools 紧凑 JSON：{est.tools_inner_tokens:,} tokens")
    print(f"    • 握手小消息（initialize / notifications / tools/list 请求 / ping 等）：")
    print(f"      {est.handshake_small_tokens:,} tokens")
    print(f"    • 说明：模型侧通常会把 tools/list 的完整响应计入上下文；")
    print(f"      静态快照与 Go 源码中 schema 若略有差异，请以 --live 实测为准。\n")

    # ── 2. 各工具 schema token ──────────────────────────────────────────
    print(f"【2】各工具 Schema Token 占比（按 schema token 降序）")
    print(f"    {'工具名':<35} {'schema':>8}  {'调用input':>10}  {'响应output':>11}")
    print(f"    {'-'*35} {'-'*8}  {'-'*10}  {'-'*11}")
    sorted_tools = sorted(est.per_tool, key=lambda t: t.schema_tokens, reverse=True)
    for t in sorted_tools:
        print(f"    {t.name:<35} {t.schema_tokens:>8,}  {t.call_input_tokens:>10,}  {t.call_output_tokens:>11,}")

    total_schema = sum(t.schema_tokens for t in est.per_tool)
    avg_schema   = total_schema / len(est.per_tool) if est.per_tool else 0
    print(f"    {'合计/平均':<35} {total_schema:>8,}  {'':>10}  {'':>11}")
    print(f"    平均每工具 schema token：{avg_schema:.1f}\n")

    # ── 3. 场景模拟 ──────────────────────────────────────────────────────
    print(f"【3】典型场景 Token 消耗估算")
    print(f"    注：每个场景均包含 tools/list 的初始 input token 开销。")
    print(f"    公式：total_input = tools_list + Σ(call_input)，")
    print(f"          total_output = Σ(response_output)\n")
    tool_map = {t.name: t for t in est.per_tool}

    for sc in SCENARIOS:
        calls_input  = sum(tool_map[n].call_input_tokens  for n in sc.tool_calls if n in tool_map)
        calls_output = sum(tool_map[n].call_output_tokens for n in sc.tool_calls if n in tool_map)
        total_input  = est.tools_list_tokens + calls_input
        total_output = calls_output
        grand_total  = total_input + total_output

        print(f"    ▶ {sc.name}")
        print(f"      {sc.description}")
        print(f"      调用序列：{' → '.join(sc.tool_calls)}")
        print(f"      tools/list input : {est.tools_list_tokens:>7,} tokens")
        print(f"      工具调用 input   : {calls_input:>7,} tokens")
        print(f"      服务端响应 output: {calls_output:>7,} tokens")
        print(f"      ┌─────────────────────────────────────────")
        print(f"      │ 合计 input  : {total_input:>7,} tokens")
        print(f"      │ 合计 output : {total_output:>7,} tokens")
        print(f"      │ 总计        : {grand_total:>7,} tokens")
        print(f"      └─────────────────────────────────────────\n")

    # ── 4. 费用估算参考 ──────────────────────────────────────────────────
    print(f"【4】费用估算参考（以「完整工作流」场景为例）")
    sc_full = next(s for s in SCENARIOS if s.name == "完整工作流（复杂场景）")
    calls_input  = sum(tool_map[n].call_input_tokens  for n in sc_full.tool_calls if n in tool_map)
    calls_output = sum(tool_map[n].call_output_tokens for n in sc_full.tool_calls if n in tool_map)
    full_input   = est.tools_list_tokens + calls_input
    full_output  = calls_output

    # 价格表（$/1M tokens，2025 Q1 参考价格）
    PRICES = [
        ("Claude Sonnet 4.5",  3.0,  15.0),
        ("Claude Opus 4.6",   15.0,  75.0),
        ("Claude Haiku 4.5",   0.8,   4.0),
        ("GPT-4o",             2.5,  10.0),
        ("GPT-4o-mini",        0.15,  0.6),
    ]
    print(f"    场景 token 量：input={full_input:,}  output={full_output:,}\n")
    print(f"    {'模型':<22} {'input 费用(USD)':>16}  {'output 费用(USD)':>16}  {'合计(USD)':>10}")
    print(f"    {'-'*22} {'-'*16}  {'-'*16}  {'-'*10}")
    for model, pin, pout in PRICES:
        cost_in  = full_input  / 1_000_000 * pin
        cost_out = full_output / 1_000_000 * pout
        print(f"    {model:<22} {cost_in:>16.6f}  {cost_out:>16.6f}  {cost_in+cost_out:>10.6f}")
    print()

    # ── 5. 关键结论 ──────────────────────────────────────────────────────
    print(f"【5】关键结论与建议")
    print(f"    • tools/list 响应体约 {est.tools_list_tokens:,} tokens，是每次会话的固定开销。")
    print(f"      由于 {est.total_tools} 个工具的 schema 全量传输，建议 AI 客户端开启工具缓存。")
    print(f"    • 平均每工具在 tools/list 中约占 {avg_schema:.0f} tokens（单条工具定义）。")
    print(f"    • 响应体 output tokens 通常远小于 input，主要受限于列表大小。")
    print(f"    • 若 AI 支持工具 schema 缓存（如 Claude 的 prompt caching），")
    print(f"      tools/list 的 {est.tools_list_tokens:,} tokens 可在多轮对话中被缓存复用，")
    print(f"      实际费用可降低 60-80%%。")
    print(f"\n{'═'*72}\n")


def build_json_output(est: SessionEstimate, token_counter_label: str) -> dict:
    output: dict = {
        "token_counter": token_counter_label,
        "source": est.source,
        "live_note": est.live_note,
        "tools_list_tokens": est.tools_list_tokens,
        "tools_inner_tokens": est.tools_inner_tokens,
        "handshake_small_tokens": est.handshake_small_tokens,
        "total_tools": est.total_tools,
        "per_tool": [
            {
                "name": t.name,
                "schema_tokens": t.schema_tokens,
                "call_input_tokens": t.call_input_tokens,
                "call_output_tokens": t.call_output_tokens,
            }
            for t in est.per_tool
        ],
        "scenarios": [],
    }
    tool_map = {t.name: t for t in est.per_tool}
    for sc in SCENARIOS:
        calls_input  = sum(tool_map[n].call_input_tokens  for n in sc.tool_calls if n in tool_map)
        calls_output = sum(tool_map[n].call_output_tokens for n in sc.tool_calls if n in tool_map)
        output["scenarios"].append({
            "name": sc.name,
            "description": sc.description,
            "tool_calls": sc.tool_calls,
            "tools_list_input": est.tools_list_tokens,
            "calls_input": calls_input,
            "calls_output": calls_output,
            "total_input": est.tools_list_tokens + calls_input,
            "total_output": calls_output,
            "grand_total": est.tools_list_tokens + calls_input + calls_output,
        })
    return output


def main() -> None:
    p = argparse.ArgumentParser(description="ops-timer MCP token 消耗测算")
    p.add_argument(
        "--encoding",
        choices=("cl100k_base", "o200k_base"),
        default="cl100k_base",
        help="tiktoken 编码：cl100k_base 偏 GPT-4 经典；o200k_base 更接近 GPT-4o",
    )
    p.add_argument("--live", metavar="BASE_URL", help="从运行中的后端拉取 tools/list，例如 http://127.0.0.1:8080")
    p.add_argument("--mcp-path", default="/mcp", help="MCP 路径，默认 /mcp")
    p.add_argument(
        "--token",
        default="",
        help="X-API-Token；也可设环境变量 API_TOKEN 或 MCP_API_TOKEN",
    )
    p.add_argument("-o", "--output", default="mcp_token_estimate.json", help="JSON 输出路径")
    p.add_argument("-q", "--quiet", action="store_true", help="不打印报告，只写 JSON")
    args = p.parse_args()

    ct, tlabel = make_token_counter(args.encoding)

    live_note = ""
    tools: Optional[list[dict]] = None
    rpc_body: Optional[str] = None
    source = "static"

    if args.live:
        token = (args.token or os.environ.get("API_TOKEN") or os.environ.get("MCP_API_TOKEN") or "").strip()
        if not token:
            print("错误：使用 --live 时必须提供 --token 或设置环境变量 API_TOKEN / MCP_API_TOKEN", file=sys.stderr)
            sys.exit(2)
        try:
            rpc_body, tools = fetch_tools_list_live(args.live, token, mcp_path=args.mcp_path)
        except Exception as e:
            print(f"拉取 tools/list 失败: {e}", file=sys.stderr)
            sys.exit(1)
        source = "live"
        live_note = f"{args.live.rstrip('/')}{args.mcp_path}"

    est = estimate(ct, tools=tools, tools_list_rpc_body=rpc_body, source=source, live_note=live_note)

    if not args.quiet:
        print_report(est, tlabel)

    out = build_json_output(est, tlabel)
    with open(args.output, "w", encoding="utf-8") as f:
        json.dump(out, f, ensure_ascii=False, indent=2)
    if not args.quiet:
        print(f"  原始数据已保存至：{args.output}\n")


if __name__ == "__main__":
    main()
