package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/jsonschema-go/jsonschema"
)

// ──────────────────────────────────────────────
//  MCP 工具定义（inputSchema 遵循 JSON Schema）
// ──────────────────────────────────────────────

type mcpToolDef struct {
	schema      *jsonschema.Schema
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema gin.H  `json:"inputSchema"`
}

// Immutable definitions and schema pointers are shared by stateless requests.
// SDK SchemaCache can then reuse exactly 80 input schemas and one output schema.
var registeredMCPTools = func() []mcpToolDef {
	tools := buildMCPTools()
	for i := range tools {
		tools[i].schema = mustMCPSchema(tools[i].InputSchema)
	}
	return tools
}()

func allTools() []mcpToolDef { return registeredMCPTools }
func mustMCPSchema(value any) *jsonschema.Schema {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(raw, &schema); err != nil {
		panic(err)
	}
	return &schema
}

func buildMCPTools() []mcpToolDef {
	str := func(desc string) gin.H { return gin.H{"type": "string", "description": desc} }
	num := func(desc string) gin.H { return gin.H{"type": "number", "description": desc} }
	boo := func(desc string) gin.H { return gin.H{"type": "boolean", "description": desc} }
	arr := func(item gin.H, desc string) gin.H {
		return gin.H{"type": "array", "items": item, "description": desc}
	}

	// ── 通用工具 ──────────────────────────────────────────────────────
	generic := mcpToolDef{
		Name:        mcpBackendRequest,
		Description: "调用任务管理器后端 HTTP API（支持 GET/POST/PUT/PATCH/DELETE）。当其他专用工具不满足需求时使用此通用工具。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"method":  str("HTTP 方法：GET/POST/PUT/PATCH/DELETE"),
				"path":    str("接口路径，必须以 /api/v1/ 或 /health 开头"),
				"query":   gin.H{"type": "object", "description": "查询参数对象（可选）"},
				"body":    gin.H{"description": "JSON 请求体（可选）"},
				"headers": gin.H{"type": "object", "additionalProperties": gin.H{"type": "string"}, "description": "额外请求头（可选）"},
			},
			"required": []string{"method", "path"},
		},
	}

	// ── 认证工具 ────────────────────────────────────────────────────────
	authGetProfile := mcpToolDef{
		Name:        "auth_get_profile",
		Description: "获取当前用户的个人资料信息（用户名、邮箱、昵称等）。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}
	authUpdateProfile := mcpToolDef{
		Name:        "auth_update_profile",
		Description: "更新当前用户的个人资料（昵称、邮箱等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"nickname": str("新昵称（可选）"),
				"email":    str("新邮箱（可选）"),
			},
		},
	}
	authChangePassword := mcpToolDef{
		Name:        "auth_change_password",
		Description: "修改当前用户的登录密码。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"old_password": str("旧密码（必填）"),
				"new_password": str("新密码（必填，至少 6 位）"),
			},
			"required": []string{"old_password", "new_password"},
		},
	}
	authGetToken := mcpToolDef{
		Name:        "auth_get_token",
		Description: "获取当前用户的 API Token 信息。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}
	authRegenerateToken := mcpToolDef{
		Name:        "auth_regenerate_token",
		Description: "重新生成当前用户的 API Token（旧 Token 将失效）。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}
	authTestEmail := mcpToolDef{
		Name:        "auth_test_email",
		Description: "发送测试邮件，验证 SMTP 邮件配置是否正常。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"to": str("收件人邮箱地址（必填）"),
			},
			"required": []string{"to"},
		},
	}
	authSMTPStatus := mcpToolDef{
		Name:        "auth_smtp_status",
		Description: "查询 SMTP 邮件服务的配置状态。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}

	// ── 计时单元工具（完整 CRUD + 特殊操作）────────────────────────────
	unitList := mcpToolDef{
		Name:        "unit_list",
		Description: "获取计时单元列表，支持按项目 ID、状态、类型等筛选。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"project_id": str("项目 ID（可选，筛选某项目下的单元）"),
				"type":       str("类型筛选：time_countdown/time_countup/count_countdown/count_countup（可选）"),
				"status":     str("状态筛选：active/paused/completed/archived（可选）"),
				"priority":   str("优先级筛选：low/normal/high/critical（可选）"),
				"page":       num("页码，默认 1"),
				"page_size":  num("每页数量，默认 20"),
			},
		},
	}
	unitGet := mcpToolDef{
		Name:        "unit_get",
		Description: "获取单个计时单元的详细信息。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("单元 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	unitCreate := mcpToolDef{
		Name:        "unit_create",
		Description: "创建一个新的计时单元。type 为 time_countdown（倒计时）或 time_countup（正计时）时使用 target_time/start_time；type 为 count_countdown（倒数计数）或 count_countup（正向计数）时使用 target_value/current_value/step。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"title":              str("单元标题（必填）"),
				"type":               str("类型（必填）：time_countdown（倒计时）/ time_countup（正计时）/ count_countdown（倒数计数）/ count_countup（正向计数）"),
				"description":        str("描述（可选）"),
				"project_id":         str("所属项目 ID（可选）"),
				"status":             str("状态：active/paused/completed/archived，默认 active"),
				"priority":           str("优先级：low/normal/high/critical，默认 normal"),
				"tags":               arr(gin.H{"type": "string"}, "标签数组（可选）"),
				"color":              str("颜色（HEX，可选）"),
				"target_time":        str("目标时间 ISO8601（time 类型使用，可选）"),
				"start_time":         str("开始时间 ISO8601（time 类型使用，可选）"),
				"display_unit":       str("时间显示单位：days/hours/minutes/seconds，默认 days"),
				"current_value":      num("当前值（count 类型使用，可选）"),
				"target_value":       num("目标值（count 类型使用，可选）"),
				"step":               num("步进值（count 类型使用，默认 1）"),
				"unit_label":         str("计数单位标签，如 次/个（可选）"),
				"remind_before_days": arr(gin.H{"type": "number"}, "提前提醒天数（可选）"),
				"remind_after_days":  arr(gin.H{"type": "number"}, "延后提醒天数（可选）"),
			},
			"required": []string{"title", "type"},
		},
	}
	unitUpdate := mcpToolDef{
		Name:        "unit_update",
		Description: "更新计时单元的基本信息（标题、描述、标签、颜色、优先级、目标时间等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":                 str("单元 ID（必填）"),
				"title":              str("新标题（可选）"),
				"description":        str("新描述（可选）"),
				"project_id":         str("新所属项目 ID（可选）"),
				"priority":           str("新优先级：low/normal/high/critical（可选）"),
				"tags":               arr(gin.H{"type": "string"}, "新标签数组（可选）"),
				"color":              str("新颜色 HEX（可选）"),
				"target_time":        str("新目标时间 ISO8601（可选）"),
				"start_time":         str("新开始时间 ISO8601（可选）"),
				"display_unit":       str("新显示单位（可选）"),
				"target_value":       num("新目标值（可选）"),
				"step":               num("新步进值（可选）"),
				"unit_label":         str("新计数单位标签（可选）"),
				"remind_before_days": arr(gin.H{"type": "number"}, "新提前提醒天数（可选）"),
				"remind_after_days":  arr(gin.H{"type": "number"}, "新延后提醒天数（可选）"),
			},
			"required": []string{"id"},
		},
	}
	unitDelete := mcpToolDef{
		Name:        "unit_delete",
		Description: "删除指定计时单元。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("单元 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	unitUpdateStatus := mcpToolDef{
		Name:        "unit_update_status",
		Description: "更新计时单元的状态（激活/暂停/完成/归档）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":     str("单元 ID（必填）"),
				"status": str("新状态：active（激活）/paused（暂停）/completed（完成）/archived（归档），必填"),
			},
			"required": []string{"id", "status"},
		},
	}
	unitStep := mcpToolDef{
		Name:        "unit_step",
		Description: "对计数型单元执行一次步进操作（+step 或 -step）。仅适用于 count_countdown / count_countup 类型。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":        str("单元 ID（必填）"),
				"direction": str("步进方向：up（+step）/ down（-step），默认 up"),
				"note":      str("操作备注（可选）"),
			},
			"required": []string{"id"},
		},
	}
	unitSetValue := mcpToolDef{
		Name:        "unit_set_value",
		Description: "直接设置计数型单元的当前值。仅适用于 count_countdown / count_countup 类型。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":    str("单元 ID（必填）"),
				"value": num("要设置的值（必填）"),
				"note":  str("操作备注（可选）"),
			},
			"required": []string{"id", "value"},
		},
	}
	unitLogs := mcpToolDef{
		Name:        "unit_logs",
		Description: "获取计时单元的操作日志/历史记录。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":        str("单元 ID（必填）"),
				"page":      num("页码，默认 1"),
				"page_size": num("每页数量，默认 20"),
			},
			"required": []string{"id"},
		},
	}
	unitSummary := mcpToolDef{
		Name:        "unit_summary",
		Description: "获取计时单元的汇总统计信息（各状态数量等）。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}

	// ── 项目工具（完整 CRUD）────────────────────────────────────────────
	projectList := mcpToolDef{
		Name:        "project_list",
		Description: "获取所有项目列表（含状态、描述、颜色等信息）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"status":    str("状态筛选：active/completed/archived（可选）"),
				"page":      num("页码，默认 1"),
				"page_size": num("每页数量，默认 20"),
			},
		},
	}
	projectGet := mcpToolDef{
		Name:        "project_get",
		Description: "获取单个项目的详细信息。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("项目 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	projectCreate := mcpToolDef{
		Name:        "project_create",
		Description: "创建新项目。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"title":       str("项目标题（必填）"),
				"description": str("项目描述（可选）"),
				"color":       str("项目颜色（HEX，可选）"),
				"icon":        str("项目图标（可选）"),
				"status":      str("状态：active/completed/archived，默认 active"),
				"max_budget":  num("项目最大预算金额（可选，0 表示不限制）"),
			},
			"required": []string{"title"},
		},
	}
	projectUpdate := mcpToolDef{
		Name:        "project_update",
		Description: "更新项目信息（标题、描述、颜色、预算等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":          str("项目 ID（必填）"),
				"title":       str("新标题（可选）"),
				"description": str("新描述（可选）"),
				"color":       str("新颜色（可选）"),
				"icon":        str("新图标（可选）"),
				"status":      str("新状态（可选）"),
				"max_budget":  num("新最大预算金额（可选）"),
			},
			"required": []string{"id"},
		},
	}
	projectDelete := mcpToolDef{
		Name:        "project_delete",
		Description: "删除指定项目。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("项目 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	projectUnits := mcpToolDef{
		Name:        "project_units",
		Description: "获取指定项目下的所有计时单元。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":        str("项目 ID（必填）"),
				"page":      num("页码，默认 1"),
				"page_size": num("每页数量，默认 20"),
			},
			"required": []string{"id"},
		},
	}
	projectBudgetStats := mcpToolDef{
		Name:        "project_budget_stats",
		Description: "获取指定项目的预算统计信息，包括总收入、总支出、净额、剩余预算、使用率和关联交易数。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("项目 ID（必填）")},
			"required":   []string{"id"},
		},
	}

	// ── 待办工具（完整 CRUD + 分组）──────────────────────────────────────
	todoList := mcpToolDef{
		Name:        "todo_list",
		Description: "查询待办事项列表，支持按分组、状态、优先级筛选。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"group_id":  str("分组 ID（可选）"),
				"status":    str("状态：pending/in_progress/done/cancelled（可选）"),
				"priority":  str("优先级：low/normal/high/critical（可选）"),
				"due_date":  str("截止日期筛选 YYYY-MM-DD（可选）"),
				"page":      num("页码，默认 1"),
				"page_size": num("每页数量，默认 20"),
			},
		},
	}
	todoGet := mcpToolDef{
		Name:        "todo_get",
		Description: "获取单条待办事项的详细信息。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("待办 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	todoCreate := mcpToolDef{
		Name:        "todo_create",
		Description: "创建一条新待办事项。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"title":       str("待办标题（必填）"),
				"description": str("详细描述（可选）"),
				"group_id":    str("分组 ID（可选）"),
				"priority":    str("优先级：low/normal/high/critical，默认 normal"),
				"importance":  map[string]interface{}{"type": "number", "minimum": 0, "maximum": 1, "description": "重要程度，0 最不重要，1 最重要；创建时默认 0.5"},
				"difficulty":  map[string]interface{}{"type": "number", "minimum": 0, "maximum": 1, "description": "困难程度，0 最不困难，1 最困难；创建时默认 0.5"},
				"due_date":    str("截止日期 YYYY-MM-DD（可选）"),
				"status":      str("初始状态：pending/in_progress/done/cancelled，默认 pending"),
			},
			"required": []string{"title"},
		},
	}
	todoUpdate := mcpToolDef{
		Name:        "todo_update",
		Description: "更新待办事项的信息（标题、描述、优先级、截止日期等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":          str("待办 ID（必填）"),
				"title":       str("新标题（可选）"),
				"description": str("新描述（可选）"),
				"group_id":    str("新分组 ID（可选）"),
				"priority":    str("新优先级（可选）"),
				"importance":  map[string]interface{}{"type": "number", "minimum": 0, "maximum": 1, "description": "重要程度，0 最不重要，1 最重要；创建时默认 0.5"},
				"difficulty":  map[string]interface{}{"type": "number", "minimum": 0, "maximum": 1, "description": "困难程度，0 最不困难，1 最困难；创建时默认 0.5"},
				"due_date":    str("新截止日期 YYYY-MM-DD（可选）"),
				"status":      str("新状态（可选）"),
			},
			"required": []string{"id"},
		},
	}
	todoDelete := mcpToolDef{
		Name:        "todo_delete",
		Description: "删除指定待办事项。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("待办 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	todoUpdateStatus := mcpToolDef{
		Name:        "todo_update_status",
		Description: "更新待办事项的完成状态。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":     str("待办 ID（必填）"),
				"status": str("新状态：pending/in_progress/done/cancelled（必填）"),
			},
			"required": []string{"id", "status"},
		},
	}
	todoBatch := mcpToolDef{
		Name:        "todo_batch",
		Description: "批量操作待办事项（批量完成、删除等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"action": str("批量操作类型：done/delete/cancel（必填）"),
				"ids":    arr(gin.H{"type": "string"}, "待办 ID 数组（必填）"),
			},
			"required": []string{"action", "ids"},
		},
	}
	todoGroupList := mcpToolDef{
		Name:        "todo_group_list",
		Description: "获取所有待办分组列表。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}
	todoGroupCreate := mcpToolDef{
		Name:        "todo_group_create",
		Description: "创建新的待办分组。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"name":  str("分组名称（必填）"),
				"color": str("分组颜色 HEX（可选）"),
				"icon":  str("分组图标（可选）"),
			},
			"required": []string{"name"},
		},
	}
	todoGroupUpdate := mcpToolDef{
		Name:        "todo_group_update",
		Description: "更新待办分组信息。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":    str("分组 ID（必填）"),
				"name":  str("新名称（可选）"),
				"color": str("新颜色（可选）"),
				"icon":  str("新图标（可选）"),
			},
			"required": []string{"id"},
		},
	}
	todoGroupDelete := mcpToolDef{
		Name:        "todo_group_delete",
		Description: "删除待办分组（不会删除分组内的待办事项，待办将移至无分组）。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("分组 ID（必填）")},
			"required":   []string{"id"},
		},
	}

	// ── 笔记工具（完整 CRUD + 分组 + 搜索）────────────────────────────────
	noteList := mcpToolDef{
		Name:        "note_list",
		Description: "查询 Markdown 笔记列表，支持按分组、标签和关键词筛选。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"group_id":  str("分组 ID（可选，传 none 表示未分组）"),
				"tag":       str("标签筛选（可选）"),
				"keyword":   str("关键词搜索（可选，会匹配标题、内容、标签和分组名）"),
				"page":      num("页码，默认 1"),
				"page_size": num("每页数量，默认 20"),
			},
		},
	}
	noteSearch := mcpToolDef{
		Name:        "note_search",
		Description: "全局搜索 Markdown 笔记内容，按标题、内容、标签和分组名匹配关键词。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"q":         str("搜索关键词（必填）"),
				"group_id":  str("分组 ID（可选，传 none 表示未分组）"),
				"tag":       str("标签筛选（可选）"),
				"page":      num("页码，默认 1"),
				"page_size": num("每页数量，默认 20"),
			},
			"required": []string{"q"},
		},
	}
	noteGet := mcpToolDef{
		Name:        "note_get",
		Description: "获取单篇 Markdown 笔记的详细信息。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("笔记 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	noteCreate := mcpToolDef{
		Name:        "note_create",
		Description: "创建一篇新的 Markdown 笔记。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"title":    str("笔记标题（必填）"),
				"content":  str("Markdown 内容（必填）"),
				"group_id": str("所属分组 ID（可选）"),
				"tags":     arr(gin.H{"type": "string"}, "标签数组（可选）"),
			},
			"required": []string{"title", "content"},
		},
	}
	noteUpdate := mcpToolDef{
		Name:        "note_update",
		Description: "更新笔记的标题、Markdown 内容、标签或所属分组。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":       str("笔记 ID（必填）"),
				"title":    str("新标题（可选）"),
				"content":  str("新 Markdown 内容（可选）"),
				"group_id": str("新分组 ID，传空字符串可取消分组（可选）"),
				"tags":     arr(gin.H{"type": "string"}, "新标签数组（可选）"),
			},
			"required": []string{"id"},
		},
	}
	noteDelete := mcpToolDef{
		Name:        "note_delete",
		Description: "删除指定笔记。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("笔记 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	noteGroupList := mcpToolDef{
		Name:        "note_group_list",
		Description: "获取所有笔记分组列表。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}
	noteGroupCreate := mcpToolDef{
		Name:        "note_group_create",
		Description: "创建新的笔记分组。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"name":       str("分组名称（必填）"),
				"color":      str("分组颜色 HEX（可选）"),
				"sort_order": num("排序值（可选）"),
			},
			"required": []string{"name"},
		},
	}
	noteGroupUpdate := mcpToolDef{
		Name:        "note_group_update",
		Description: "更新笔记分组信息。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":         str("分组 ID（必填）"),
				"name":       str("新名称（可选）"),
				"color":      str("新颜色（可选）"),
				"sort_order": num("新排序值（可选）"),
			},
			"required": []string{"id"},
		},
	}
	noteGroupDelete := mcpToolDef{
		Name:        "note_group_delete",
		Description: "删除笔记分组（不会删除分组内的笔记，笔记将移至未分组）。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("分组 ID（必填）")},
			"required":   []string{"id"},
		},
	}

	// ── 通知工具 ──────────────────────────────────────────────────────
	notificationList := mcpToolDef{
		Name:        "notification_list",
		Description: "获取通知消息列表，支持分页。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"page":      num("页码，默认 1"),
				"page_size": num("每页数量，默认 20"),
			},
		},
	}
	notificationMarkRead := mcpToolDef{
		Name:        "notification_mark_read",
		Description: "将单条通知标记为已读。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("通知 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	notificationMarkAllRead := mcpToolDef{
		Name:        "notification_mark_all_read",
		Description: "将所有未读通知标记为已读。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}
	notificationUnreadCount := mcpToolDef{
		Name:        "notification_unread_count",
		Description: "获取未读通知数量。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}
	notificationDelete := mcpToolDef{
		Name:        "notification_delete",
		Description: "删除指定通知。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("通知 ID（必填）")},
			"required":   []string{"id"},
		},
	}

	// ── 日程工具 ──────────────────────────────────────────────────────
	scheduleList := mcpToolDef{
		Name:        "schedule_list",
		Description: "查询日程列表，可按日期范围和状态筛选。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"start_date": str("开始日期 YYYY-MM-DD（可选）"),
				"end_date":   str("结束日期 YYYY-MM-DD（可选）"),
				"status":     str("状态：planned/in_progress/completed/cancelled（可选）"),
				"page":       num("页码，默认 1"),
				"page_size":  num("每页数量，默认 100"),
			},
		},
	}
	scheduleGet := mcpToolDef{
		Name:        "schedule_get",
		Description: "获取某条日程的详细信息（含关联资源）。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("日程 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	scheduleCreate := mcpToolDef{
		Name:        "schedule_create",
		Description: "创建一条新日程安排。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"title":           str("日程标题（必填）"),
				"description":     str("描述（可选）"),
				"start_time":      str("开始时间 ISO8601，如 2024-01-01T09:00:00（必填）"),
				"end_time":        str("结束时间 ISO8601（必填）"),
				"all_day":         boo("是否全天事件，默认 false"),
				"color":           str("颜色（HEX，可选）"),
				"location":        str("地点（可选）"),
				"status":          str("状态：planned/in_progress/completed/cancelled，默认 planned"),
				"recurrence_type": str("重复类型：none/daily/weekly/monthly/yearly，默认 none"),
				"tags":            arr(gin.H{"type": "string"}, "标签（可选）"),
			},
			"required": []string{"title", "start_time", "end_time"},
		},
	}
	scheduleUpdate := mcpToolDef{
		Name:        "schedule_update",
		Description: "更新已有日程的信息（标题、时间、状态等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":              str("日程 ID（必填）"),
				"title":           str("新标题（可选）"),
				"description":     str("新描述（可选）"),
				"start_time":      str("新开始时间 ISO8601（可选）"),
				"end_time":        str("新结束时间 ISO8601（可选）"),
				"all_day":         boo("是否全天（可选）"),
				"status":          str("新状态（可选）"),
				"location":        str("新地点（可选）"),
				"recurrence_type": str("新重复类型（可选）"),
				"color":           str("新颜色（可选）"),
				"tags":            arr(gin.H{"type": "string"}, "新标签（可选）"),
			},
			"required": []string{"id"},
		},
	}
	scheduleDelete := mcpToolDef{
		Name:        "schedule_delete",
		Description: "删除指定日程。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("日程 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	scheduleAddResource := mcpToolDef{
		Name:        "schedule_add_resource",
		Description: "为日程关联一个已有资源（项目、待办或计时单元）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":            str("日程 ID（必填）"),
				"resource_type": str("资源类型：project/todo/unit（必填）"),
				"resource_id":   str("资源 ID（必填）"),
				"note":          str("备注（可选）"),
			},
			"required": []string{"id", "resource_type", "resource_id"},
		},
	}
	scheduleRemoveResource := mcpToolDef{
		Name:        "schedule_remove_resource",
		Description: "从日程中移除一个关联资源。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":          str("日程 ID（必填）"),
				"resource_id": str("关联资源记录 ID（必填）"),
			},
			"required": []string{"id", "resource_id"},
		},
	}

	// ── 预算：钱包工具 ────────────────────────────────────────────────
	walletList := mcpToolDef{
		Name:        "wallet_list",
		Description: "获取所有钱包/账户列表（含余额、本月收支汇总）。",
		InputSchema: gin.H{"type": "object", "properties": gin.H{}},
	}
	walletGet := mcpToolDef{
		Name:        "wallet_get",
		Description: "获取单个钱包的详细信息。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("钱包 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	walletCreate := mcpToolDef{
		Name:        "wallet_create",
		Description: "创建新钱包/账户（银行卡、现金、支付宝、微信等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"name":        str("钱包名称（必填）"),
				"type":        str("类型：bank/cash/credit/alipay/wechat/other，默认 bank"),
				"balance":     num("初始余额，默认 0"),
				"currency":    str("货币代码，默认 CNY"),
				"color":       str("颜色（HEX，可选）"),
				"icon":        str("MDI 图标名（可选）"),
				"description": str("备注说明（可选）"),
				"is_default":  boo("是否设为默认钱包，默认 false"),
			},
			"required": []string{"name"},
		},
	}
	walletUpdate := mcpToolDef{
		Name:        "wallet_update",
		Description: "更新钱包信息（名称、颜色、默认设置等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":          str("钱包 ID（必填）"),
				"name":        str("新名称（可选）"),
				"type":        str("新类型（可选）"),
				"color":       str("新颜色（可选）"),
				"icon":        str("新图标（可选）"),
				"description": str("新备注（可选）"),
				"is_default":  boo("是否设为默认（可选）"),
			},
			"required": []string{"id"},
		},
	}
	walletDelete := mcpToolDef{
		Name:        "wallet_delete",
		Description: "删除指定钱包（软删除，已有交易记录不受影响）。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("钱包 ID（必填）")},
			"required":   []string{"id"},
		},
	}

	// ── 预算：分类工具 ────────────────────────────────────────────────
	categoryList := mcpToolDef{
		Name:        "budget_category_list",
		Description: "获取收支分类列表，可按类型筛选（income/expense/both）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"type": str("筛选类型：income/expense/both（不填则返回全部）"),
			},
		},
	}
	categoryCreate := mcpToolDef{
		Name:        "budget_category_create",
		Description: "创建自定义收支分类。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"name":  str("分类名称（必填）"),
				"type":  str("类型：income（收入）/expense（支出）/both（通用），必填"),
				"color": str("颜色 HEX（可选）"),
				"icon":  str("MDI 图标名，如 mdi-food（可选）"),
			},
			"required": []string{"name", "type"},
		},
	}
	categoryUpdate := mcpToolDef{
		Name:        "budget_category_update",
		Description: "更新分类信息（系统内置分类不可修改）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":    str("分类 ID（必填）"),
				"name":  str("新名称（可选）"),
				"type":  str("新类型（可选）"),
				"color": str("新颜色（可选）"),
				"icon":  str("新图标（可选）"),
			},
			"required": []string{"id"},
		},
	}
	categoryDelete := mcpToolDef{
		Name:        "budget_category_delete",
		Description: "删除自定义分类（系统内置分类不可删除，有关联交易的分类不可删除）。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("分类 ID（必填）")},
			"required":   []string{"id"},
		},
	}

	// ── 预算：收支记录工具 ────────────────────────────────────────────
	txList := mcpToolDef{
		Name:        "transaction_list",
		Description: "查询收支记录列表，支持按钱包、类型、日期范围、分类、项目等筛选。结果按交易时间倒序排列。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"wallet_id":   str("钱包 ID 筛选（可选）"),
				"category_id": str("分类 ID 筛选（可选）"),
				"project_id":  str("项目 ID 筛选，查看某项目关联的收支记录（可选）"),
				"type":        str("类型筛选：income/expense/transfer（可选）"),
				"start_date":  str("开始日期 YYYY-MM-DD（可选）"),
				"end_date":    str("结束日期 YYYY-MM-DD（可选）"),
				"min_amount":  num("最小金额筛选（可选）"),
				"max_amount":  num("最大金额筛选（可选）"),
				"keyword":     str("备注关键词搜索（可选）"),
				"page":        num("页码，默认 1"),
				"page_size":   num("每页数量，默认 20"),
			},
		},
	}
	txGet := mcpToolDef{
		Name:        "transaction_get",
		Description: "获取单条收支记录的详细信息。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("记录 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	txCreate := mcpToolDef{
		Name:        "transaction_create",
		Description: "新增一条收支记录（收入/支出/转账）。新增后自动更新对应钱包余额。可关联到某个项目。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"wallet_id":      str("钱包 ID（必填）"),
				"type":           str("类型：income（收入）/expense（支出）/transfer（转账），必填"),
				"amount":         num("金额，必须大于 0（必填）"),
				"category_id":    str("分类 ID（可选）"),
				"project_id":     str("关联项目 ID，将此收支记录归属到指定项目（可选）"),
				"note":           str("备注（可选）"),
				"tags":           arr(gin.H{"type": "string"}, "标签（可选）"),
				"transaction_at": str("交易时间 ISO8601，如 2024-01-15T12:30:00（必填）"),
				"to_wallet_id":   str("目标钱包 ID（仅 transfer 类型需要）"),
			},
			"required": []string{"wallet_id", "type", "amount", "transaction_at"},
		},
	}
	txUpdate := mcpToolDef{
		Name:        "transaction_update",
		Description: "更新收支记录的金额、分类、备注、标签、交易时间或关联项目。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":             str("记录 ID（必填）"),
				"category_id":    str("新分类 ID（可选）"),
				"project_id":     str("新关联项目 ID，传空字符串可取消关联（可选）"),
				"amount":         num("新金额（可选）"),
				"note":           str("新备注（可选）"),
				"tags":           arr(gin.H{"type": "string"}, "新标签（可选）"),
				"transaction_at": str("新交易时间 ISO8601（可选）"),
			},
			"required": []string{"id"},
		},
	}
	txDelete := mcpToolDef{
		Name:        "transaction_delete",
		Description: "删除收支记录（软删除），删除后自动回滚对应钱包余额。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("记录 ID（必填）")},
			"required":   []string{"id"},
		},
	}

	// ── 预算：统计工具 ────────────────────────────────────────────────
	budgetStats := mcpToolDef{
		Name:        "budget_stats",
		Description: "获取预算统计汇总，包括总收入、总支出、净余额和各分类消费占比。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"wallet_id":  str("钱包 ID（不填则统计所有钱包）"),
				"start_date": str("统计起始日期 YYYY-MM-DD（可选）"),
				"end_date":   str("统计截止日期 YYYY-MM-DD（可选）"),
			},
		},
	}

	// ── 密钥管理工具 ──────────────────────────────────────────────────
	secretList := mcpToolDef{
		Name:        "secret_list",
		Description: "获取密钥列表（不含密钥值），支持按名称、标签、项目筛选。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"name":       str("名称模糊搜索（可选）"),
				"tag":        str("标签筛选（可选）"),
				"project_id": str("项目 ID 筛选（可选）"),
				"page":       num("页码，默认 1"),
				"page_size":  num("每页数量，默认 20"),
			},
		},
	}
	secretGet := mcpToolDef{
		Name:        "secret_get",
		Description: "获取单个密钥的详细信息（含密钥值）。此操作会被审计记录。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("密钥 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	secretGetValue := mcpToolDef{
		Name:        "secret_get_value",
		Description: "仅获取密钥的值（明文）。此操作会被审计记录为 value_read。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("密钥 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	secretCreate := mcpToolDef{
		Name:        "secret_create",
		Description: "创建一个新密钥。密钥名称必须唯一。可关联到项目并添加标签。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"name":        str("密钥名称（必填，唯一）"),
				"value":       str("密钥值（必填，明文存储）"),
				"description": str("描述（可选）"),
				"tags":        arr(gin.H{"type": "string"}, "标签数组（可选）"),
				"project_id":  str("关联项目 ID（可选）"),
			},
			"required": []string{"name", "value"},
		},
	}
	secretUpdate := mcpToolDef{
		Name:        "secret_update",
		Description: "更新密钥的名称、值、描述、标签或关联项目。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":          str("密钥 ID（必填）"),
				"name":        str("新名称（可选）"),
				"value":       str("新密钥值（可选）"),
				"description": str("新描述（可选）"),
				"tags":        arr(gin.H{"type": "string"}, "新标签数组（可选）"),
				"project_id":  str("新关联项目 ID，传空字符串可取消关联（可选）"),
			},
			"required": []string{"id"},
		},
	}
	secretDelete := mcpToolDef{
		Name:        "secret_delete",
		Description: "删除指定密钥（软删除）。",
		InputSchema: gin.H{
			"type":       "object",
			"properties": gin.H{"id": str("密钥 ID（必填）")},
			"required":   []string{"id"},
		},
	}
	secretAuditLogs := mcpToolDef{
		Name:        "secret_audit_logs",
		Description: "查询密钥的审计日志，记录了每次密钥的访问历史（谁访问的、什么操作、IP 地址等）。",
		InputSchema: gin.H{
			"type": "object",
			"properties": gin.H{
				"id":        str("密钥 ID（可选，不填则查询所有密钥的审计日志）"),
				"action":    str("操作类型筛选：created/read/updated/deleted/value_read/listed（可选）"),
				"page":      num("页码，默认 1"),
				"page_size": num("每页数量，默认 20"),
			},
		},
	}

	return []mcpToolDef{
		generic,
		// Auth (7)
		authGetProfile, authUpdateProfile, authChangePassword, authGetToken, authRegenerateToken, authTestEmail, authSMTPStatus,
		// Units (10)
		unitList, unitGet, unitCreate, unitUpdate, unitDelete, unitUpdateStatus, unitStep, unitSetValue, unitLogs, unitSummary,
		// Projects (7)
		projectList, projectGet, projectCreate, projectUpdate, projectDelete, projectUnits, projectBudgetStats,
		// Todos (11)
		todoList, todoGet, todoCreate, todoUpdate, todoDelete, todoUpdateStatus, todoBatch,
		todoGroupList, todoGroupCreate, todoGroupUpdate, todoGroupDelete,
		// Notes (10)
		noteList, noteSearch, noteGet, noteCreate, noteUpdate, noteDelete,
		noteGroupList, noteGroupCreate, noteGroupUpdate, noteGroupDelete,
		// Notifications (5)
		notificationList, notificationMarkRead, notificationMarkAllRead, notificationUnreadCount, notificationDelete,
		// Schedules (7)
		scheduleList, scheduleGet, scheduleCreate, scheduleUpdate, scheduleDelete, scheduleAddResource, scheduleRemoveResource,
		// Wallets (5)
		walletList, walletGet, walletCreate, walletUpdate, walletDelete,
		// Categories (4)
		categoryList, categoryCreate, categoryUpdate, categoryDelete,
		// Transactions (5)
		txList, txGet, txCreate, txUpdate, txDelete,
		// Stats (1)
		budgetStats,
		// Secrets (7)
		secretList, secretGet, secretGetValue, secretCreate, secretUpdate, secretDelete, secretAuditLogs,
	}
}

// ──────────────────────────────────────────────
//  工具名 → HTTP 调用映射
// ──────────────────────────────────────────────

type httpCall struct {
	method string
	path   string
	query  map[string]any
	body   map[string]any
}

func toolToHTTP(name string, args map[string]any) (*httpCall, error) {
	s := func(k string) string { return toString(args[k]) }
	has := func(k string) bool { _, ok := args[k]; return ok }
	requireID := func() (string, error) {
		id := s("id")
		if id == "" || strings.ContainsAny(id, "/\\%?#") || strings.Contains(id, "..") {
			return "", fmt.Errorf("缺少必填参数 id")
		}
		return id, nil
	}

	switch name {

	// ── Auth ──
	case "auth_get_profile":
		return &httpCall{method: "GET", path: "/api/v1/auth/profile"}, nil
	case "auth_update_profile":
		return &httpCall{method: "PUT", path: "/api/v1/auth/profile", body: args}, nil
	case "auth_change_password":
		return &httpCall{method: "PUT", path: "/api/v1/auth/password", body: args}, nil
	case "auth_get_token":
		return &httpCall{method: "GET", path: "/api/v1/auth/token"}, nil
	case "auth_regenerate_token":
		return &httpCall{method: "POST", path: "/api/v1/auth/token/regenerate"}, nil
	case "auth_test_email":
		return &httpCall{method: "POST", path: "/api/v1/auth/test-email", body: args}, nil
	case "auth_smtp_status":
		return &httpCall{method: "GET", path: "/api/v1/auth/smtp-status"}, nil

	// ── Units ──
	case "unit_list":
		return &httpCall{method: "GET", path: "/api/v1/units", query: pickStrings(args, "project_id", "type", "status", "priority", "page", "page_size")}, nil
	case "unit_get":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/units/" + id}, nil
	case "unit_create":
		return &httpCall{method: "POST", path: "/api/v1/units", body: args}, nil
	case "unit_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/units/" + id, body: omitKeys(args, "id")}, nil
	case "unit_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/units/" + id}, nil
	case "unit_update_status":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PATCH", path: "/api/v1/units/" + id + "/status", body: pickKeys(args, "status")}, nil
	case "unit_step":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "POST", path: "/api/v1/units/" + id + "/step", body: omitKeys(args, "id")}, nil
	case "unit_set_value":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/units/" + id + "/value", body: omitKeys(args, "id")}, nil
	case "unit_logs":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/units/" + id + "/logs", query: pickStrings(args, "page", "page_size")}, nil
	case "unit_summary":
		return &httpCall{method: "GET", path: "/api/v1/units/summary"}, nil

	// ── Projects ──
	case "project_list":
		return &httpCall{method: "GET", path: "/api/v1/projects", query: pickStrings(args, "status", "page", "page_size")}, nil
	case "project_get":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/projects/" + id}, nil
	case "project_create":
		return &httpCall{method: "POST", path: "/api/v1/projects", body: args}, nil
	case "project_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/projects/" + id, body: omitKeys(args, "id")}, nil
	case "project_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/projects/" + id}, nil
	case "project_units":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/projects/" + id + "/units", query: pickStrings(args, "page", "page_size")}, nil
	case "project_budget_stats":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/projects/" + id + "/budget"}, nil

	// ── Todos ──
	case "todo_list":
		return &httpCall{method: "GET", path: "/api/v1/todos", query: pickStrings(args, "group_id", "status", "priority", "due_date", "page", "page_size")}, nil
	case "todo_get":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/todos/" + id}, nil
	case "todo_create":
		return &httpCall{method: "POST", path: "/api/v1/todos", body: args}, nil
	case "todo_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/todos/" + id, body: omitKeys(args, "id")}, nil
	case "todo_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/todos/" + id}, nil
	case "todo_update_status":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PATCH", path: "/api/v1/todos/" + id + "/status", body: pickKeys(args, "status")}, nil
	case "todo_batch":
		return &httpCall{method: "POST", path: "/api/v1/todos/batch", body: args}, nil

	// ── Todo Groups ──
	case "todo_group_list":
		return &httpCall{method: "GET", path: "/api/v1/todo-groups"}, nil
	case "todo_group_create":
		return &httpCall{method: "POST", path: "/api/v1/todo-groups", body: args}, nil
	case "todo_group_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/todo-groups/" + id, body: omitKeys(args, "id")}, nil
	case "todo_group_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/todo-groups/" + id}, nil

	// ── Notes ──
	case "note_list":
		return &httpCall{method: "GET", path: "/api/v1/notes", query: pickStrings(args, "group_id", "tag", "keyword", "page", "page_size")}, nil
	case "note_search":
		return &httpCall{method: "GET", path: "/api/v1/notes/search", query: pickStrings(args, "q", "group_id", "tag", "page", "page_size")}, nil
	case "note_get":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/notes/" + id}, nil
	case "note_create":
		return &httpCall{method: "POST", path: "/api/v1/notes", body: args}, nil
	case "note_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/notes/" + id, body: omitKeys(args, "id")}, nil
	case "note_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/notes/" + id}, nil

	// ── Note Groups ──
	case "note_group_list":
		return &httpCall{method: "GET", path: "/api/v1/note-groups"}, nil
	case "note_group_create":
		return &httpCall{method: "POST", path: "/api/v1/note-groups", body: args}, nil
	case "note_group_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/note-groups/" + id, body: omitKeys(args, "id")}, nil
	case "note_group_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/note-groups/" + id}, nil

	// ── Notifications ──
	case "notification_list":
		return &httpCall{method: "GET", path: "/api/v1/notifications", query: pickStrings(args, "page", "page_size")}, nil
	case "notification_mark_read":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PATCH", path: "/api/v1/notifications/" + id + "/read"}, nil
	case "notification_mark_all_read":
		return &httpCall{method: "POST", path: "/api/v1/notifications/read-all"}, nil
	case "notification_unread_count":
		return &httpCall{method: "GET", path: "/api/v1/notifications/unread-count"}, nil
	case "notification_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/notifications/" + id}, nil

	// ── Schedules ──
	case "schedule_list":
		return &httpCall{method: "GET", path: "/api/v1/schedules", query: pickStrings(args, "start_date", "end_date", "status", "page", "page_size")}, nil
	case "schedule_get":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/schedules/" + id}, nil
	case "schedule_create":
		return &httpCall{method: "POST", path: "/api/v1/schedules", body: omitKeys(args, "id")}, nil
	case "schedule_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/schedules/" + id, body: omitKeys(args, "id")}, nil
	case "schedule_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/schedules/" + id}, nil
	case "schedule_add_resource":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "POST", path: "/api/v1/schedules/" + id + "/resources", body: pickKeys(args, "resource_type", "resource_id", "note")}, nil
	case "schedule_remove_resource":
		id, rid := s("id"), s("resource_id")
		if id == "" || rid == "" {
			return nil, fmt.Errorf("缺少必填参数 id / resource_id")
		}
		return &httpCall{method: "DELETE", path: "/api/v1/schedules/" + id + "/resources/" + rid}, nil

	// ── Budget：Wallets ──
	case "wallet_list":
		return &httpCall{method: "GET", path: "/api/v1/wallets"}, nil
	case "wallet_get":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/wallets/" + id}, nil
	case "wallet_create":
		return &httpCall{method: "POST", path: "/api/v1/wallets", body: args}, nil
	case "wallet_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/wallets/" + id, body: omitKeys(args, "id")}, nil
	case "wallet_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/wallets/" + id}, nil

	// ── Budget：Categories ──
	case "budget_category_list":
		q := map[string]any{}
		if has("type") {
			q["type"] = s("type")
		}
		return &httpCall{method: "GET", path: "/api/v1/budget/categories", query: q}, nil
	case "budget_category_create":
		return &httpCall{method: "POST", path: "/api/v1/budget/categories", body: args}, nil
	case "budget_category_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/budget/categories/" + id, body: omitKeys(args, "id")}, nil
	case "budget_category_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/budget/categories/" + id}, nil

	// ── Budget：Transactions ──
	case "transaction_list":
		return &httpCall{method: "GET", path: "/api/v1/transactions", query: pickStrings(args, "wallet_id", "category_id", "project_id", "type", "start_date", "end_date", "min_amount", "max_amount", "keyword", "page", "page_size")}, nil
	case "transaction_get":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/transactions/" + id}, nil
	case "transaction_create":
		return &httpCall{method: "POST", path: "/api/v1/transactions", body: args}, nil
	case "transaction_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/transactions/" + id, body: omitKeys(args, "id")}, nil
	case "transaction_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/transactions/" + id}, nil

	// ── Budget：Stats ──
	case "budget_stats":
		return &httpCall{method: "GET", path: "/api/v1/budget/stats", query: pickStrings(args, "wallet_id", "start_date", "end_date")}, nil

	// ── Secrets ──
	case "secret_list":
		return &httpCall{method: "GET", path: "/api/v1/secrets", query: pickStrings(args, "name", "tag", "project_id", "page", "page_size")}, nil
	case "secret_get":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/secrets/" + id}, nil
	case "secret_get_value":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "GET", path: "/api/v1/secrets/" + id + "/value"}, nil
	case "secret_create":
		return &httpCall{method: "POST", path: "/api/v1/secrets", body: args}, nil
	case "secret_update":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "PUT", path: "/api/v1/secrets/" + id, body: omitKeys(args, "id")}, nil
	case "secret_delete":
		id, err := requireID()
		if err != nil {
			return nil, err
		}
		return &httpCall{method: "DELETE", path: "/api/v1/secrets/" + id}, nil
	case "secret_audit_logs":
		if has("id") && s("id") != "" {
			return &httpCall{method: "GET", path: "/api/v1/secrets/" + s("id") + "/audit-logs", query: pickStrings(args, "action", "page", "page_size")}, nil
		}
		return &httpCall{method: "GET", path: "/api/v1/secret-audit-logs", query: pickStrings(args, "secret_id", "action", "page", "page_size")}, nil

	default:
		return nil, fmt.Errorf("未知工具名: %s", name)
	}
}
