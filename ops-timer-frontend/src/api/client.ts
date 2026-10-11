import axios from 'axios'
import { getActivePinia } from 'pinia'
import type { ApiResponse } from '@/types'
import router from '@/router'

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
  withCredentials: true,
  headers: { 'Content-Type': 'application/json', 'X-Taskmgr-Browser': '1' },
})

// 防止多个并发 401 请求重复触发跳转
let isHandling401 = false

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 && !error.config?.url?.startsWith('/auth/browser/') && !isHandling401) {
      isHandling401 = true

      // 清除本地存储
      localStorage.removeItem('token')
      localStorage.removeItem('user')

      // 同步清空 Pinia 中的用户，避免路由守卫误判已登录
      const pinia = getActivePinia()
      if (pinia?.state.value?.['auth']) {
        pinia.state.value['auth'].initialized = true
        pinia.state.value['auth'].user = null
      }

      // During initial navigation the guard handles the redirect after profile loading.
      if (router.currentRoute.value.matched.length && router.currentRoute.value.path !== '/login') {
        router.push({ path: '/login', query: { redirect: router.currentRoute.value.fullPath } }).finally(() => { isHandling401 = false })
      } else {
        isHandling401 = false
      }
    }
    return Promise.reject(error)
  }
)

export async function get<T>(url: string, params?: any): Promise<ApiResponse<T>> {
  const { data } = await client.get<ApiResponse<T>>(url, { params })
  return data
}

export async function post<T>(url: string, body?: any): Promise<ApiResponse<T>> {
  const { data } = await client.post<ApiResponse<T>>(url, body)
  return data
}

export async function put<T>(url: string, body?: any): Promise<ApiResponse<T>> {
  const { data } = await client.put<ApiResponse<T>>(url, body)
  return data
}

export async function patch<T>(url: string, body?: any): Promise<ApiResponse<T>> {
  const { data } = await client.patch<ApiResponse<T>>(url, body)
  return data
}

export async function del<T>(url: string): Promise<any> {
  const { data } = await client.delete<ApiResponse<T>>(url)
  return data
}

export default client
