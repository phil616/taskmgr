package dto

import "time"

type CreateTodoRequest struct {
	GroupID     *string  `json:"group_id"`
	Title       string   `json:"title" binding:"required,max=256"`
	Description string   `json:"description"`
	Status      string   `json:"status" binding:"omitempty,oneof=pending in_progress done cancelled"`
	Priority    string   `json:"priority" binding:"omitempty,oneof=low normal high critical"`
	Importance  *float64 `json:"importance" binding:"omitempty,gte=0,lte=1"`
	Difficulty  *float64 `json:"difficulty" binding:"omitempty,gte=0,lte=1"`
	DueDate     *string  `json:"due_date"`
	SortOrder   *int     `json:"sort_order"`
}

type UpdateTodoRequest struct {
	GroupID     *string  `json:"group_id"`
	Title       string   `json:"title" binding:"omitempty,max=256"`
	Description *string  `json:"description"`
	Status      string   `json:"status" binding:"omitempty,oneof=pending in_progress done cancelled"`
	Priority    string   `json:"priority" binding:"omitempty,oneof=low normal high critical"`
	Importance  *float64 `json:"importance" binding:"omitempty,gte=0,lte=1"`
	Difficulty  *float64 `json:"difficulty" binding:"omitempty,gte=0,lte=1"`
	DueDate     *string  `json:"due_date"`
	SortOrder   *int     `json:"sort_order"`
}

type UpdateTodoStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending in_progress done cancelled"`
}

type BatchTodoRequest struct {
	Action string   `json:"action" binding:"required,oneof=complete delete"`
	IDs    []string `json:"ids" binding:"required,min=1"`
}

type TodoResponse struct {
	ID          string     `json:"id"`
	GroupID     *string    `json:"group_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	Importance  float64    `json:"importance"`
	Difficulty  float64    `json:"difficulty"`
	DueDate     *time.Time `json:"due_date"`
	SortOrder   int        `json:"sort_order"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TodoQueryParams struct {
	Status   string `form:"status"`
	Priority string `form:"priority"`
	GroupID  string `form:"group_id"`
	DueDate  string `form:"due_date"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

type CreateTodoGroupRequest struct {
	Name      string `json:"name" binding:"required,max=64"`
	Color     string `json:"color"`
	SortOrder *int   `json:"sort_order"`
}

type UpdateTodoGroupRequest struct {
	Name      string  `json:"name" binding:"omitempty,max=64"`
	Color     *string `json:"color"`
	SortOrder *int    `json:"sort_order"`
}

type TodoGroupResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	SortOrder int       `json:"sort_order"`
	TodoCount int64     `json:"todo_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
