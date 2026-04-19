package service

import (
	"errors"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"
	"ops-timer-backend/internal/repository"

	"github.com/google/uuid"
)

var ErrProjectNotFound = errors.New("项目不存在")

type ProjectService struct {
	projectRepo *repository.ProjectRepository
	unitRepo    *repository.UnitRepository
	txRepo      *repository.TransactionRepository
}

func NewProjectService(projectRepo *repository.ProjectRepository, unitRepo *repository.UnitRepository, txRepo *repository.TransactionRepository) *ProjectService {
	return &ProjectService{projectRepo: projectRepo, unitRepo: unitRepo, txRepo: txRepo}
}

func (s *ProjectService) Create(req *dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	project := &model.Project{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		Status:      "active",
		Color:       req.Color,
		Icon:        req.Icon,
		MaxBudget:   req.MaxBudget,
	}

	if req.Status != "" {
		project.Status = req.Status
	}
	if req.SortOrder != nil {
		project.SortOrder = *req.SortOrder
	}

	if err := s.projectRepo.Create(project); err != nil {
		return nil, err
	}

	return s.toResponse(project, true), nil
}

func (s *ProjectService) GetByID(id string) (*dto.ProjectResponse, error) {
	project, err := s.projectRepo.FindByID(id)
	if err != nil {
		return nil, ErrProjectNotFound
	}
	return s.toResponse(project, true), nil
}

func (s *ProjectService) List(params *dto.ProjectQueryParams) ([]dto.ProjectResponse, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	projects, total, err := s.projectRepo.List(params.Status, params.Page, params.PageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.ProjectResponse, len(projects))
	for i, p := range projects {
		responses[i] = *s.toResponse(&p, true)
	}
	return responses, total, nil
}

func (s *ProjectService) Update(id string, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	project, err := s.projectRepo.FindByID(id)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if req.Title != "" {
		project.Title = req.Title
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Status != "" {
		project.Status = req.Status
	}
	if req.Color != nil {
		project.Color = *req.Color
	}
	if req.Icon != nil {
		project.Icon = *req.Icon
	}
	if req.SortOrder != nil {
		project.SortOrder = *req.SortOrder
	}
	if req.MaxBudget != nil {
		project.MaxBudget = *req.MaxBudget
	}

	if err := s.projectRepo.Update(project); err != nil {
		return nil, err
	}
	return s.toResponse(project, true), nil
}

func (s *ProjectService) Delete(id string) error {
	_, err := s.projectRepo.FindByID(id)
	if err != nil {
		return ErrProjectNotFound
	}
	s.projectRepo.ClearProjectUnits(id)
	s.txRepo.ClearProjectTransactions(id)
	return s.projectRepo.Delete(id)
}

func (s *ProjectService) GetBudgetStats(id string) (*dto.ProjectBudgetStats, error) {
	project, err := s.projectRepo.FindByID(id)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	stats := s.txRepo.StatByProject(id)
	budgetStats := &dto.ProjectBudgetStats{
		MaxBudget: project.MaxBudget,
	}
	for _, st := range stats {
		switch st.Type {
		case model.TransactionTypeIncome:
			budgetStats.TotalIncome = st.Total
			budgetStats.TxCount += st.Count
		case model.TransactionTypeExpense:
			budgetStats.TotalExpense = st.Total
			budgetStats.TxCount += st.Count
		}
	}
	budgetStats.NetAmount = budgetStats.TotalIncome - budgetStats.TotalExpense
	budgetStats.Remaining = project.MaxBudget - budgetStats.TotalExpense
	if project.MaxBudget > 0 {
		budgetStats.UsageRate = budgetStats.TotalExpense / project.MaxBudget
	}
	return budgetStats, nil
}

func (s *ProjectService) toResponse(project *model.Project, withStats bool) *dto.ProjectResponse {
	resp := &dto.ProjectResponse{
		ID:          project.ID,
		Title:       project.Title,
		Description: project.Description,
		Status:      project.Status,
		Color:       project.Color,
		Icon:        project.Icon,
		SortOrder:   project.SortOrder,
		MaxBudget:   project.MaxBudget,
		CreatedAt:   timeutil.Normalize(project.CreatedAt),
		UpdatedAt:   timeutil.Normalize(project.UpdatedAt),
	}

	if withStats {
		activeCount, _ := s.unitRepo.CountByProjectAndStatus(project.ID, model.UnitStatusActive)
		completedCount, _ := s.unitRepo.CountByProjectAndStatus(project.ID, model.UnitStatusCompleted)
		totalCount, _ := s.unitRepo.CountByProjectAndStatus(project.ID, "")

		resp.UnitStats = &dto.ProjectUnitStats{
			ActiveCount:    activeCount,
			CompletedCount: completedCount,
			TotalCount:     totalCount,
		}

		txStats := s.txRepo.StatByProject(project.ID)
		budgetStats := &dto.ProjectBudgetStats{
			MaxBudget: project.MaxBudget,
		}
		for _, st := range txStats {
			switch st.Type {
			case model.TransactionTypeIncome:
				budgetStats.TotalIncome = st.Total
				budgetStats.TxCount += st.Count
			case model.TransactionTypeExpense:
				budgetStats.TotalExpense = st.Total
				budgetStats.TxCount += st.Count
			}
		}
		budgetStats.NetAmount = budgetStats.TotalIncome - budgetStats.TotalExpense
		budgetStats.Remaining = project.MaxBudget - budgetStats.TotalExpense
		if project.MaxBudget > 0 {
			budgetStats.UsageRate = budgetStats.TotalExpense / project.MaxBudget
		}
		resp.BudgetStats = budgetStats
	}

	return resp
}
