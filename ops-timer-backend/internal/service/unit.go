package service

import (
	"errors"
	"math"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"
	"ops-timer-backend/internal/repository"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnitNotFound     = errors.New("计时单元不存在")
	ErrNotCountType     = errors.New("该操作仅适用于数值型计时单元")
	ErrExceedNotAllowed = errors.New("不允许超出目标值")
	ErrInvalidUnitType  = errors.New("无效的计时单元类型")
)

type UnitService struct {
	unitRepo *repository.UnitRepository
	logRepo  *repository.UnitLogRepository
}

func NewUnitService(unitRepo *repository.UnitRepository, logRepo *repository.UnitLogRepository) *UnitService {
	return &UnitService{unitRepo: unitRepo, logRepo: logRepo}
}

func (s *UnitService) Create(req *dto.CreateUnitRequest) (*dto.UnitResponse, error) {
	unit := &model.Unit{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Status:      "active",
		Priority:    "normal",
		Tags:        model.JSONStringArray(req.Tags),
		Color:       req.Color,
		DisplayUnit: "days",
		Step:        1,
	}

	if req.Status != "" {
		unit.Status = req.Status
	}
	if req.Priority != "" {
		unit.Priority = req.Priority
	}
	if req.DisplayUnit != "" {
		unit.DisplayUnit = req.DisplayUnit
	}

	switch req.Type {
	case model.UnitTypeTimeCountdown:
		unit.TargetTime = req.TargetTime
		unit.RemindBeforeDays = model.JSONIntArray(req.RemindBeforeDays)
	case model.UnitTypeTimeCountup:
		unit.StartTime = req.StartTime
		unit.RemindAfterDays = model.JSONIntArray(req.RemindAfterDays)
	case model.UnitTypeCountCountdown:
		unit.CurrentValue = req.CurrentValue
		unit.TargetValue = req.TargetValue
		unit.UnitLabel = req.UnitLabel
		unit.RemindOnValues = model.JSONFloatArray(req.RemindOnValues)
		if req.AllowExceed != nil {
			unit.AllowExceed = *req.AllowExceed
		}
	case model.UnitTypeCountCountup:
		unit.CurrentValue = req.CurrentValue
		unit.UnitLabel = req.UnitLabel
		unit.RemindOnValues = model.JSONFloatArray(req.RemindOnValues)
	default:
		return nil, ErrInvalidUnitType
	}

	if req.Step != nil {
		unit.Step = *req.Step
	}

	if err := s.unitRepo.Create(unit); err != nil {
		return nil, err
	}

	return s.toResponse(unit), nil
}

func (s *UnitService) GetByID(id string) (*dto.UnitResponse, error) {
	unit, err := s.unitRepo.FindByID(id)
	if err != nil {
		return nil, ErrUnitNotFound
	}
	return s.toResponse(unit), nil
}

func (s *UnitService) List(params *dto.UnitQueryParams) ([]dto.UnitResponse, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	filter := repository.UnitFilter{
		Status:    params.Status,
		ProjectID: params.ProjectID,
		Priority:  params.Priority,
		Q:         params.Q,
		SortBy:    params.SortBy,
		SortOrder: params.SortOrder,
		Page:      params.Page,
		PageSize:  params.PageSize,
	}

	if params.Type != "" {
		filter.Types = strings.Split(params.Type, ",")
	}
	if params.Tags != "" {
		filter.Tags = strings.Split(params.Tags, ",")
	}

	units, total, err := s.unitRepo.List(filter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.UnitResponse, len(units))
	for i, u := range units {
		responses[i] = *s.toResponse(&u)
	}
	return responses, total, nil
}

func (s *UnitService) Update(id string, req *dto.UpdateUnitRequest) (*dto.UnitResponse, error) {
	unit, err := s.unitRepo.FindByID(id)
	if err != nil {
		return nil, ErrUnitNotFound
	}

	if req.Title != "" {
		unit.Title = req.Title
	}
	if req.Description != nil {
		unit.Description = *req.Description
	}
	if req.Status != "" {
		unit.Status = req.Status
	}
	if req.Priority != "" {
		unit.Priority = req.Priority
	}
	if req.Tags != nil {
		unit.Tags = model.JSONStringArray(req.Tags)
	}
	if req.Color != nil {
		unit.Color = *req.Color
	}
	if req.ClearProject {
		unit.ProjectID = nil
	} else if req.ProjectID != nil {
		unit.ProjectID = req.ProjectID
	}

	if req.TargetTime != nil {
		unit.TargetTime = req.TargetTime
	}
	if req.StartTime != nil {
		unit.StartTime = req.StartTime
	}
	if req.DisplayUnit != "" {
		unit.DisplayUnit = req.DisplayUnit
	}
	if req.RemindBeforeDays != nil {
		unit.RemindBeforeDays = model.JSONIntArray(req.RemindBeforeDays)
	}
	if req.RemindAfterDays != nil {
		unit.RemindAfterDays = model.JSONIntArray(req.RemindAfterDays)
	}

	if req.CurrentValue != nil {
		unit.CurrentValue = req.CurrentValue
	}
	if req.TargetValue != nil {
		unit.TargetValue = req.TargetValue
	}
	if req.Step != nil {
		unit.Step = *req.Step
	}
	if req.UnitLabel != nil {
		unit.UnitLabel = *req.UnitLabel
	}
	if req.AllowExceed != nil {
		unit.AllowExceed = *req.AllowExceed
	}
	if req.RemindOnValues != nil {
		unit.RemindOnValues = model.JSONFloatArray(req.RemindOnValues)
	}

	if err := s.unitRepo.Update(unit); err != nil {
		return nil, err
	}
	return s.toResponse(unit), nil
}

func (s *UnitService) UpdateStatus(id string, status string) (*dto.UnitResponse, error) {
	unit, err := s.unitRepo.FindByID(id)
	if err != nil {
		return nil, ErrUnitNotFound
	}
	unit.Status = status
	if err := s.unitRepo.Update(unit); err != nil {
		return nil, err
	}
	return s.toResponse(unit), nil
}

func (s *UnitService) Delete(id string) error {
	_, err := s.unitRepo.FindByID(id)
	if err != nil {
		return ErrUnitNotFound
	}
	return s.unitRepo.Delete(id)
}

func (s *UnitService) Step(id string, req *dto.StepRequest) (*dto.UnitResponse, error) {
	unit, err := s.unitRepo.FindByID(id)
	if err != nil {
		return nil, ErrUnitNotFound
	}

	if !unit.IsCountType() {
		return nil, ErrNotCountType
	}

	currentVal := float64(0)
	if unit.CurrentValue != nil {
		currentVal = *unit.CurrentValue
	}

	delta := unit.Step
	if req.Direction == "down" {
		delta = -delta
	}

	newVal := currentVal + delta

	if unit.IsCountdown() && unit.TargetValue != nil && !unit.AllowExceed && newVal > *unit.TargetValue {
		return nil, ErrExceedNotAllowed
	}

	log := &model.UnitLog{
		ID:          uuid.New().String(),
		UnitID:      unit.ID,
		Delta:       delta,
		ValueBefore: currentVal,
		ValueAfter:  newVal,
		Note:        req.Note,
		OperatedAt:  timeutil.Now(),
	}

	if err := s.logRepo.Create(log); err != nil {
		return nil, err
	}

	unit.CurrentValue = &newVal
	if err := s.unitRepo.Update(unit); err != nil {
		return nil, err
	}

	return s.toResponse(unit), nil
}

func (s *UnitService) SetValue(id string, req *dto.SetValueRequest) (*dto.UnitResponse, error) {
	unit, err := s.unitRepo.FindByID(id)
	if err != nil {
		return nil, ErrUnitNotFound
	}

	if !unit.IsCountType() {
		return nil, ErrNotCountType
	}

	currentVal := float64(0)
	if unit.CurrentValue != nil {
		currentVal = *unit.CurrentValue
	}

	if unit.IsCountdown() && unit.TargetValue != nil && !unit.AllowExceed && req.Value > *unit.TargetValue {
		return nil, ErrExceedNotAllowed
	}

	log := &model.UnitLog{
		ID:          uuid.New().String(),
		UnitID:      unit.ID,
		Delta:       req.Value - currentVal,
		ValueBefore: currentVal,
		ValueAfter:  req.Value,
		Note:        req.Note,
		OperatedAt:  timeutil.Now(),
	}

	if err := s.logRepo.Create(log); err != nil {
		return nil, err
	}

	unit.CurrentValue = &req.Value
	if err := s.unitRepo.Update(unit); err != nil {
		return nil, err
	}

	return s.toResponse(unit), nil
}

func (s *UnitService) GetLogs(unitID string, page, pageSize int) ([]dto.UnitLogResponse, int64, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	logs, total, err := s.logRepo.ListByUnitID(unitID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.UnitLogResponse, len(logs))
	for i, l := range logs {
		responses[i] = dto.UnitLogResponse{
			ID:          l.ID,
			UnitID:      l.UnitID,
			Delta:       l.Delta,
			ValueBefore: l.ValueBefore,
			ValueAfter:  l.ValueAfter,
			Note:        l.Note,
			OperatedAt:  l.OperatedAt,
		}
	}
	return responses, total, nil
}

func (s *UnitService) GetSummary() (*dto.UnitSummary, error) {
	counts, err := s.unitRepo.CountByStatus()
	if err != nil {
		return nil, err
	}

	expiringCount, _ := s.unitRepo.CountExpiring(7)
	expiredCount, _ := s.unitRepo.CountExpired()

	return &dto.UnitSummary{
		TotalActive:    counts[model.UnitStatusActive],
		TotalPaused:    counts[model.UnitStatusPaused],
		TotalCompleted: counts[model.UnitStatusCompleted],
		TotalArchived:  counts[model.UnitStatusArchived],
		ExpiringCount:  expiringCount,
		ExpiredCount:   expiredCount,
	}, nil
}

func (s *UnitService) ListByProject(projectID string, page, pageSize int) ([]dto.UnitResponse, int64, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	units, total, err := s.unitRepo.ListByProjectID(projectID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.UnitResponse, len(units))
	for i, u := range units {
		responses[i] = *s.toResponse(&u)
	}
	return responses, total, nil
}

func (s *UnitService) toResponse(unit *model.Unit) *dto.UnitResponse {
	resp := &dto.UnitResponse{
		ID:          unit.ID,
		ProjectID:   unit.ProjectID,
		Title:       unit.Title,
		Description: unit.Description,
		Type:        unit.Type,
		Status:      unit.Status,
		Priority:    unit.Priority,
		Tags:        []string(unit.Tags),
		Color:       unit.Color,
		CreatedAt:   timeutil.Normalize(unit.CreatedAt),
		UpdatedAt:   timeutil.Normalize(unit.UpdatedAt),
	}

	if resp.Tags == nil {
		resp.Tags = []string{}
	}

	switch unit.Type {
	case model.UnitTypeTimeCountdown:
		resp.TargetTime = timeutil.NormalizePtr(unit.TargetTime)
		resp.DisplayUnit = unit.DisplayUnit
		resp.RemindBeforeDays = []int(unit.RemindBeforeDays)
		if unit.TargetTime != nil {
			remaining := time.Until(*unit.TargetTime).Seconds()
			resp.RemainingSeconds = &remaining
		}
	case model.UnitTypeTimeCountup:
		resp.StartTime = timeutil.NormalizePtr(unit.StartTime)
		resp.DisplayUnit = unit.DisplayUnit
		resp.RemindAfterDays = []int(unit.RemindAfterDays)
		if unit.StartTime != nil {
			elapsed := time.Since(*unit.StartTime).Seconds()
			resp.ElapsedSeconds = &elapsed
		}
	case model.UnitTypeCountCountdown:
		resp.CurrentValue = unit.CurrentValue
		resp.TargetValue = unit.TargetValue
		resp.Step = unit.Step
		resp.UnitLabel = unit.UnitLabel
		resp.AllowExceed = unit.AllowExceed
		resp.RemindOnValues = []float64(unit.RemindOnValues)
		if unit.TargetValue != nil && *unit.TargetValue > 0 {
			cv := float64(0)
			if unit.CurrentValue != nil {
				cv = *unit.CurrentValue
			}
			progress := math.Min(cv / *unit.TargetValue * 100, 100)
			resp.Progress = &progress
		}
	case model.UnitTypeCountCountup:
		resp.CurrentValue = unit.CurrentValue
		resp.Step = unit.Step
		resp.UnitLabel = unit.UnitLabel
		resp.RemindOnValues = []float64(unit.RemindOnValues)
	}

	return resp
}
