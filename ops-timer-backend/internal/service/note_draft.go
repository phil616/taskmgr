package service

import (
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/repository"
	"strings"
)

var ErrNoteDraftNotFound = errors.New("草稿不存在，可能已被提交或删除")
var ErrNoteDraftIncomplete = errors.New("提交笔记前请填写标题和正文")

func (s *NoteService) ListDrafts() ([]model.NoteDraft, error) { return s.noteRepo.ListDrafts() }

func (s *NoteService) SaveDraft(id string, req *dto.SaveNoteDraftRequest) (*model.NoteDraft, error) {
	draft := &model.NoteDraft{ID: uuid.NewString()}
	if id != "" {
		var err error
		draft, err = s.noteRepo.FindDraft(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteDraftNotFound
		}
		if err != nil {
			return nil, err
		}
	}
	draft.NoteID, draft.GroupID = req.NoteID, req.GroupID
	draft.Title, draft.Content, draft.Tags = req.Title, req.Content, model.JSONStringArray(req.Tags)
	if draft.Tags == nil {
		draft.Tags = model.JSONStringArray{}
	}
	var err error
	if id == "" {
		err = s.noteRepo.CreateDraft(draft)
	} else {
		err = s.noteRepo.UpdateDraft(draft)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNoteDraftNotFound
	}
	if err != nil {
		return nil, err
	}
	return draft, nil
}

func (s *NoteService) DeleteDraft(id string) error {
	err := s.noteRepo.DeleteDraft(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNoteDraftNotFound
	}
	return err
}

// Publishing is atomic: failed note validation/writes always preserve the draft.
func (s *NoteService) PublishDraft(id string, req *dto.SaveNoteDraftRequest) (*dto.NoteResponse, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		return nil, ErrNoteDraftIncomplete
	}
	var result *dto.NoteResponse
	err := s.noteRepo.Transaction(func(notes *repository.NoteRepository, groups *repository.NoteGroupRepository) error {
		if _, err := notes.FindDraft(id); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoteDraftNotFound
			}
			return err
		}
		svc := NewNoteService(notes, groups)
		var err error
		if req.NoteID == nil {
			result, err = svc.Create(&dto.CreateNoteRequest{Title: strings.TrimSpace(req.Title), Content: req.Content, GroupID: req.GroupID, Tags: req.Tags})
		} else {
			groupID := ""
			if req.GroupID != nil {
				groupID = *req.GroupID
			}
			title := strings.TrimSpace(req.Title)
			result, err = svc.Update(*req.NoteID, &dto.UpdateNoteRequest{Title: &title, Content: &req.Content, GroupID: &groupID, Tags: &req.Tags})
		}
		if err != nil {
			return err
		}
		return svc.DeleteDraft(id)
	})
	return result, err
}
