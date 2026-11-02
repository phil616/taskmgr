package service

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf16"

	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/certparse"
	"ops-timer-backend/internal/pkg/timeutil"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const backupBatchSize = 200
const legacyBackupFormat = "ops-task-manager-export"

type BackupService struct {
	db         *gorm.DB
	appName    string
	appVersion string
}

func NewBackupService(db *gorm.DB, appName, appVersion string) *BackupService {
	return &BackupService{
		db:         db,
		appName:    appName,
		appVersion: appVersion,
	}
}

func (s *BackupService) Export() (*dto.BackupPackage, error) {
	data := dto.BackupData{}

	if err := s.db.Order("sort_order ASC, created_at ASC").Find(&data.Projects).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("created_at ASC").Find(&data.Units).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("operated_at ASC, id ASC").Find(&data.UnitLogs).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("sort_order ASC, created_at ASC").Find(&data.TodoGroups).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("sort_order ASC, created_at ASC").Find(&data.Todos).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("sort_order ASC, created_at ASC").Find(&data.NoteGroups).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("updated_at ASC, created_at ASC").Find(&data.Notes).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("updated_at ASC, id ASC").Find(&data.NoteDrafts).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("triggered_at ASC, id ASC").Find(&data.Notifications).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("start_time ASC, created_at ASC").Find(&data.Schedules).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("created_at ASC, id ASC").Find(&data.ScheduleResources).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("sort_order ASC, created_at ASC").Find(&data.Wallets).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("sort_order ASC, created_at ASC").Find(&data.Categories).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("transaction_at ASC, created_at ASC").Find(&data.Transactions).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("created_at ASC").Find(&data.Secrets).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("created_at ASC, id ASC").Find(&data.SecretAuditLogs).Error; err != nil {
		return nil, err
	}

	if err := s.db.Order("created_at ASC, id ASC").Find(&data.Attachments).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("id ASC").Find(&data.Certificates).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("created_at ASC, id ASC").Find(&data.AttachmentLinks).Error; err != nil {
		return nil, err
	}

	return &dto.BackupPackage{
		BackupMetadata: dto.BackupMetadata{
			Format:        dto.BackupFormat,
			SchemaVersion: dto.BackupSchemaVersion,
			ExportedAt:    timeutil.Now(),
			Encoding:      "utf-8",
			App: dto.BackupApp{
				Name:    s.appName,
				Version: s.appVersion,
			},
		},
		Data: data,
	}, nil
}

func (s *BackupService) Import(strategy string, raw []byte) (*dto.BackupImportResult, error) {
	if strategy != dto.BackupStrategyMerge && strategy != dto.BackupStrategyOverwrite {
		return nil, fmt.Errorf("不支持的导入策略: %s", strategy)
	}

	payload, err := decodeBackupPayload(raw)
	if err != nil {
		return nil, err
	}
	normalizeBackupPackage(&payload)
	if err := validateBackupCertificates(&payload.Data); err != nil {
		return nil, err
	}
	if err := s.validatePackage(&payload); err != nil {
		return nil, err
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if strategy == dto.BackupStrategyOverwrite {
			if err := clearBackupTables(tx); err != nil {
				return err
			}
		}

		if strategy == dto.BackupStrategyMerge {
			if err := mergeBackup(tx, &payload.Data); err != nil {
				return err
			}
			return nil
		}

		return insertBackup(tx, &payload.Data)
	}); err != nil {
		return nil, err
	}

	return &dto.BackupImportResult{
		Strategy: strategy,
		Stats: dto.BackupStats{
			Attachments:       len(payload.Data.Attachments),
			Certificates:      len(payload.Data.Certificates),
			AttachmentLinks:   len(payload.Data.AttachmentLinks),
			Projects:          len(payload.Data.Projects),
			Units:             len(payload.Data.Units),
			UnitLogs:          len(payload.Data.UnitLogs),
			TodoGroups:        len(payload.Data.TodoGroups),
			Todos:             len(payload.Data.Todos),
			NoteGroups:        len(payload.Data.NoteGroups),
			Notes:             len(payload.Data.Notes),
			NoteDrafts:        len(payload.Data.NoteDrafts),
			Notifications:     len(payload.Data.Notifications),
			Schedules:         len(payload.Data.Schedules),
			ScheduleResources: len(payload.Data.ScheduleResources),
			Wallets:           len(payload.Data.Wallets),
			Categories:        len(payload.Data.Categories),
			Transactions:      len(payload.Data.Transactions),
			Secrets:           len(payload.Data.Secrets),
			SecretAuditLogs:   len(payload.Data.SecretAuditLogs),
		},
	}, nil
}

func (s *BackupService) validatePackage(pkg *dto.BackupPackage) error {
	if pkg.Format != "" && pkg.Format != dto.BackupFormat {
		if pkg.Format != legacyBackupFormat {
			return fmt.Errorf("不支持的备份格式: %s", pkg.Format)
		}
	}
	if pkg.SchemaVersion > dto.BackupSchemaVersion {
		return fmt.Errorf("备份 schema_version=%d 高于当前支持版本 %d", pkg.SchemaVersion, dto.BackupSchemaVersion)
	}
	return nil
}

func decodeBackupPayload(raw []byte) (dto.BackupPackage, error) {
	decoded, err := decodeTextBytes(raw)
	if err != nil {
		return dto.BackupPackage{}, err
	}

	var pkg dto.BackupPackage
	if err := json.Unmarshal(decoded, &pkg); err != nil {
		return dto.BackupPackage{}, fmt.Errorf("备份文件 JSON 解析失败: %w", err)
	}
	return pkg, nil
}

func normalizeBackupPackage(pkg *dto.BackupPackage) {
	if pkg.Data.ScheduleResources == nil {
		pkg.Data.ScheduleResources = []model.ScheduleResource{}
	}

	for i := range pkg.Data.Schedules {
		if len(pkg.Data.Schedules[i].Resources) > 0 {
			pkg.Data.ScheduleResources = append(pkg.Data.ScheduleResources, pkg.Data.Schedules[i].Resources...)
			pkg.Data.Schedules[i].Resources = nil
		}
	}
}

func decodeTextBytes(raw []byte) ([]byte, error) {
	if len(raw) >= 3 && raw[0] == 0xef && raw[1] == 0xbb && raw[2] == 0xbf {
		return raw[3:], nil
	}
	if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe {
		return utf16ToUTF8(raw[2:], binary.LittleEndian), nil
	}
	if len(raw) >= 2 && raw[0] == 0xfe && raw[1] == 0xff {
		return utf16ToUTF8(raw[2:], binary.BigEndian), nil
	}
	return raw, nil
}

func utf16ToUTF8(raw []byte, order binary.ByteOrder) []byte {
	if len(raw)%2 == 1 {
		raw = raw[:len(raw)-1]
	}
	u16 := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		u16 = append(u16, order.Uint16(raw[i:i+2]))
	}
	return []byte(string(utf16.Decode(u16)))
}

func clearBackupTables(tx *gorm.DB) error {
	for _, target := range []any{
		&model.AttachmentLink{},
		&model.Certificate{},
		&model.Attachment{},
		&model.SecretAuditLog{},
		&model.UnitLog{},
		&model.Notification{},
		&model.ScheduleResource{},
		&model.Schedule{},
		&model.Transaction{},
		&model.Secret{},
		&model.Note{},
		&model.NoteDraft{},
		&model.NoteGroup{},
		&model.Todo{},
		&model.TodoGroup{},
		&model.Unit{},
		&model.BudgetCategory{},
		&model.Wallet{},
		&model.Project{},
	} {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(target).Error; err != nil {
			return err
		}
	}
	return nil
}

func insertBackup(tx *gorm.DB, data *dto.BackupData) error {
	return createBackupData(tx, data, false)
}

func mergeBackup(tx *gorm.DB, data *dto.BackupData) error {
	if err := createBackupData(tx, data, true); err != nil {
		return err
	}
	return mergeSecretsByName(tx, data.Secrets)
}

func createBackupData(tx *gorm.DB, data *dto.BackupData, upsert bool) error {
	if err := createBatch(tx, data.Attachments, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Certificates, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.AttachmentLinks, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Projects, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.TodoGroups, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.NoteGroups, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Wallets, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Categories, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Units, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Todos, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Notes, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.NoteDrafts, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Schedules, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.ScheduleResources, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.Transactions, upsert); err != nil {
		return err
	}
	if !upsert {
		if err := createBatch(tx, data.Secrets, false); err != nil {
			return err
		}
	}
	if err := createBatch(tx, data.Notifications, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.UnitLogs, upsert); err != nil {
		return err
	}
	if err := createBatch(tx, data.SecretAuditLogs, upsert); err != nil {
		return err
	}
	return nil
}

func createBatch[T any](tx *gorm.DB, items []T, upsert bool) error {
	if len(items) == 0 {
		return nil
	}

	query := tx
	if upsert {
		query = query.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		})
	}
	return query.CreateInBatches(items, backupBatchSize).Error
}

func mergeSecretsByName(tx *gorm.DB, secrets []model.Secret) error {
	if len(secrets) == 0 {
		return nil
	}

	for _, secret := range secrets {
		var existing model.Secret
		err := tx.Where("id = ? OR name = ?", secret.ID, secret.Name).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if err == gorm.ErrRecordNotFound {
			if err := tx.Create(&secret).Error; err != nil {
				return err
			}
			continue
		}

		existing.Name = secret.Name
		existing.Value = secret.Value
		existing.Description = secret.Description
		existing.Tags = secret.Tags
		existing.ProjectID = secret.ProjectID
		existing.CreatedAt = secret.CreatedAt
		existing.UpdatedAt = secret.UpdatedAt

		if err := tx.Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}

func ReadAllBackup(r io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取备份文件失败: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("备份文件为空")
	}
	return raw, nil
}

// Reparse original public files on import so backups cannot inject private keys or
// substitute expiry metadata. Preserve stable IDs and countdown associations.
func validateBackupCertificates(data *dto.BackupData) error {
	byAttachment := make(map[string][]int)
	for i, c := range data.Certificates {
		byAttachment[c.AttachmentID] = append(byAttachment[c.AttachmentID], i)
	}
	seen := make(map[string]bool)
	for i := range data.Attachments {
		a := &data.Attachments[i]
		if seen[a.ID] {
			return fmt.Errorf("重复的附件 ID: %s", a.ID)
		}
		seen[a.ID] = true
		parsed, err := certparse.Parse(a.Content)
		if err != nil {
			return fmt.Errorf("附件 %s 无效: %w", a.Filename, err)
		}
		indices := byAttachment[a.ID]
		if len(indices) != len(parsed) {
			return fmt.Errorf("附件 %s 的证书信息不完整", a.Filename)
		}
		used := make([]bool, len(parsed))
		for _, index := range indices {
			existing := data.Certificates[index]
			matched := false
			for j, c := range parsed {
				if !used[j] && c.Fingerprint == existing.Fingerprint {
					c.ID, c.AttachmentID, c.CountdownUnitID = existing.ID, a.ID, existing.CountdownUnitID
					data.Certificates[index] = c
					used[j] = true
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf("附件 %s 的证书指纹不匹配", a.Filename)
			}
		}
		a.Size = int64(len(a.Content))
		a.ContentType = "application/octet-stream"
		a.Certificates = nil
	}
	for _, c := range data.Certificates {
		if !seen[c.AttachmentID] {
			return fmt.Errorf("证书缺少原始附件: %s", c.ID)
		}
	}
	for _, link := range data.AttachmentLinks {
		if !seen[link.AttachmentID] {
			return fmt.Errorf("关联缺少原始附件: %s", link.ID)
		}
	}
	return nil
}
