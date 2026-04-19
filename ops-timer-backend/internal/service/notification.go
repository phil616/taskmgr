package service

import (
	"errors"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"
	"ops-timer-backend/internal/repository"

	"github.com/google/uuid"
)

var ErrNotificationNotFound = errors.New("通知不存在")

type NotificationService struct {
	notifRepo *repository.NotificationRepository
}

func NewNotificationService(notifRepo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{notifRepo: notifRepo}
}

func (s *NotificationService) List(params *dto.NotificationQueryParams) ([]dto.NotificationResponse, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	notifications, total, err := s.notifRepo.List(params.IsRead, params.Level, params.Page, params.PageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.NotificationResponse, len(notifications))
	for i, n := range notifications {
		responses[i] = s.toResponse(&n)
	}
	return responses, total, nil
}

func (s *NotificationService) MarkAsRead(id string) error {
	_, err := s.notifRepo.FindByID(id)
	if err != nil {
		return ErrNotificationNotFound
	}
	return s.notifRepo.MarkAsRead(id)
}

func (s *NotificationService) MarkAllAsRead() error {
	return s.notifRepo.MarkAllAsRead()
}

func (s *NotificationService) UnreadCount() (int64, error) {
	return s.notifRepo.UnreadCount()
}

func (s *NotificationService) Delete(id string) error {
	_, err := s.notifRepo.FindByID(id)
	if err != nil {
		return ErrNotificationNotFound
	}
	return s.notifRepo.Delete(id)
}

func (s *NotificationService) CreateNotification(unitID, level, message string) error {
	n := &model.Notification{
		ID:      uuid.New().String(),
		UnitID:  unitID,
		Level:   level,
		Message: message,
	}
	return s.notifRepo.Create(n)
}

func (s *NotificationService) toResponse(n *model.Notification) dto.NotificationResponse {
	resp := dto.NotificationResponse{
		ID:          n.ID,
		UnitID:      n.UnitID,
		Level:       n.Level,
		Message:     n.Message,
		IsRead:      n.IsRead,
		TriggeredAt: timeutil.Normalize(n.TriggeredAt),
		ReadAt:      timeutil.NormalizePtr(n.ReadAt),
	}
	if n.Unit != nil {
		resp.UnitTitle = n.Unit.Title
	}
	return resp
}
