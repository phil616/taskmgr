import { get, post, put, del } from './client'
import type {
  Schedule,
  ScheduleResource,
  CreateScheduleRequest,
  UpdateScheduleRequest,
  AddScheduleResourceRequest,
  ScheduleQueryParams,
} from '@/types'

export const scheduleApi = {
  /** 获取日程列表（可按日期范围过滤） */
  list: (params?: ScheduleQueryParams) =>
    get<Schedule[]>('/schedules', params as Record<string, any>),

  /** 创建日程 */
  create: (data: CreateScheduleRequest) => post<Schedule>('/schedules', data),

  /** 获取日程详情（含关联资源） */
  getById: (id: string) => get<Schedule>(`/schedules/${id}`),

  /** 更新日程 */
  update: (id: string, data: UpdateScheduleRequest) =>
    put<Schedule>(`/schedules/${id}`, data),

  /** 删除日程 */
  delete: (id: string) => del(`/schedules/${id}`),

  /** 给日程添加关联资源 */
  addResource: (scheduleId: string, data: AddScheduleResourceRequest) =>
    post<ScheduleResource>(`/schedules/${scheduleId}/resources`, data),

  /** 从日程移除关联资源 */
  removeResource: (scheduleId: string, resourceId: string) =>
    del(`/schedules/${scheduleId}/resources/${resourceId}`),
}
