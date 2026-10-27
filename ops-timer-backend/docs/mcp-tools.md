# MCP 工具注册表

由 `go generate ./internal/api/handler` 生成，测试自动校验。共 80 个注册工具，默认启用 74 个；OAuth 工具列表还受 scope 限制，通用代理不向 OAuth 客户端开放。

| 工具 | OAuth scope | 默认启用 | 说明 |
|---|---|---|---|
| `backend_request` | 不委托 | 否 | 调用任务管理器后端 HTTP API（支持 GET/POST/PUT/PATCH/DELETE）。当其他专用工具不满足需求时使用此通用工具。 |
| `auth_get_profile` | data:read | 是 | 获取当前用户的个人资料信息（用户名、邮箱、昵称等）。 |
| `auth_update_profile` | 不委托 | 是 | 更新当前用户的个人资料（昵称、邮箱等）。 |
| `auth_change_password` | 不委托 | 否 | 修改当前用户的登录密码。 |
| `auth_get_token` | 不委托 | 否 | 获取当前用户的 API Token 信息。 |
| `auth_regenerate_token` | 不委托 | 否 | 重新生成当前用户的 API Token（旧 Token 将失效）。 |
| `auth_test_email` | 不委托 | 是 | 发送测试邮件，验证 SMTP 邮件配置是否正常。 |
| `auth_smtp_status` | 不委托 | 是 | 查询 SMTP 邮件服务的配置状态。 |
| `unit_list` | data:read | 是 | 获取计时单元列表，支持按项目 ID、状态、类型等筛选。 |
| `unit_get` | data:read | 是 | 获取单个计时单元的详细信息。 |
| `unit_create` | data:write | 是 | 创建一个新的计时单元。type 为 time_countdown（倒计时）或 time_countup（正计时）时使用 target_time/start_time；type 为 count_countdown（倒数计数）或 count_countup（正向计数）时使用 target_value/current_value/step。 |
| `unit_update` | data:write | 是 | 更新计时单元的基本信息（标题、描述、标签、颜色、优先级、目标时间等）。 |
| `unit_delete` | data:write | 是 | 删除指定计时单元。 |
| `unit_update_status` | data:write | 是 | 更新计时单元的状态（激活/暂停/完成/归档）。 |
| `unit_step` | data:write | 是 | 对计数型单元执行一次步进操作（+step 或 -step）。仅适用于 count_countdown / count_countup 类型。 |
| `unit_set_value` | data:write | 是 | 直接设置计数型单元的当前值。仅适用于 count_countdown / count_countup 类型。 |
| `unit_logs` | data:read | 是 | 获取计时单元的操作日志/历史记录。 |
| `unit_summary` | data:read | 是 | 获取计时单元的汇总统计信息（各状态数量等）。 |
| `project_list` | data:read | 是 | 获取所有项目列表（含状态、描述、颜色等信息）。 |
| `project_get` | data:read | 是 | 获取单个项目的详细信息。 |
| `project_create` | data:write | 是 | 创建新项目。 |
| `project_update` | data:write | 是 | 更新项目信息（标题、描述、颜色、预算等）。 |
| `project_delete` | data:write | 是 | 删除指定项目。 |
| `project_units` | data:read | 是 | 获取指定项目下的所有计时单元。 |
| `project_budget_stats` | data:read | 是 | 获取指定项目的预算统计信息，包括总收入、总支出、净额、剩余预算、使用率和关联交易数。 |
| `todo_list` | data:read | 是 | 查询待办事项列表，支持按分组、状态、优先级筛选。 |
| `todo_get` | data:read | 是 | 获取单条待办事项的详细信息。 |
| `todo_create` | data:write | 是 | 创建一条新待办事项。 |
| `todo_update` | data:write | 是 | 更新待办事项的信息（标题、描述、优先级、截止日期等）。 |
| `todo_delete` | data:write | 是 | 删除指定待办事项。 |
| `todo_update_status` | data:write | 是 | 更新待办事项的完成状态。 |
| `todo_batch` | data:write | 是 | 批量操作待办事项（批量完成、删除等）。 |
| `todo_group_list` | data:read | 是 | 获取所有待办分组列表。 |
| `todo_group_create` | data:write | 是 | 创建新的待办分组。 |
| `todo_group_update` | data:write | 是 | 更新待办分组信息。 |
| `todo_group_delete` | data:write | 是 | 删除待办分组（不会删除分组内的待办事项，待办将移至无分组）。 |
| `note_list` | data:read | 是 | 查询 Markdown 笔记列表，支持按分组、标签和关键词筛选。 |
| `note_search` | data:read | 是 | 全局搜索 Markdown 笔记内容，按标题、内容、标签和分组名匹配关键词。 |
| `note_get` | data:read | 是 | 获取单篇 Markdown 笔记的详细信息。 |
| `note_create` | data:write | 是 | 创建一篇新的 Markdown 笔记。 |
| `note_update` | data:write | 是 | 更新笔记的标题、Markdown 内容、标签或所属分组。 |
| `note_delete` | data:write | 是 | 删除指定笔记。 |
| `note_group_list` | data:read | 是 | 获取所有笔记分组列表。 |
| `note_group_create` | data:write | 是 | 创建新的笔记分组。 |
| `note_group_update` | data:write | 是 | 更新笔记分组信息。 |
| `note_group_delete` | data:write | 是 | 删除笔记分组（不会删除分组内的笔记，笔记将移至未分组）。 |
| `notification_list` | data:read | 是 | 获取通知消息列表，支持分页。 |
| `notification_mark_read` | data:write | 是 | 将单条通知标记为已读。 |
| `notification_mark_all_read` | data:write | 是 | 将所有未读通知标记为已读。 |
| `notification_unread_count` | data:read | 是 | 获取未读通知数量。 |
| `notification_delete` | data:write | 是 | 删除指定通知。 |
| `schedule_list` | data:read | 是 | 查询日程列表，可按日期范围和状态筛选。 |
| `schedule_get` | data:read | 是 | 获取某条日程的详细信息（含关联资源）。 |
| `schedule_create` | data:write | 是 | 创建一条新日程安排。 |
| `schedule_update` | data:write | 是 | 更新已有日程的信息（标题、时间、状态等）。 |
| `schedule_delete` | data:write | 是 | 删除指定日程。 |
| `schedule_add_resource` | data:write | 是 | 为日程关联一个已有资源（项目、待办或计时单元）。 |
| `schedule_remove_resource` | data:write | 是 | 从日程中移除一个关联资源。 |
| `wallet_list` | data:read | 是 | 获取所有钱包/账户列表（含余额、本月收支汇总）。 |
| `wallet_get` | data:read | 是 | 获取单个钱包的详细信息。 |
| `wallet_create` | data:write | 是 | 创建新钱包/账户（银行卡、现金、支付宝、微信等）。 |
| `wallet_update` | data:write | 是 | 更新钱包信息（名称、颜色、默认设置等）。 |
| `wallet_delete` | data:write | 是 | 删除指定钱包（软删除，已有交易记录不受影响）。 |
| `budget_category_list` | data:read | 是 | 获取收支分类列表，可按类型筛选（income/expense/both）。 |
| `budget_category_create` | data:write | 是 | 创建自定义收支分类。 |
| `budget_category_update` | data:write | 是 | 更新分类信息（系统内置分类不可修改）。 |
| `budget_category_delete` | data:write | 是 | 删除自定义分类（系统内置分类不可删除，有关联交易的分类不可删除）。 |
| `transaction_list` | data:read | 是 | 查询收支记录列表，支持按钱包、类型、日期范围、分类、项目等筛选。结果按交易时间倒序排列。 |
| `transaction_get` | data:read | 是 | 获取单条收支记录的详细信息。 |
| `transaction_create` | data:write | 是 | 新增一条收支记录（收入/支出/转账）。新增后自动更新对应钱包余额。可关联到某个项目。 |
| `transaction_update` | data:write | 是 | 更新收支记录的金额、分类、备注、标签、交易时间或关联项目。 |
| `transaction_delete` | data:write | 是 | 删除收支记录（软删除），删除后自动回滚对应钱包余额。 |
| `budget_stats` | data:read | 是 | 获取预算统计汇总，包括总收入、总支出、净余额和各分类消费占比。 |
| `secret_list` | secrets:read | 是 | 获取密钥列表（不含密钥值），支持按名称、标签、项目筛选。 |
| `secret_get` | secrets:read | 否 | 获取单个密钥的详细信息（含密钥值）。此操作会被审计记录。 |
| `secret_get_value` | secrets:read | 否 | 仅获取密钥的值（明文）。此操作会被审计记录为 value_read。 |
| `secret_create` | secrets:write | 是 | 创建一个新密钥。密钥名称必须唯一。可关联到项目并添加标签。 |
| `secret_update` | secrets:write | 是 | 更新密钥的名称、值、描述、标签或关联项目。 |
| `secret_delete` | secrets:write | 是 | 删除指定密钥（软删除）。 |
| `secret_audit_logs` | secrets:read | 是 | 查询密钥的审计日志，记录了每次密钥的访问历史（谁访问的、什么操作、IP 地址等）。 |
