import { del, get, post, put } from './client'
import type {
  CreateNoteRequest,
  Note,
  NoteGroup,
  NoteQueryParams,
  NoteSearchQueryParams,
  UpdateNoteRequest,
} from '@/types'

export const noteApi = {
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
