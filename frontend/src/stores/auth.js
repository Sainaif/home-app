import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api/client'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref(localStorage.getItem('accessToken'))
  const refreshToken = ref(localStorage.getItem('refreshToken'))
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))
  const permissions = ref(JSON.parse(localStorage.getItem('permissions') || '[]'))

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdmin = computed(() => user.value?.role === 'ADMIN')

  const hasPermission = (permission) => {
    return permissions.value.includes(permission)
  }

  async function login(email, password) {
    const response = await api.post('/auth/login', {
      email,
      password
    })

    accessToken.value = response.data.access
    refreshToken.value = response.data.refresh

    // Fetch user data with permissions
    const userResponse = await api.get('/users/me', {
      headers: { Authorization: `Bearer ${accessToken.value}` }
    })
    user.value = userResponse.data
    permissions.value = userResponse.data.permissions || []

    localStorage.setItem('accessToken', accessToken.value)
    localStorage.setItem('refreshToken', refreshToken.value)
    localStorage.setItem('user', JSON.stringify(user.value))
    localStorage.setItem('permissions', JSON.stringify(permissions.value))
    localStorage.setItem('mustChangePassword', response.data.mustChangePassword)

    return response.data
  }

  async function refresh() {
    try {
      const response = await api.post('/auth/refresh', {
        refresh_token: refreshToken.value
      })

      accessToken.value = response.data.access
      refreshToken.value = response.data.refresh

      localStorage.setItem('accessToken', accessToken.value)
      localStorage.setItem('refreshToken', refreshToken.value)

      return true
    } catch (error) {
      logout()
      return false
    }
  }

  function logout() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    permissions.value = []

    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('user')
    localStorage.removeItem('permissions')
  }

  function setTokens(access, refresh) {
    accessToken.value = access
    refreshToken.value = refresh

    localStorage.setItem('accessToken', access)
    localStorage.setItem('refreshToken', refresh)
  }

  async function loadUser() {
    const userResponse = await api.get('/users/me', {
      headers: { Authorization: `Bearer ${accessToken.value}` }
    })
    user.value = userResponse.data
    permissions.value = userResponse.data.permissions || []
    localStorage.setItem('user', JSON.stringify(user.value))
    localStorage.setItem('permissions', JSON.stringify(permissions.value))
  }

  async function validateSession() {
    // Try to fetch current user data
    // If the access token is expired, the API interceptor will automatically
    // refresh it using the refresh token
    // If the refresh token is also expired, this will throw an error
    try {
      await loadUser()
      return true
    } catch (error) {
      // If we get here, both tokens are invalid
      throw error
    }
  }

  return {
    accessToken,
    refreshToken,
    user,
    permissions,
    isAuthenticated,
    isAdmin,
    hasPermission,
    login,
    refresh,
    logout,
    setTokens,
    loadUser,
    validateSession
  }
})