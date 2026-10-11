import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import { authApi } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  // Retire the old script-readable credentials. Authentication now uses an HttpOnly cookie.
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const isLoggedIn = computed(() => !!user.value)
  let initialization: Promise<void> | null = null

  async function login(username: string, password: string, rememberMe = false) {
    const resp = await authApi.login(username, password, rememberMe)
    user.value = resp.data.user
    initialized.value = true
  }

  function clearAuth() {
    user.value = null
    initialized.value = true
  }

  async function logout() {
    // Do not report success when the server could not revoke the persistent cookie.
    await authApi.logout()
    clearAuth()
  }

  async function fetchProfile() {
    const resp = await authApi.getProfile()
    user.value = resp.data
  }

  async function initialize() {
    if (initialized.value) return
    if (!initialization) {
      initialization = fetchProfile().then(() => { initialized.value = true }).catch((error) => {
        if (error.response?.status === 401) clearAuth()
        // A network failure isn't proof that the cookie expired; permit a later retry.
      }).finally(() => { initialization = null })
    }
    await initialization
  }

  return { user, isLoggedIn, initialized, initialize, login, logout, clearAuth, fetchProfile }
})
