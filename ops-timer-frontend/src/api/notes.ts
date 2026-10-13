import { del, get, post, put } from './client'
import type {
  CreateNoteRequest,
  Note,
  NoteGroup,
  NoteQueryParams,
  NoteSearchQueryParams,
  UpdateNoteRequest,
} from '@/types'

export interface NoteDraftPayload {
  note_id: string | null
  title: string
  content: string
  group_id: string | null
  tags: string[]
}
export interface NoteDraft extends NoteDraftPayload {
  id: string
  created_at: string
  updated_at: string
}

export const noteApi = {
  listDrafts: () => get<NoteDraft[]>('/note-drafts'),
  saveDraft: (data: NoteDraftPayload, id?: string) => id
    ? put<NoteDraft>(`/note-drafts/${id}`, data)
    : post<NoteDraft>('/note-drafts', data),
  deleteDraft: (id: string) => del(`/note-drafts/${id}`),
  publishDraft: (id: string, data: NoteDraftPayload) => post<Note>(`/note-drafts/${id}/publish`, data),
  list: (params?: NoteQueryParams) =>
    get<Note[]>('/notes', params),

  search: (params: NoteSearchQueryParams) =>
    get<Note[]>('/notes/search', params),

  create: (data: CreateNoteRequest) =>
    post<Note>('/notes', data),

  getById: (id: string) =>
    get<Note>(`/notes/${id}`),

  update: (id: string, data: UpdateNoteRequest) =>
    put<Note>(`/notes/${id}`, data),

  delete: (id: string) =>
    del(`/notes/${id}`),

  listGroups: () =>
    get<NoteGroup[]>('/note-groups'),

  createGroup: (data: { name: string; color?: string; sort_order?: number }) =>
    post<NoteGroup>('/note-groups', data),

  updateGroup: (id: string, data: { name?: string; color?: string; sort_order?: number }) =>
    put<NoteGroup>(`/note-groups/${id}`, data),

  deleteGroup: (id: string) =>
    del(`/note-groups/${id}`),
}
