package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"
)

func draftError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrNoteDraftNotFound) || errors.Is(err, service.ErrNoteNotFound) || errors.Is(err, service.ErrNoteGroupNotFound) {
		response.NotFound(c, err.Error())
	} else if errors.Is(err, service.ErrNoteDraftIncomplete) {
		response.BadRequest(c, err.Error(), nil)
	} else {
		response.InternalError(c, "草稿操作失败，请重试")
	}
}

func (h *NoteHandler) ListDrafts(c *gin.Context) {
	drafts, err := h.noteService.ListDrafts()
	if err != nil {
		draftError(c, err)
		return
	}
	response.Success(c, drafts)
}

func (h *NoteHandler) SaveDraft(c *gin.Context) {
	var req dto.SaveNoteDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "草稿参数无效", nil)
		return
	}
	draft, err := h.noteService.SaveDraft(c.Param("id"), &req)
	if err != nil {
		draftError(c, err)
		return
	}
	if c.Param("id") == "" {
		response.Created(c, draft)
	} else {
		response.Success(c, draft)
	}
}

func (h *NoteHandler) DeleteDraft(c *gin.Context) {
	if err := h.noteService.DeleteDraft(c.Param("id")); err != nil {
		draftError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *NoteHandler) PublishDraft(c *gin.Context) {
	var req dto.SaveNoteDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "笔记参数无效", nil)
		return
	}
	note, err := h.noteService.PublishDraft(c.Param("id"), &req)
	if err != nil {
		draftError(c, err)
		return
	}
	response.Success(c, note)
}
