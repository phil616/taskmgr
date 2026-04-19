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
	ErrNoteNotFound      = errors.New("笔记不存在")
	ErrNoteGroupNotFound = errors.New("笔记分组不存在")
)

type NoteService struct {
	noteRepo  *repository.NoteRepository
	groupRepo *repository.NoteGroupRepository
}

func NewNoteService(noteRepo *repository.NoteRepository, groupRepo *repository.NoteGroupRepository) *NoteService {
	return &NoteService{noteRepo: noteRepo, groupRepo: groupRepo}
}

func (s *NoteService) Create(req *dto.CreateNoteRequest) (*dto.NoteResponse, error) {
	if req.GroupID != nil && *req.GroupID != "" {
		if _, err := s.groupRepo.FindByID(*req.GroupID); err != nil {
			return nil, ErrNoteGroupNotFound
		}
	}

	note := &model.Note{
		ID:      uuid.New().String(),
		GroupID: req.GroupID,
		Title:   req.Title,
		Content: req.Content,
		Tags:    model.JSONStringArray(req.Tags),
	}

	if err := s.noteRepo.Create(note); err != nil {
		return nil, err
	}

	note, _ = s.noteRepo.FindByID(note.ID)
	return s.toResponse(note), nil
}

func (s *NoteService) GetByID(id string) (*dto.NoteResponse, error) {
	note, err := s.noteRepo.FindByID(id)
	if err != nil {
		return nil, ErrNoteNotFound
	}
	return s.toResponse(note), nil
}

func (s *NoteService) List(params *dto.NoteQueryParams) ([]dto.NoteResponse, int64, error) {
	page, pageSize := sanitizePage(params.Page, params.PageSize)

	notes, total, err := s.noteRepo.List(repository.NoteFilter{
		GroupID:  params.GroupID,
		Tag:      params.Tag,
		Keyword:  params.Keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.NoteResponse, len(notes))
	for i, note := range notes {
		responses[i] = *s.toResponse(&note)
	}
	return responses, total, nil
}

func (s *NoteService) Search(params *dto.NoteSearchQueryParams) ([]dto.NoteResponse, int64, error) {
	page, pageSize := sanitizePage(params.Page, params.PageSize)

	notes, total, err := s.noteRepo.List(repository.NoteFilter{
		GroupID:  params.GroupID,
		Tag:      params.Tag,
		Keyword:  params.Query,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	responses := make([]dto.NoteResponse, len(notes))
	for i, note := range notes {
		responses[i] = *s.toResponse(&note)
	}
	return responses, total, nil
}

func (s *NoteService) Update(id string, req *dto.UpdateNoteRequest) (*dto.NoteResponse, error) {
	note, err := s.noteRepo.FindByID(id)
	if err != nil {
		return nil, ErrNoteNotFound
	}

	if req.GroupID != nil {
		if *req.GroupID == "" {
			note.GroupID = nil
		} else {
			if _, err := s.groupRepo.FindByID(*req.GroupID); err != nil {
				return nil, ErrNoteGroupNotFound
			}
			note.GroupID = req.GroupID
		}
	}
	if req.Title != nil {
		note.Title = *req.Title
	}
	if req.Content != nil {
		note.Content = *req.Content
	}
	if req.Tags != nil {
		note.Tags = model.JSONStringArray(*req.Tags)
	}

	if err := s.noteRepo.Update(note); err != nil {
		return nil, err
	}

	note, _ = s.noteRepo.FindByID(note.ID)
	return s.toResponse(note), nil
}

func (s *NoteService) Delete(id string) error {
	if _, err := s.noteRepo.FindByID(id); err != nil {
		return ErrNoteNotFound
	}
	return s.noteRepo.Delete(id)
}

func (s *NoteService) CreateGroup(req *dto.CreateNoteGroupRequest) (*dto.NoteGroupResponse, error) {
	group := &model.NoteGroup{
		ID:    uuid.New().String(),
		Name:  req.Name,
		Color: req.Color,
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}

	if err := s.groupRepo.Create(group); err != nil {
		return nil, err
	}
	return s.toGroupResponse(group), nil
}

func (s *NoteService) ListGroups() ([]dto.NoteGroupResponse, error) {
	groups, err := s.groupRepo.List()
	if err != nil {
		return nil, err
	}

	responses := make([]dto.NoteGroupResponse, len(groups))
	for i, group := range groups {
		responses[i] = *s.toGroupResponse(&group)
	}
	return responses, nil
}

func (s *NoteService) UpdateGroup(id string, req *dto.UpdateNoteGroupRequest) (*dto.NoteGroupResponse, error) {
	group, err := s.groupRepo.FindByID(id)
	if err != nil {
		return nil, ErrNoteGroupNotFound
	}

	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Color != nil {
		group.Color = *req.Color
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}

	if err := s.groupRepo.Update(group); err != nil {
		return nil, err
	}
	return s.toGroupResponse(group), nil
}

func (s *NoteService) DeleteGroup(id string) error {
	if _, err := s.groupRepo.FindByID(id); err != nil {
		return ErrNoteGroupNotFound
	}
	_ = s.noteRepo.ClearGroupNotes(id)
	return s.groupRepo.Delete(id)
}

func (s *NoteService) toResponse(note *model.Note) *dto.NoteResponse {
	resp := &dto.NoteResponse{
		ID:        note.ID,
		GroupID:   note.GroupID,
		Title:     note.Title,
		Content:   note.Content,
		Tags:      []string(note.Tags),
		CreatedAt: timeutil.Normalize(note.CreatedAt),
		UpdatedAt: timeutil.Normalize(note.UpdatedAt),
	}
	if note.Group != nil {
		resp.GroupName = note.Group.Name
	}
	if resp.Tags == nil {
		resp.Tags = []string{}
	}
	return resp
}

func (s *NoteService) toGroupResponse(group *model.NoteGroup) *dto.NoteGroupResponse {
	return &dto.NoteGroupResponse{
		ID:        group.ID,
		Name:      group.Name,
		Color:     group.Color,
		SortOrder: group.SortOrder,
		NoteCount: group.NoteCount,
		CreatedAt: timeutil.Normalize(group.CreatedAt),
		UpdatedAt: timeutil.Normalize(group.UpdatedAt),
	}
}

func sanitizePage(page, pageSize int) (int, int) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	return page, pageSize
}
