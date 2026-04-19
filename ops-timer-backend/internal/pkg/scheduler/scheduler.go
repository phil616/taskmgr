package scheduler

import (
	"fmt"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/email"
	"ops-timer-backend/internal/pkg/timeutil"
	"ops-timer-backend/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type Scheduler struct {
	cron         *cron.Cron
	unitRepo     *repository.UnitRepository
	notifRepo    *repository.NotificationRepository
	userRepo     *repository.UserRepository
	emailService *email.Service
	logger       *zap.Logger
}

func NewScheduler(
	unitRepo *repository.UnitRepository,
	notifRepo *repository.NotificationRepository,
	userRepo *repository.UserRepository,
	emailSvc *email.Service,
	logger *zap.Logger,
) *Scheduler {
	return &Scheduler{
		cron:         cron.New(cron.WithLocation(timeutil.Location())),
		unitRepo:     unitRepo,
		notifRepo:    notifRepo,
		userRepo:     userRepo,
		emailService: emailSvc,
		logger:       logger,
	}
}

func (s *Scheduler) Start(interval string) error {
	cronExpr := "@every " + interval
	_, err := s.cron.AddFunc(cronExpr, s.scanNotifications)
	if err != nil {
		return err
	}
	s.cron.Start()
	s.logger.Info("notification scheduler started", zap.String("interval", interval))
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) scanNotifications() {
	units, err := s.unitRepo.FindActiveUnits()
	if err != nil {
		s.logger.Error("failed to fetch active units", zap.Error(err))
		return
	}
	for _, unit := range units {
		s.checkUnit(&unit)
	}
}

func (s *Scheduler) checkUnit(unit *model.Unit) {
	switch unit.Type {
	case model.UnitTypeTimeCountdown:
		s.checkTimeCountdown(unit)
	case model.UnitTypeTimeCountup:
		s.checkTimeCountup(unit)
	case model.UnitTypeCountCountdown, model.UnitTypeCountCountup:
		s.checkCountReminders(unit)
	}
}

func (s *Scheduler) checkTimeCountdown(unit *model.Unit) {
	if unit.TargetTime == nil {
		return
	}

	remaining := time.Until(*unit.TargetTime)
	remainingDays := int(remaining.Hours() / 24)

	if remaining <= 0 {
		s.notify(unit, model.NotificationLevelCritical,
			fmt.Sprintf("[%s] 已超期", unit.Title),
			[]email.Detail{
				{Label: "截止时间", Value: unit.TargetTime.Format("2006-01-02 15:04")},
				{Label: "状态", Value: "已超过截止时间"},
			})
		return
	}

	for _, days := range unit.RemindBeforeDays {
		if remainingDays <= days {
			level := model.NotificationLevelWarning
			if remainingDays <= 1 {
				level = model.NotificationLevelCritical
			}
			s.notify(unit, level,
				fmt.Sprintf("[%s] 距到期还剩 %d 天", unit.Title, remainingDays),
				[]email.Detail{
					{Label: "截止时间", Value: unit.TargetTime.Format("2006-01-02 15:04")},
					{Label: "剩余天数", Value: fmt.Sprintf("%d 天", remainingDays)},
					{Label: "提醒阈值", Value: fmt.Sprintf("≤ %d 天时通知", days)},
				})
			return
		}
	}
}

func (s *Scheduler) checkTimeCountup(unit *model.Unit) {
	if unit.StartTime == nil {
		return
	}

	elapsed := time.Since(*unit.StartTime)
	elapsedDays := int(elapsed.Hours() / 24)

	for i := len(unit.RemindAfterDays) - 1; i >= 0; i-- {
		days := unit.RemindAfterDays[i]
		if elapsedDays >= days {
			s.notify(unit, model.NotificationLevelWarning,
				fmt.Sprintf("[%s] 已持续 %d 天", unit.Title, elapsedDays),
				[]email.Detail{
					{Label: "开始时间", Value: unit.StartTime.Format("2006-01-02 15:04")},
					{Label: "已持续", Value: fmt.Sprintf("%d 天", elapsedDays)},
					{Label: "提醒阈值", Value: fmt.Sprintf("≥ %d 天时通知", days)},
				})
			return
		}
	}
}

func (s *Scheduler) checkCountReminders(unit *model.Unit) {
	if unit.CurrentValue == nil {
		return
	}

	for _, val := range unit.RemindOnValues {
		if *unit.CurrentValue >= val {
			s.notify(unit, model.NotificationLevelInfo,
				fmt.Sprintf("[%s] 当前值已达 %.0f %s", unit.Title, *unit.CurrentValue, unit.UnitLabel),
				[]email.Detail{
					{Label: "当前值", Value: fmt.Sprintf("%.4g %s", *unit.CurrentValue, unit.UnitLabel)},
					{Label: "触发阈值", Value: fmt.Sprintf("%.4g %s", val, unit.UnitLabel)},
				})
			return
		}
	}
}

// notify 同时写入站内通知并（如已配置）发送邮件
func (s *Scheduler) notify(unit *model.Unit, level, message string, details []email.Detail) {
	exists, err := s.notifRepo.ExistsTodayForUnit(unit.ID, level)
	if err != nil || exists {
		return
	}

	n := &model.Notification{
		ID:          uuid.New().String(),
		UnitID:      unit.ID,
		Level:       level,
		Message:     message,
		TriggeredAt: timeutil.Now(),
	}

	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("failed to create notification", zap.Error(err))
		return
	}

	// 发送邮件通知
	if s.emailService.Enabled() {
		go s.sendEmailNotification(unit, message, level, details)
	}
}

func (s *Scheduler) sendEmailNotification(unit *model.Unit, message, level string, details []email.Detail) {
	users, err := s.userRepo.FindAllWithEmail()
	if err != nil {
		s.logger.Error("failed to fetch users for email notification", zap.Error(err))
		return
	}

	unitTypeLabel := unitTypeLabel(unit.Type)

	for _, user := range users {
		if err := s.emailService.SendNotification(
			user.Email,
			unit.Title,
			unitTypeLabel,
			message,
			level,
			details,
		); err != nil {
			s.logger.Error("failed to send email notification",
				zap.String("to", user.Email),
				zap.String("unit_id", unit.ID),
				zap.Error(err))
		} else {
			s.logger.Info("email notification sent",
				zap.String("to", user.Email),
				zap.String("unit", unit.Title),
				zap.String("level", level))
		}
	}
}

func unitTypeLabel(t string) string {
	switch t {
	case model.UnitTypeTimeCountdown:
		return "时间倒计时"
	case model.UnitTypeTimeCountup:
		return "时间正计时"
	case model.UnitTypeCountCountdown:
		return "数值倒计时"
	case model.UnitTypeCountCountup:
		return "数值正计时"
	}
	return t
}
