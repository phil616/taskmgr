package handler

import (
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ScheduleHandler struct {
	scheduleService *service.ScheduleService
}

func NewScheduleHandler(scheduleService *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleService: scheduleService}
}

// Create POST /schedules
func (h *ScheduleHandler) Create(c *gin.Context) {
	var req dto.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	schedule, err := h.scheduleService.Create(&req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Created(c, schedule)
}

// Get GET /schedules/:id
func (h *ScheduleHandler) Get(c *gin.Context) {
	id := c.Param("id")
	schedule, err := h.scheduleService.GetByID(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, schedule)
}

// List GET /schedules
func (h *ScheduleHandler) List(c *gin.Context) {
	var params dto.ScheduleQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	schedules, total, err := h.scheduleService.List(&params)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.SuccessWithMeta(c, schedules, &response.Meta{
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: response.CalculateTotalPages(total, params.PageSize),
	})
}

// Update PUT /schedules/:id
func (h *ScheduleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	schedule, err := h.scheduleService.Update(id, &req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Success(c, schedule)
}

// Delete DELETE /schedules/:id
func (h *ScheduleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.scheduleService.Delete(id); err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.NoContent(c)
}

// AddResource POST /schedules/:id/resources
func (h *ScheduleHandler) AddResource(c *gin.Context) {
	scheduleID := c.Param("id")
	var req dto.AddScheduleResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	res, err := h.scheduleService.AddResource(scheduleID, &req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Created(c, res)
}

// RemoveResource DELETE /schedules/:id/resources/:resource_id
func (h *ScheduleHandler) RemoveResource(c *gin.Context) {
	scheduleID := c.Param("id")
	resourceID := c.Param("resource_id")
	if err := h.scheduleService.RemoveResource(scheduleID, resourceID); err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.NoContent(c)
}
