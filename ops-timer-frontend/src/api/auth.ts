import { post, get, put } from './client'
import type { LoginResponse, User, TokenResponse } from '@/types'

export const authApi = {
  login: (username: string, password: string) =>
    post<LoginResponse>('/auth/login', { username, password }),

  logout: () => post('/auth/logout'),

  getProfile: () => get<User>('/auth/profile'),

  updateProfile: (data: { username?: string; display_name?: string; email?: string }) =>
    put<User>('/auth/profile', data),

  changePassword: (old_password: string, new_password: string) =>
    put('/auth/password', { old_password, new_password }),

  getToken: () => get<TokenResponse>('/auth/token'),

  regenerateToken: () => post<TokenResponse>('/auth/token/regenerate'),

  /** 发送测试邮件到当前用户的通知邮箱 */
  testEmail: () => post<{ message: string }>('/auth/test-email'),

  /** 查询 SMTP 是否已在服务端配置 */
  smtpStatus: () => get<{ enabled: boolean }>('/auth/smtp-status'),
}
