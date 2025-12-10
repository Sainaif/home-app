import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api/client'

// Safe JSON parse helper - returns default value if parsing fails
function safeJsonParse(value, defaultValue) {
  if (!value) return defaultValue
  try {
    return JSON.parse(value)
  } catch (e) {
    console.warn('Failed to parse localStorage value:', e)
    return defaultValue
  }
}

// Auth store - handles login, logout, and user permissions
export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref(localStorage.getItem('accessToken'))
  const refreshToken = ref(localStorage.getItem('refreshToken'))
  const user = ref(safeJsonParse(localStorage.getItem('user'), null))
  const permissions = ref(safeJsonParse(localStorage.getItem('permissions'), []))
  const mustChangePassword = ref(localStorage.getItem('mustChangePassword') === 'true')

  // Auth configuration from server
  const authConfig = ref({
    allowEmailLogin: true,
    allowUsernameLogin: false,
    requireUsername: false,
    twoFAEnabled: false,
    vapidPublicKey: ''
  })

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdmin = computed(() => user.value?.role === 'ADMIN')

  // hasPermission checks if user has a specific permission
  const hasPermission = (permission) => {
    return permissions.value.includes(permission)
  }

  // fetchAuthConfig gets auth configuration from server
  async function fetchAuthConfig() {
    try {
      const response = await api.get('/auth/config')
      authConfig.value = response.data
    } catch (error) {
      console.warn('Failed to fetch auth config, using defaults:', error)
    }
    return authConfig.value
  }

  // login with identifier (email or username) and password
  async function login(identifier, password) {
    const response = await api.post('/auth/login', {
      identifier,
      email: identifier, // Backward compatibility with older backends
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
    mustChangePassword.value = response.data.mustChangePassword === true || response.data.mustChangePassword === 'true'

    return response.data
  }

  // refresh gets new access token (called automatically when token expires)
  async function refresh() {
    try {
      const response = await api.post('/auth/refresh', {
        refreshToken: refreshToken.value
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

  // logout clears everything and revokes session on server
  async function logout() {
    // Try to revoke session on server (fire and forget)
    if (refreshToken.value) {
      try {
        await api.post('/auth/logout', { refreshToken: refreshToken.value })
      } catch (err) {
        // Ignore errors - we're logging out anyway
        console.warn('Failed to revoke session on server:', err)
      }
    }

    accessToken.value = null
    refreshToken.value = null
    user.value = null
    permissions.value = []

    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('user')
    localStorage.removeItem('permissions')
    localStorage.removeItem('mustChangePassword')
    mustChangePassword.value = false
  }

  // clearMustChangePassword updates tokens after password change
  function clearMustChangePassword(newAccessToken, newRefreshToken) {
    accessToken.value = newAccessToken
    refreshToken.value = newRefreshToken
    mustChangePassword.value = false

    localStorage.setItem('accessToken', newAccessToken)
    localStorage.setItem('refreshToken', newRefreshToken)
    localStorage.setItem('mustChangePassword', 'false')
  }

  function setTokens(access, refresh) {
    accessToken.value = access
    refreshToken.value = refresh

    localStorage.setItem('accessToken', access)
    localStorage.setItem('refreshToken', refresh)
  }

  // loadUser fetches current user info from API
  async function loadUser() {
    const userResponse = await api.get('/users/me', {
      headers: { Authorization: `Bearer ${accessToken.value}` }
    })
    user.value = userResponse.data
    permissions.value = userResponse.data.permissions || []
    localStorage.setItem('user', JSON.stringify(user.value))
    localStorage.setItem('permissions', JSON.stringify(permissions.value))
  }

  // validateSession checks if tokens are still valid
  async function validateSession() {
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
    authConfig,
    mustChangePassword,
    isAuthenticated,
    isAdmin,
    hasPermission,
    fetchAuthConfig,
    login,
    refresh,
    logout,
    setTokens,
    loadUser,
    validateSession,
    clearMustChangePassword
  }
})
