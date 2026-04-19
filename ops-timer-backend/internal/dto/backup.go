package dto

import (
	"time"

	"ops-timer-backend/internal/model"
)

const (
	BackupFormat        = "ops-task-manager-backup"
	BackupSchemaVersion = 1

	BackupStrategyMerge     = "merge"
	BackupStrategyOverwrite = "overwrite"
)

type BackupMetadata struct {
	Format        string    `json:"format"`
	SchemaVersion int       `json:"schema_version"`
	ExportedAt    time.Time `json:"exported_at"`
	Encoding      string    `json:"encoding"`
	App           BackupApp `json:"app"`
}

type BackupApp struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type BackupData struct {
	Projects         []model.Project         `json:"projects"`
	Units            []model.Unit            `json:"units"`
	UnitLogs         []model.UnitLog         `json:"unit_logs"`
	TodoGroups       []model.TodoGroup       `json:"todo_groups"`
	Todos            []model.Todo            `json:"todos"`
	NoteGroups       []model.NoteGroup       `json:"note_groups"`
	Notes            []model.Note            `json:"notes"`
	Notifications    []model.Notification    `json:"notifications"`
	Schedules        []model.Schedule        `json:"schedules"`
	ScheduleResources []model.ScheduleResource `json:"schedule_resources"`
	Wallets          []model.Wallet          `json:"wallets"`
	Categories       []model.BudgetCategory  `json:"categories"`
	Transactions     []model.Transaction     `json:"transactions"`
	Secrets          []model.Secret          `json:"secrets"`
	SecretAuditLogs  []model.SecretAuditLog  `json:"secret_audit_logs"`
}

type BackupPackage struct {
	BackupMetadata
	Data BackupData `json:"data"`
}

type BackupImportResult struct {
	Strategy string         `json:"strategy"`
	Stats    BackupStats    `json:"stats"`
}

type BackupStats struct {
	Projects          int `json:"projects"`
	Units             int `json:"units"`
	UnitLogs          int `json:"unit_logs"`
	TodoGroups        int `json:"todo_groups"`
	Todos             int `json:"todos"`
	NoteGroups        int `json:"note_groups"`
	Notes             int `json:"notes"`
	Notifications     int `json:"notifications"`
	Schedules         int `json:"schedules"`
	ScheduleResources int `json:"schedule_resources"`
	Wallets           int `json:"wallets"`
	Categories        int `json:"categories"`
	Transactions      int `json:"transactions"`
	Secrets           int `json:"secrets"`
	SecretAuditLogs   int `json:"secret_audit_logs"`
}
