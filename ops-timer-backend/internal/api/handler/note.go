package handler

import (
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type NoteHandler struct {
	noteService *service.NoteService
}

func NewNoteHandler(noteService *service.NoteService) *NoteHandler {
	return &NoteHandler{noteService: noteService}
}

func (h *NoteHandler) Create(c *gin.Context) {
	var req dto.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	note, err := h.noteService.Create(&req)
	if err != nil {
		if err == service.ErrNoteGroupNotFound {
			response.NotFound(c, err.Error())
			return
		}
		response.BusinessError(c, err.Error())
		return
	}
	response.Created(c, note)
}

func (h *NoteHandler) Get(c *gin.Context) {
	note, err := h.noteService.GetByID(c.Param("id"))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, note)
}

func (h *NoteHandler) List(c *gin.Context) {
	var params dto.NoteQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	notes, total, err := h.noteService.List(&params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	page, pageSize := normalizePagination(params.Page, params.PageSize)
	response.SuccessWithMeta(c, notes, &response.Meta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: response.CalculateTotalPages(total, pageSize),
	})
}

func (h *NoteHandler) Search(c *gin.Context) {
	var params dto.NoteSearchQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	notes, total, err := h.noteService.Search(&params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	page, pageSize := normalizePagination(params.Page, params.PageSize)
	response.SuccessWithMeta(c, notes, &response.Meta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: response.CalculateTotalPages(total, pageSize),
	})
}

func (h *NoteHandler) Update(c *gin.Context) {
	var req dto.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	note, err := h.noteService.Update(c.Param("id"), &req)
	if err != nil {
		switch err {
		case service.ErrNoteNotFound, service.ErrNoteGroupNotFound:
			response.NotFound(c, err.Error())
		default:
			response.BusinessError(c, err.Error())
		}
		return
	}
	response.Success(c, note)
}

func (h *NoteHandler) Delete(c *gin.Context) {
	if err := h.noteService.Delete(c.Param("id")); err != nil {
		if err == service.ErrNoteNotFound {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.NoContent(c)
}

func (h *NoteHandler) CreateGroup(c *gin.Context) {
	var req dto.CreateNoteGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	group, err := h.noteService.CreateGroup(&req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Created(c, group)
}

func (h *NoteHandler) ListGroups(c *gin.Context) {
	groups, err := h.noteService.ListGroups()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, groups)
}

func (h *NoteHandler) UpdateGroup(c *gin.Context) {
	var req dto.UpdateNoteGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	group, err := h.noteService.UpdateGroup(c.Param("id"), &req)
	if err != nil {
		if err == service.ErrNoteGroupNotFound {
			response.NotFound(c, err.Error())
			return
		}
		response.BusinessError(c, err.Error())
		return
	}
	response.Success(c, group)
}

func (h *NoteHandler) DeleteGroup(c *gin.Context) {
	if err := h.noteService.DeleteGroup(c.Param("id")); err != nil {
		if err == service.ErrNoteGroupNotFound {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.NoContent(c)
}

func normalizePagination(page, pageSize int) (int, int) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	return page, pageSize
}
