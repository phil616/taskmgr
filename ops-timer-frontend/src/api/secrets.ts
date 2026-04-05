import { get, post, put, del } from './client'
import type { Secret, SecretBrief, SecretAuditLog, SecretQueryParams } from '@/types'

export const secretApi = {
  list: (params?: SecretQueryParams) =>
    get<SecretBrief[]>('/secrets', params),

  create: (data: any) =>
    post<Secret>('/secrets', data),

  getById: (id: string) =>
    get<Secret>(`/secrets/${id}`),

  getValue: (id: string) =>
    get<{ id: string; name: string; value: string }>(`/secrets/${id}/value`),

  update: (id: string, data: any) =>
    put<Secret>(`/secrets/${id}`, data),

  delete: (id: string) =>
    del(`/secrets/${id}`),

  getAuditLogs: (id: string, params?: Record<string, any>) =>
    get<SecretAuditLog[]>(`/secrets/${id}/audit-logs`, params),

  getAllAuditLogs: (params?: Record<string, any>) =>
    get<SecretAuditLog[]>('/secret-audit-logs', params),
}
