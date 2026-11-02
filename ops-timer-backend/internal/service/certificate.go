package service

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/certparse"
)

type CertificateService struct{ db *gorm.DB }

type CertificateInputError struct{ Message string }

func (e *CertificateInputError) Error() string { return e.Message }

func NewCertificateService(db *gorm.DB) *CertificateService { return &CertificateService{db: db} }

// ValidateOwner uses a fixed model map rather than interpolating client table names.
func validateAttachmentOwner(db *gorm.DB, owner dto.AttachmentOwner) error {
	var target any
	switch owner.OwnerType {
	case "project":
		target = &model.Project{}
	case "unit":
		target = &model.Unit{}
	case "todo":
		target = &model.Todo{}
	case "note":
		target = &model.Note{}
	case "schedule":
		target = &model.Schedule{}
	default:
		return &CertificateInputError{Message: "不支持的附件关联类型"}
	}
	if _, err := uuid.Parse(owner.OwnerID); err != nil {
		return &CertificateInputError{Message: "附件关联 ID 格式错误"}
	}
	if err := db.First(target, "id = ?", owner.OwnerID).Error; err != nil {
		return err
	}
	return nil
}

func (s *CertificateService) Upload(filename string, data []byte, owner *dto.AttachmentOwner) (*dto.AttachmentResponse, error) {
	certs, err := certparse.Parse(data)
	if err != nil {
		return nil, &CertificateInputError{Message: err.Error()}
	}
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if filename == "." || filename == "" || utf8.RuneCountInString(filename) > 255 || strings.ContainsAny(filename, "\r\n\x00") {
		return nil, &CertificateInputError{Message: "文件名无效或超过 255 字符"}
	}
	a := model.Attachment{ID: uuid.NewString(), Filename: filename, Size: int64(len(data)), Content: data,
		ContentType: "application/octet-stream", Certificates: certs}
	for i := range a.Certificates {
		a.Certificates[i].ID = uuid.NewString()
		a.Certificates[i].AttachmentID = a.ID
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if owner != nil {
			if err := validateAttachmentOwner(tx, *owner); err != nil {
				return err
			}
		}
		if err := tx.Create(&a).Error; err != nil {
			return err
		}
		if owner != nil {
			return linkAttachment(tx, a.ID, *owner)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp := attachmentResponse(a)
	return &resp, nil
}

func (s *CertificateService) List(q dto.AttachmentQuery) ([]dto.AttachmentResponse, int64, error) {
	if (q.OwnerType == "") != (q.OwnerID == "") {
		return nil, 0, &CertificateInputError{Message: "关联类型和 ID 必须同时提供"}
	}
	query := s.db.Model(&model.Attachment{})
	if q.OwnerType != "" {
		if err := validateAttachmentOwner(s.db, dto.AttachmentOwner{OwnerType: q.OwnerType, OwnerID: q.OwnerID}); err != nil {
			return nil, 0, err
		}
		query = query.Where("id IN (?)", s.db.Model(&model.AttachmentLink{}).Select("attachment_id").Where("owner_type = ? AND owner_id = ?", q.OwnerType, q.OwnerID))
	}
	if q.Type != "" {
		query = query.Where("id IN (?)", s.db.Model(&model.Certificate{}).Select("attachment_id").Where("type = ?", q.Type))
	}
	if q.Q != "" {
		pattern := "%" + q.Q + "%"
		query = query.Where("filename ILIKE ? OR id IN (?)", pattern, s.db.Model(&model.Certificate{}).Select("attachment_id").Where("common_name ILIKE ? OR subject ILIKE ? OR fingerprint ILIKE ?", pattern, pattern, pattern))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var attachments []model.Attachment
	err := query.Omit("Content").Preload("Certificates", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).Order("created_at DESC, id").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&attachments).Error
	result := make([]dto.AttachmentResponse, 0, len(attachments))
	for _, a := range attachments {
		result = append(result, attachmentResponse(a))
	}
	return result, total, err
}

func attachmentResponse(a model.Attachment) dto.AttachmentResponse {
	r := dto.AttachmentResponse{ID: a.ID, Filename: a.Filename, ContentType: a.ContentType, Size: a.Size, CreatedAt: a.CreatedAt, Certificates: make([]dto.CertificateResponse, 0, len(a.Certificates))}
	now := time.Now()
	for _, cert := range a.Certificates {
		c := dto.CertificateResponse{Certificate: cert, Status: "valid"}
		if cert.ExpiresAt != nil {
			remaining := cert.ExpiresAt.Sub(now).Seconds()
			c.RemainingSeconds = &remaining
			if remaining <= 0 {
				c.Status = "expired"
			} else if remaining <= 30*86400 {
				c.Status = "expiring"
			}
		}
		if cert.NotBefore.After(now) {
			c.Status = "not_yet_valid"
		}
		if cert.Revoked {
			c.Status = "revoked"
		}
		r.Certificates = append(r.Certificates, c)
	}
	return r
}

func (s *CertificateService) Download(id string) (*model.Attachment, error) {
	var a model.Attachment
	if err := s.db.First(&a, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *CertificateService) Delete(id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var a model.Attachment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, "id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Where("attachment_id = ?", id).Delete(&model.AttachmentLink{}).Error; err != nil {
			return err
		}
		if err := tx.Where("attachment_id = ?", id).Delete(&model.Certificate{}).Error; err != nil {
			return err
		}
		return tx.Delete(&a).Error
	})
}

func linkAttachment(tx *gorm.DB, id string, owner dto.AttachmentOwner) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.AttachmentLink{ID: uuid.NewString(), AttachmentID: id, OwnerType: owner.OwnerType, OwnerID: owner.OwnerID}).Error
}

func (s *CertificateService) Link(id string, owner dto.AttachmentOwner, remove bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var a model.Attachment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Omit("Content").First(&a, "id = ?", id).Error; err != nil {
			return err
		}
		if err := validateAttachmentOwner(tx, owner); err != nil {
			return err
		}
		if remove {
			return tx.Where("attachment_id = ? AND owner_type = ? AND owner_id = ?", id, owner.OwnerType, owner.OwnerID).Delete(&model.AttachmentLink{}).Error
		}
		return linkAttachment(tx, id, owner)
	})
}

func (s *CertificateService) CreateCountdown(id string) (*model.Unit, error) {
	var unit model.Unit
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var c model.Certificate
		if err := tx.First(&c, "id = ?", id).Error; err != nil {
			return err
		}
		// Match attachment deletion's lock order so creating a countdown and
		// deleting its source file cannot leave a dangling attachment link.
		var attachment model.Attachment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Omit("Content").First(&attachment, "id = ?", c.AttachmentID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, "id = ?", id).Error; err != nil {
			return err
		}
		if c.Revoked {
			return &CertificateInputError{Message: "证书已撤销，无法创建到期倒计时"}
		}
		if c.ExpiresAt == nil {
			return &CertificateInputError{Message: "证书没有到期时间，无法创建到期倒计时"}
		}
		if c.CountdownUnitID != nil {
			err := tx.First(&unit, "id = ?", *c.CountdownUnitID).Error
			if err == nil {
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		name := c.CommonName
		if name == "" {
			name = c.Subject
		}
		if name == "" {
			name = c.Fingerprint
		}
		title := []rune(name + " 证书到期")
		if len(title) > 128 {
			title = title[:128]
		}
		unit = model.Unit{ID: uuid.NewString(), Title: string(title), Type: model.UnitTypeTimeCountdown,
			Status: model.UnitStatusActive, Priority: model.PriorityNormal, DisplayUnit: "days", TargetTime: c.ExpiresAt,
			RemindBeforeDays: model.JSONIntArray{30, 7, 1}, Tags: model.JSONStringArray{"证书"}, Step: 1}
		// A certificate uploaded from a project inherits that project for its countdown.
		var link model.AttachmentLink
		if err := tx.Where("attachment_id = ? AND owner_type = ?", c.AttachmentID, "project").Order("created_at, id").First(&link).Error; err == nil {
			var project model.Project
			if err := tx.First(&project, "id = ?", link.OwnerID).Error; err == nil {
				unit.ProjectID = &project.ID
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(&unit).Error; err != nil {
			return err
		}
		if err := linkAttachment(tx, c.AttachmentID, dto.AttachmentOwner{OwnerType: "unit", OwnerID: unit.ID}); err != nil {
			return err
		}
		return tx.Model(&c).Update("countdown_unit_id", unit.ID).Error
	})
	return &unit, err
}
