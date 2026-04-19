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
	ErrSecretNotFound      = errors.New("密钥不存在")
	ErrSecretNameDuplicate = errors.New("密钥名称已存在")
)

type SecretService struct {
	secretRepo  *repository.SecretRepository
	auditRepo   *repository.SecretAuditLogRepository
	projectRepo *repository.ProjectRepository
}

func NewSecretService(
	secretRepo *repository.SecretRepository,
	auditRepo *repository.SecretAuditLogRepository,
	projectRepo *repository.ProjectRepository,
) *SecretService {
	return &SecretService{
		secretRepo:  secretRepo,
		auditRepo:   auditRepo,
		projectRepo: projectRepo,
	}
}

type AuditContext struct {
	UserID    string
	Username  string
	IPAddress string
	UserAgent string
}

func (s *SecretService) Create(req *dto.CreateSecretRequest, ctx *AuditContext) (*dto.SecretResponse, error) {
	if existing, _ := s.secretRepo.FindByName(req.Name); existing != nil {
		return nil, ErrSecretNameDuplicate
	}

	secret := &model.Secret{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Value:       req.Value,
		Description: req.Description,
		Tags:        req.Tags,
		ProjectID:   req.ProjectID,
	}

	if err := s.secretRepo.Create(secret); err != nil {
		return nil, err
	}

	s.writeAudit(secret.ID, model.SecretActionCreated, ctx, "创建密钥: "+secret.Name)

	secret, _ = s.secretRepo.FindByID(secret.ID)
	return s.toResponse(secret), nil
}

func (s *SecretService) GetByID(id string, ctx *AuditContext) (*dto.SecretResponse, error) {
	secret, err := s.secretRepo.FindByID(id)
	if err != nil {
		return nil, ErrSecretNotFound
	}

	s.writeAudit(secret.ID, model.SecretActionRead, ctx, "查看密钥详情")

	return s.toResponse(secret), nil
}

func (s *SecretService) GetValue(id string, ctx *AuditContext) (*dto.SecretResponse, error) {
	secret, err := s.secretRepo.FindByID(id)
	if err != nil {
		return nil, ErrSecretNotFound
	}

	s.writeAudit(secret.ID, model.SecretActionValueRead, ctx, "读取密钥值")

	return s.toResponse(secret), nil
}

func (s *SecretService) List(params *dto.SecretQueryParams, ctx *AuditContext) ([]dto.SecretBriefResponse, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	secrets, total, err := s.secretRepo.List(params.Name, params.Tag, params.ProjectID, params.Page, params.PageSize)
	if err != nil {
		return nil, 0, err
	}

	s.writeAudit("", model.SecretActionListed, ctx, "列出密钥列表")

	responses := make([]dto.SecretBriefResponse, len(secrets))
	for i, sec := range secrets {
		responses[i] = *s.toBriefResponse(&sec)
	}
	return responses, total, nil
}

func (s *SecretService) Update(id string, req *dto.UpdateSecretRequest, ctx *AuditContext) (*dto.SecretResponse, error) {
	secret, err := s.secretRepo.FindByID(id)
	if err != nil {
		return nil, ErrSecretNotFound
	}

	if req.Name != nil && *req.Name != secret.Name {
		if existing, _ := s.secretRepo.FindByName(*req.Name); existing != nil && existing.ID != id {
			return nil, ErrSecretNameDuplicate
		}
		secret.Name = *req.Name
	}
	if req.Value != nil {
		secret.Value = *req.Value
	}
	if req.Description != nil {
		secret.Description = *req.Description
	}
	if req.Tags != nil {
		secret.Tags = req.Tags
	}
	if req.ProjectID != nil {
		if *req.ProjectID == "" {
			secret.ProjectID = nil
		} else {
			secret.ProjectID = req.ProjectID
		}
	}

	if err := s.secretRepo.Update(secret); err != nil {
		return nil, err
	}

	s.writeAudit(secret.ID, model.SecretActionUpdated, ctx, "更新密钥: "+secret.Name)

	secret, _ = s.secretRepo.FindByID(secret.ID)
	return s.toResponse(secret), nil
}

func (s *SecretService) Delete(id string, ctx *AuditContext) error {
	secret, err := s.secretRepo.FindByID(id)
	if err != nil {
		return ErrSecretNotFound
	}

	s.writeAudit(secret.ID, model.SecretActionDeleted, ctx, "删除密钥: "+secret.Name)

	return s.secretRepo.Delete(id)
}

func (s *SecretService) ListAuditLogs(params *dto.SecretAuditQueryParams) ([]dto.SecretAuditLogResponse, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	logs, total, err := s.auditRepo.ListBySecret(params.SecretID, params.Action, params.Page, params.PageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.SecretAuditLogResponse, len(logs))
	for i, l := range logs {
		responses[i] = dto.SecretAuditLogResponse{
			ID:        l.ID,
			SecretID:  l.SecretID,
			Action:    l.Action,
			UserID:    l.UserID,
			Username:  l.Username,
			IPAddress: l.IPAddress,
			UserAgent: l.UserAgent,
			Detail:    l.Detail,
			CreatedAt: timeutil.Normalize(l.CreatedAt),
		}
	}
	return responses, total, nil
}

func (s *SecretService) writeAudit(secretID, action string, ctx *AuditContext, detail string) {
	log := &model.SecretAuditLog{
		ID:        uuid.New().String(),
		SecretID:  secretID,
		Action:    action,
		UserID:    ctx.UserID,
		Username:  ctx.Username,
		IPAddress: ctx.IPAddress,
		UserAgent: ctx.UserAgent,
		Detail:    detail,
	}
	_ = s.auditRepo.Create(log)
}

func (s *SecretService) toResponse(secret *model.Secret) *dto.SecretResponse {
	resp := &dto.SecretResponse{
		ID:          secret.ID,
		Name:        secret.Name,
		Value:       secret.Value,
		Description: secret.Description,
		Tags:        secret.Tags,
		ProjectID:   secret.ProjectID,
		CreatedAt:   timeutil.Normalize(secret.CreatedAt),
		UpdatedAt:   timeutil.Normalize(secret.UpdatedAt),
	}
	if secret.Project != nil {
		resp.ProjectName = secret.Project.Title
	}
	if resp.Tags == nil {
		resp.Tags = []string{}
	}
	return resp
}

func (s *SecretService) toBriefResponse(secret *model.Secret) *dto.SecretBriefResponse {
	resp := &dto.SecretBriefResponse{
		ID:          secret.ID,
		Name:        secret.Name,
		Description: secret.Description,
		Tags:        secret.Tags,
		ProjectID:   secret.ProjectID,
		CreatedAt:   timeutil.Normalize(secret.CreatedAt),
		UpdatedAt:   timeutil.Normalize(secret.UpdatedAt),
	}
	if secret.Project != nil {
		resp.ProjectName = secret.Project.Title
	}
	if resp.Tags == nil {
		resp.Tags = []string{}
	}
	return resp
}
