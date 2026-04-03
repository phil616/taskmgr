package service

import (
	"fmt"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/repository"
	"time"

	"github.com/google/uuid"
)

type ScheduleService struct {
	scheduleRepo *repository.ScheduleRepository
	projectRepo  *repository.ProjectRepository
	unitRepo     *repository.UnitRepository
	todoRepo     *repository.TodoRepository
}

func NewScheduleService(
	scheduleRepo *repository.ScheduleRepository,
	projectRepo *repository.ProjectRepository,
	unitRepo *repository.UnitRepository,
	todoRepo *repository.TodoRepository,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		projectRepo:  projectRepo,
		unitRepo:     unitRepo,
		todoRepo:     todoRepo,
	}
}

func (s *ScheduleService) Create(req *dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	startTime, err := parseTime(req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("start_time 格式错误: %w", err)
	}
	endTime, err := parseTime(req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("end_time 格式错误: %w", err)
	}
	if !endTime.After(startTime) {
		return nil, fmt.Errorf("end_time 必须晚于 start_time")
	}

	var recurrenceEnd *time.Time
	if req.RecurrenceEnd != nil && *req.RecurrenceEnd != "" {
		t, err := parseTime(*req.RecurrenceEnd)
		if err != nil {
			return nil, fmt.Errorf("recurrence_end 格式错误: %w", err)
		}
		recurrenceEnd = &t
	}

	recurrenceType := req.RecurrenceType
	if recurrenceType == "" {
		recurrenceType = model.RecurrenceNone
	}
	status := req.Status
	if status == "" {
		status = model.ScheduleStatusPlanned
	}

	schedule := &model.Schedule{
		ID:             uuid.New().String(),
		Title:          req.Title,
		Description:    req.Description,
		StartTime:      startTime,
		EndTime:        endTime,
		AllDay:         req.AllDay,
		Color:          req.Color,
		Location:       req.Location,
		Status:         status,
		RecurrenceType: recurrenceType,
		RecurrenceEnd:  recurrenceEnd,
		Tags:           req.Tags,
	}

	if err := s.scheduleRepo.Create(schedule); err != nil {
		return nil, err
	}
	return s.toResponse(schedule), nil
}

func (s *ScheduleService) GetByID(id string) (*dto.ScheduleResponse, error) {
	schedule, err := s.scheduleRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("日程不存在")
	}
	resp := s.toResponse(schedule)
	s.enrichResources(resp, schedule.Resources)
	return resp, nil
}

func (s *ScheduleService) Update(id string, req *dto.UpdateScheduleRequest) (*dto.ScheduleResponse, error) {
	schedule, err := s.scheduleRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("日程不存在")
	}

	if req.Title != "" {
		schedule.Title = req.Title
	}
	if req.Description != nil {
		schedule.Description = *req.Description
	}
	if req.StartTime != "" {
		t, err := parseTime(req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("start_time 格式错误: %w", err)
		}
		schedule.StartTime = t
	}
	if req.EndTime != "" {
		t, err := parseTime(req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("end_time 格式错误: %w", err)
		}
		schedule.EndTime = t
	}
	if !schedule.EndTime.After(schedule.StartTime) {
		return nil, fmt.Errorf("end_time 必须晚于 start_time")
	}
	if req.AllDay != nil {
		schedule.AllDay = *req.AllDay
	}
	if req.Color != nil {
		schedule.Color = *req.Color
	}
	if req.Location != nil {
		schedule.Location = *req.Location
	}
	if req.Status != "" {
		schedule.Status = req.Status
	}
	if req.RecurrenceType != "" {
		schedule.RecurrenceType = req.RecurrenceType
	}
	if req.RecurrenceEnd != nil {
		if *req.RecurrenceEnd == "" {
			schedule.RecurrenceEnd = nil
		} else {
			t, err := parseTime(*req.RecurrenceEnd)
			if err != nil {
				return nil, fmt.Errorf("recurrence_end 格式错误: %w", err)
			}
			schedule.RecurrenceEnd = &t
		}
	}
	if req.Tags != nil {
		schedule.Tags = req.Tags
	}

	if err := s.scheduleRepo.Update(schedule); err != nil {
		return nil, err
	}
	resp := s.toResponse(schedule)
	s.enrichResources(resp, schedule.Resources)
	return resp, nil
}

func (s *ScheduleService) Delete(id string) error {
	if _, err := s.scheduleRepo.GetByID(id); err != nil {
		return fmt.Errorf("日程不存在")
	}
	return s.scheduleRepo.Delete(id)
}

func (s *ScheduleService) List(params *dto.ScheduleQueryParams) ([]*dto.ScheduleResponse, int64, error) {
	var startDate, endDate time.Time

	if params.StartDate != "" {
		t, err := time.Parse("2006-01-02", params.StartDate)
		if err != nil {
			return nil, 0, fmt.Errorf("start_date 格式错误，请用 YYYY-MM-DD")
		}
		startDate = t
	}
	if params.EndDate != "" {
		t, err := time.Parse("2006-01-02", params.EndDate)
		if err != nil {
			return nil, 0, fmt.Errorf("end_date 格式错误，请用 YYYY-MM-DD")
		}
		// 包含当天结束
		endDate = t.Add(24 * time.Hour)
	}

	schedules, total, err := s.scheduleRepo.List(startDate, endDate, params.Status, params.Page, params.PageSize)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*dto.ScheduleResponse, 0, len(schedules))
	for i := range schedules {
		resp := s.toResponse(&schedules[i])
		s.enrichResources(resp, schedules[i].Resources)
		result = append(result, resp)
	}
	return result, total, nil
}

func (s *ScheduleService) AddResource(scheduleID string, req *dto.AddScheduleResourceRequest) (*dto.ScheduleResourceResponse, error) {
	if _, err := s.scheduleRepo.GetByID(scheduleID); err != nil {
		return nil, fmt.Errorf("日程不存在")
	}

	// 验证引用资源存在
	if err := s.validateResource(req.ResourceType, req.ResourceID); err != nil {
		return nil, err
	}

	res := &model.ScheduleResource{
		ID:           uuid.New().String(),
		ScheduleID:   scheduleID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Note:         req.Note,
	}
	if err := s.scheduleRepo.AddResource(res); err != nil {
		return nil, err
	}

	resp := s.resourceToResponse(res)
	s.enrichSingleResource(resp)
	return resp, nil
}

func (s *ScheduleService) RemoveResource(scheduleID, resourceID string) error {
	if _, err := s.scheduleRepo.GetByID(scheduleID); err != nil {
		return fmt.Errorf("日程不存在")
	}
	if _, err := s.scheduleRepo.GetResourceByID(resourceID); err != nil {
		return fmt.Errorf("关联资源不存在")
	}
	return s.scheduleRepo.RemoveResource(scheduleID, resourceID)
}

// ---- 内部辅助方法 ----

func (s *ScheduleService) toResponse(m *model.Schedule) *dto.ScheduleResponse {
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	return &dto.ScheduleResponse{
		ID:             m.ID,
		Title:          m.Title,
		Description:    m.Description,
		StartTime:      m.StartTime,
		EndTime:        m.EndTime,
		AllDay:         m.AllDay,
		Color:          m.Color,
		Location:       m.Location,
		Status:         m.Status,
		RecurrenceType: m.RecurrenceType,
		RecurrenceEnd:  m.RecurrenceEnd,
		Tags:           tags,
		Resources:      []dto.ScheduleResourceResponse{},
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func (s *ScheduleService) resourceToResponse(r *model.ScheduleResource) *dto.ScheduleResourceResponse {
	return &dto.ScheduleResourceResponse{
		ID:           r.ID,
		ScheduleID:   r.ScheduleID,
		ResourceType: r.ResourceType,
		ResourceID:   r.ResourceID,
		Note:         r.Note,
		CreatedAt:    r.CreatedAt,
	}
}

func (s *ScheduleService) enrichResources(resp *dto.ScheduleResponse, resources []model.ScheduleResource) {
	for _, r := range resources {
		rr := s.resourceToResponse(&r)
		s.enrichSingleResource(rr)
		resp.Resources = append(resp.Resources, *rr)
	}
}

// enrichSingleResource 填充资源标题、颜色、状态等冗余字段
func (s *ScheduleService) enrichSingleResource(r *dto.ScheduleResourceResponse) {
	switch r.ResourceType {
	case model.ResourceTypeProject:
		if p, err := s.projectRepo.FindByID(r.ResourceID); err == nil {
			r.ResourceTitle = p.Title
			r.ResourceColor = p.Color
			r.ResourceStatus = p.Status
		}
	case model.ResourceTypeTodo:
		if t, err := s.todoRepo.FindByID(r.ResourceID); err == nil {
			r.ResourceTitle = t.Title
			r.ResourceStatus = t.Status
		}
	case model.ResourceTypeUnit:
		if u, err := s.unitRepo.FindByID(r.ResourceID); err == nil {
			r.ResourceTitle = u.Title
			r.ResourceColor = u.Color
			r.ResourceStatus = u.Status
		}
	}
}

func (s *ScheduleService) validateResource(resourceType, resourceID string) error {
	switch resourceType {
	case model.ResourceTypeProject:
		if _, err := s.projectRepo.FindByID(resourceID); err != nil {
			return fmt.Errorf("项目不存在: %s", resourceID)
		}
	case model.ResourceTypeTodo:
		if _, err := s.todoRepo.FindByID(resourceID); err != nil {
			return fmt.Errorf("待办不存在: %s", resourceID)
		}
	case model.ResourceTypeUnit:
		if _, err := s.unitRepo.FindByID(resourceID); err != nil {
			return fmt.Errorf("计时单元不存在: %s", resourceID)
		}
	default:
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}
	return nil
}

// parseTime 支持多种常见时间格式（RFC3339、带秒、不带秒、空格分隔）
func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,          // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05", // 带秒无时区
		"2006-01-02T15:04",    // datetime-local 无秒（浏览器默认格式）
		"2006-01-02 15:04:05", // 空格分隔带秒
		"2006-01-02 15:04",    // 空格分隔无秒
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间: %s", s)
}
