package handler

import (
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type SecretHandler struct {
	secretService *service.SecretService
}

func NewSecretHandler(secretService *service.SecretService) *SecretHandler {
	return &SecretHandler{secretService: secretService}
}

func (h *SecretHandler) auditCtx(c *gin.Context) *service.AuditContext {
	return &service.AuditContext{
		UserID:    c.GetString("user_id"),
		Username:  c.GetString("username"),
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
}

func (h *SecretHandler) Create(c *gin.Context) {
	var req dto.CreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	secret, err := h.secretService.Create(&req, h.auditCtx(c))
	if err != nil {
		if err == service.ErrSecretNameDuplicate {
			response.BusinessError(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}
	response.Created(c, secret)
}

func (h *SecretHandler) Get(c *gin.Context) {
	id := c.Param("id")
	secret, err := h.secretService.GetByID(id, h.auditCtx(c))
	if err != nil {
		if err == service.ErrSecretNotFound {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}
	response.Success(c, secret)
}

func (h *SecretHandler) GetValue(c *gin.Context) {
	id := c.Param("id")
	secret, err := h.secretService.GetValue(id, h.auditCtx(c))
	if err != nil {
		if err == service.ErrSecretNotFound {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}
	response.Success(c, gin.H{"id": secret.ID, "name": secret.Name, "value": secret.Value})
}

func (h *SecretHandler) List(c *gin.Context) {
	var params dto.SecretQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	secrets, total, err := h.secretService.List(&params, h.auditCtx(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMeta(c, secrets, &response.Meta{
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: response.CalculateTotalPages(total, params.PageSize),
	})
}

func (h *SecretHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	secret, err := h.secretService.Update(id, &req, h.auditCtx(c))
	if err != nil {
		switch err {
		case service.ErrSecretNotFound:
			response.NotFound(c, err.Error())
		case service.ErrSecretNameDuplicate:
			response.BusinessError(c, err.Error())
		default:
			response.InternalError(c, err.Error())
		}
		return
	}
	response.Success(c, secret)
}

func (h *SecretHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.secretService.Delete(id, h.auditCtx(c)); err != nil {
		if err == service.ErrSecretNotFound {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}
	response.NoContent(c)
}

func (h *SecretHandler) AuditLogs(c *gin.Context) {
	var params dto.SecretAuditQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	secretID := c.Param("id")
	if secretID != "" {
		params.SecretID = secretID
	}

	logs, total, err := h.secretService.ListAuditLogs(&params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMeta(c, logs, &response.Meta{
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: response.CalculateTotalPages(total, params.PageSize),
	})
}
