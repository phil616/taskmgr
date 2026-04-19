package service

import (
	"errors"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"
	"ops-timer-backend/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrTodoNotFound  = errors.New("待办事项不存在")
	ErrGroupNotFound = errors.New("分组不存在")
)

type TodoService struct {
	todoRepo  *repository.TodoRepository
	groupRepo *repository.TodoGroupRepository
}

func NewTodoService(todoRepo *repository.TodoRepository, groupRepo *repository.TodoGroupRepository) *TodoService {
	return &TodoService{todoRepo: todoRepo, groupRepo: groupRepo}
}

func (s *TodoService) Create(req *dto.CreateTodoRequest) (*dto.TodoResponse, error) {
	todo := &model.Todo{
		ID:          uuid.New().String(),
		GroupID:     req.GroupID,
		Title:       req.Title,
		Description: req.Description,
		Status:      model.TodoStatusPending,
		Priority:    model.PriorityNormal,
	}

	if req.Status != "" {
		todo.Status = req.Status
	}
	if req.Priority != "" {
		todo.Priority = req.Priority
	}
	if req.DueDate != nil {
		t, err := timeutil.ParseDate(*req.DueDate)
		if err == nil {
			todo.DueDate = &t
		}
	}
	if req.SortOrder != nil {
		todo.SortOrder = *req.SortOrder
	}

	if err := s.todoRepo.Create(todo); err != nil {
		return nil, err
	}

	return s.toResponse(todo), nil
}

func (s *TodoService) GetByID(id string) (*dto.TodoResponse, error) {
	todo, err := s.todoRepo.FindByID(id)
	if err != nil {
		return nil, ErrTodoNotFound
	}
	return s.toResponse(todo), nil
}

func (s *TodoService) List(params *dto.TodoQueryParams) ([]dto.TodoResponse, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	filter := repository.TodoFilter{
		Status:   params.Status,
		Priority: params.Priority,
		GroupID:  params.GroupID,
		DueDate:  params.DueDate,
		Page:     params.Page,
		PageSize: params.PageSize,
	}

	todos, total, err := s.todoRepo.List(filter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.TodoResponse, len(todos))
	for i, t := range todos {
		responses[i] = *s.toResponse(&t)
	}
	return responses, total, nil
}

func (s *TodoService) Update(id string, req *dto.UpdateTodoRequest) (*dto.TodoResponse, error) {
	todo, err := s.todoRepo.FindByID(id)
	if err != nil {
		return nil, ErrTodoNotFound
	}

	if req.Title != "" {
		todo.Title = req.Title
	}
	if req.Description != nil {
		todo.Description = *req.Description
	}
	if req.Status != "" {
		todo.Status = req.Status
		if req.Status == model.TodoStatusDone {
			now := timeutil.Now()
			todo.CompletedAt = &now
		}
	}
	if req.Priority != "" {
		todo.Priority = req.Priority
	}
	if req.GroupID != nil {
		todo.GroupID = req.GroupID
	}
	if req.DueDate != nil {
		t, err := timeutil.ParseDate(*req.DueDate)
		if err == nil {
			todo.DueDate = &t
		}
	}
	if req.SortOrder != nil {
		todo.SortOrder = *req.SortOrder
	}

	if err := s.todoRepo.Update(todo); err != nil {
		return nil, err
	}
	return s.toResponse(todo), nil
}

func (s *TodoService) UpdateStatus(id string, status string) (*dto.TodoResponse, error) {
	todo, err := s.todoRepo.FindByID(id)
	if err != nil {
		return nil, ErrTodoNotFound
	}

	todo.Status = status
	if status == model.TodoStatusDone {
		now := timeutil.Now()
		todo.CompletedAt = &now
	}

	if err := s.todoRepo.Update(todo); err != nil {
		return nil, err
	}
	return s.toResponse(todo), nil
}

func (s *TodoService) Delete(id string) error {
	_, err := s.todoRepo.FindByID(id)
	if err != nil {
		return ErrTodoNotFound
	}
	return s.todoRepo.Delete(id)
}

func (s *TodoService) BatchAction(req *dto.BatchTodoRequest) error {
	switch req.Action {
	case "complete":
		return s.todoRepo.BatchUpdateStatus(req.IDs, model.TodoStatusDone)
	case "delete":
		return s.todoRepo.BatchDelete(req.IDs)
	default:
		return errors.New("不支持的批量操作")
	}
}

func (s *TodoService) CreateGroup(req *dto.CreateTodoGroupRequest) (*dto.TodoGroupResponse, error) {
	group := &model.TodoGroup{
		ID:    uuid.New().String(),
		Name:  req.Name,
		Color: req.Color,
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}

	if err := s.groupRepo.Create(group); err != nil {
		return nil, err
	}

	return s.toGroupResponse(group), nil
}

func (s *TodoService) ListGroups() ([]dto.TodoGroupResponse, error) {
	groups, err := s.groupRepo.List()
	if err != nil {
		return nil, err
	}

	responses := make([]dto.TodoGroupResponse, len(groups))
	for i, g := range groups {
		responses[i] = *s.toGroupResponse(&g)
	}
	return responses, nil
}

func (s *TodoService) UpdateGroup(id string, req *dto.UpdateTodoGroupRequest) (*dto.TodoGroupResponse, error) {
	group, err := s.groupRepo.FindByID(id)
	if err != nil {
		return nil, ErrGroupNotFound
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Color != nil {
		group.Color = *req.Color
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}

	if err := s.groupRepo.Update(group); err != nil {
		return nil, err
	}
	return s.toGroupResponse(group), nil
}

func (s *TodoService) DeleteGroup(id string) error {
	_, err := s.groupRepo.FindByID(id)
	if err != nil {
		return ErrGroupNotFound
	}
	s.todoRepo.ClearGroupTodos(id)
	return s.groupRepo.Delete(id)
}

func (s *TodoService) toResponse(todo *model.Todo) *dto.TodoResponse {
	return &dto.TodoResponse{
		ID:          todo.ID,
		GroupID:     todo.GroupID,
		Title:       todo.Title,
		Description: todo.Description,
		Status:      todo.Status,
		Priority:    todo.Priority,
		DueDate:     timeutil.NormalizePtr(todo.DueDate),
		SortOrder:   todo.SortOrder,
		CompletedAt: timeutil.NormalizePtr(todo.CompletedAt),
		CreatedAt:   timeutil.Normalize(todo.CreatedAt),
		UpdatedAt:   timeutil.Normalize(todo.UpdatedAt),
	}
}

func (s *TodoService) toGroupResponse(group *model.TodoGroup) *dto.TodoGroupResponse {
	return &dto.TodoGroupResponse{
		ID:        group.ID,
		Name:      group.Name,
		Color:     group.Color,
		SortOrder: group.SortOrder,
		TodoCount: group.TodoCount,
		CreatedAt: timeutil.Normalize(group.CreatedAt),
		UpdatedAt: timeutil.Normalize(group.UpdatedAt),
	}
}
