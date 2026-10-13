package repository

import (
	"gorm.io/gorm"
	"ops-timer-backend/internal/model"
)

func (r *NoteRepository) ListDrafts() ([]model.NoteDraft, error) {
	drafts := []model.NoteDraft{}
	err := r.db.Order("updated_at DESC, id ASC").Find(&drafts).Error
	return drafts, err
}

func (r *NoteRepository) FindDraft(id string) (*model.NoteDraft, error) {
	var draft model.NoteDraft
	err := r.db.First(&draft, "id = ?", id).Error
	return &draft, err
}

func (r *NoteRepository) CreateDraft(draft *model.NoteDraft) error {
	return r.db.Create(draft).Error
}

func (r *NoteRepository) UpdateDraft(draft *model.NoteDraft) error {
	// Do not recreate a draft concurrently removed/published by another client.
	result := r.db.Model(&model.NoteDraft{}).Where("id = ?", draft.ID).
		Select("NoteID", "GroupID", "Title", "Content", "Tags", "UpdatedAt").Updates(draft)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *NoteRepository) DeleteDraft(id string) error {
	result := r.db.Delete(&model.NoteDraft{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *NoteRepository) Transaction(fn func(*NoteRepository, *NoteGroupRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewNoteRepository(tx), NewNoteGroupRepository(tx))
	})
}
