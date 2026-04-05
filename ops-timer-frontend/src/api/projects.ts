import { get, post, put, patch, del } from './client'
import type { Project, Unit, ProjectBudgetStats, Transaction, TransactionQueryParams } from '@/types'

export const projectApi = {
  list: (params?: Record<string, any>) => get<Project[]>('/projects', params),

  create: (data: any) => post<Project>('/projects', data),

  getById: (id: string) => get<Project>(`/projects/${id}`),

  update: (id: string, data: any) => put<Project>(`/projects/${id}`, data),

  patchUpdate: (id: string, data: any) => patch<Project>(`/projects/${id}`, data),

  delete: (id: string) => del(`/projects/${id}`),

  getUnits: (id: string, params?: Record<string, any>) =>
    get<Unit[]>(`/projects/${id}/units`, params),

  getBudgetStats: (id: string) =>
    get<ProjectBudgetStats>(`/projects/${id}/budget`),
}
