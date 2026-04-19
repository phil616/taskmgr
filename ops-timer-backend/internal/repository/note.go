package repository

import (
	"ops-timer-backend/internal/model"

	"gorm.io/gorm"
)

type NoteRepository struct {
	db *gorm.DB
}

func NewNoteRepository(db *gorm.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) Create(note *model.Note) error {
	return r.db.Create(note).Error
}

func (r *NoteRepository) FindByID(id string) (*model.Note, error) {
	var note model.Note
	err := r.db.Preload("Group").First(&note, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

type NoteFilter struct {
	GroupID  string
	Tag      string
	Keyword  string
	Page     int
	PageSize int
}

func (r *NoteRepository) List(filter NoteFilter) ([]model.Note, int64, error) {
	var notes []model.Note
	var total int64

	query := r.db.Model(&model.Note{}).Joins("LEFT JOIN note_groups ON note_groups.id = notes.group_id")

	if filter.GroupID != "" {
		if filter.GroupID == "none" {
			query = query.Where("notes.group_id IS NULL")
		} else {
			query = query.Where("notes.group_id = ?", filter.GroupID)
		}
	}
	if filter.Tag != "" {
		query = query.Where("notes.tags LIKE ?", "%\""+filter.Tag+"\"%")
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where(
			"notes.title LIKE ? OR notes.content LIKE ? OR notes.tags LIKE ? OR note_groups.name LIKE ?",
			keyword, keyword, keyword, keyword,
		)
	}

	query.Count(&total)

	err := query.Preload("Group").
		Order("notes.updated_at DESC, notes.created_at DESC").
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		Find(&notes).Error

	return notes, total, err
}

func (r *NoteRepository) Update(note *model.Note) error {
	return r.db.Save(note).Error
}

func (r *NoteRepository) Delete(id string) error {
	return r.db.Delete(&model.Note{}, "id = ?", id).Error
}

func (r *NoteRepository) ClearGroupNotes(groupID string) error {
	return r.db.Model(&model.Note{}).
		Where("group_id = ?", groupID).
		Update("group_id", nil).Error
}

type NoteGroupRepository struct {
	db *gorm.DB
}

func NewNoteGroupRepository(db *gorm.DB) *NoteGroupRepository {
	return &NoteGroupRepository{db: db}
}

func (r *NoteGroupRepository) Create(group *model.NoteGroup) error {
	return r.db.Create(group).Error
}

func (r *NoteGroupRepository) FindByID(id string) (*model.NoteGroup, error) {
	var group model.NoteGroup
	err := r.db.First(&group, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *NoteGroupRepository) List() ([]model.NoteGroup, error) {
	var groups []model.NoteGroup
	err := r.db.Model(&model.NoteGroup{}).
		Select("note_groups.*, COUNT(notes.id) AS note_count").
		Joins("LEFT JOIN notes ON notes.group_id = note_groups.id AND notes.deleted_at IS NULL").
		Group("note_groups.id").
		Order("note_groups.sort_order ASC, note_groups.created_at DESC").
		Find(&groups).Error
	return groups, err
}

func (r *NoteGroupRepository) Update(group *model.NoteGroup) error {
	return r.db.Save(group).Error
}

func (r *NoteGroupRepository) Delete(id string) error {
	return r.db.Delete(&model.NoteGroup{}, "id = ?", id).Error
}
