import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import { authApi } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<User | null>(JSON.parse(localStorage.getItem('user') || 'null'))

  const isLoggedIn = computed(() => !!token.value)

  async function login(username: string, password: string) {
    const resp = await authApi.login(username, password)
    token.value = resp.data.token
    user.value = resp.data.user
    localStorage.setItem('token', resp.data.token)
    localStorage.setItem('user', JSON.stringify(resp.data.user))
  }

  /** 仅清理本地状态，不调用后端接口（供 401 拦截器等场景使用） */
  function clearAuth() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch {
      // ignore
    }
    clearAuth()
  }

  async function fetchProfile() {
    const resp = await authApi.getProfile()
    user.value = resp.data
    localStorage.setItem('user', JSON.stringify(resp.data))
  }

  return { token, user, isLoggedIn, login, logout, clearAuth, fetchProfile }
})
