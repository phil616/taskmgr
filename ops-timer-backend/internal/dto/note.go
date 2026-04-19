package dto

import "time"

type CreateNoteRequest struct {
	GroupID *string  `json:"group_id"`
	Title   string   `json:"title" binding:"required,max=256"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type UpdateNoteRequest struct {
	GroupID *string   `json:"group_id"`
	Title   *string   `json:"title" binding:"omitempty,max=256"`
	Content *string   `json:"content"`
	Tags    *[]string `json:"tags"`
}

type NoteResponse struct {
	ID        string    `json:"id"`
	GroupID   *string   `json:"group_id"`
	GroupName string    `json:"group_name,omitempty"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NoteQueryParams struct {
	GroupID  string `form:"group_id"`
	Tag      string `form:"tag"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

type NoteSearchQueryParams struct {
	Query    string `form:"q" binding:"required"`
	GroupID  string `form:"group_id"`
	Tag      string `form:"tag"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

type CreateNoteGroupRequest struct {
	Name      string `json:"name" binding:"required,max=64"`
	Color     string `json:"color"`
	SortOrder *int   `json:"sort_order"`
}

type UpdateNoteGroupRequest struct {
	Name      *string `json:"name" binding:"omitempty,max=64"`
	Color     *string `json:"color"`
	SortOrder *int    `json:"sort_order"`
}

type NoteGroupResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	SortOrder int       `json:"sort_order"`
	NoteCount int64     `json:"note_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
