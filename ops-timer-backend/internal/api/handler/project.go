package handler

import (
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	projectService *service.ProjectService
	unitService    *service.UnitService
}

func NewProjectHandler(projectService *service.ProjectService, unitService *service.UnitService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, unitService: unitService}
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	project, err := h.projectService.Create(&req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Created(c, project)
}

func (h *ProjectHandler) Get(c *gin.Context) {
	id := c.Param("id")
	project, err := h.projectService.GetByID(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, project)
}

func (h *ProjectHandler) List(c *gin.Context) {
	var params dto.ProjectQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	projects, total, err := h.projectService.List(&params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMeta(c, projects, &response.Meta{
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: response.CalculateTotalPages(total, params.PageSize),
	})
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	project, err := h.projectService.Update(id, &req)
	if err != nil {
		if err == service.ErrProjectNotFound {
			response.NotFound(c, err.Error())
		} else {
			response.BusinessError(c, err.Error())
		}
		return
	}
	response.Success(c, project)
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.projectService.Delete(id); err != nil {
		if err == service.ErrProjectNotFound {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}
	response.NoContent(c)
}

func (h *ProjectHandler) GetUnits(c *gin.Context) {
	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	units, total, err := h.unitService.ListByProject(id, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMeta(c, units, &response.Meta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: response.CalculateTotalPages(total, pageSize),
	})
}

func (h *ProjectHandler) GetBudgetStats(c *gin.Context) {
	id := c.Param("id")
	stats, err := h.projectService.GetBudgetStats(id)
	if err != nil {
		if err == service.ErrProjectNotFound {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}
	response.Success(c, stats)
}
