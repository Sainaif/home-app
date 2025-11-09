import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'
import api from '../api/client'

vi.mock('../api/client')

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  describe('Initial State', () => {
    it('should start with no authentication', () => {
      const authStore = useAuthStore()
      expect(authStore.isAuthenticated).toBe(false)
      expect(authStore.user).toBeNull()
      expect(authStore.isAdmin).toBe(false)
    })

    it('should load tokens from localStorage if available', () => {
      localStorage.setItem('accessToken', 'test-access-token')
      localStorage.setItem('refreshToken', 'test-refresh-token')
      localStorage.setItem('user', JSON.stringify({ id: '1', email: 'test@example.com', role: 'ADMIN' }))
      localStorage.setItem('permissions', JSON.stringify(['bills.create', 'users.read']))

      const authStore = useAuthStore()

      expect(authStore.accessToken).toBe('test-access-token')
      expect(authStore.refreshToken).toBe('test-refresh-token')
      expect(authStore.user).toEqual({ id: '1', email: 'test@example.com', role: 'ADMIN' })
      expect(authStore.permissions).toEqual(['bills.create', 'users.read'])
      expect(authStore.isAuthenticated).toBe(true)
      expect(authStore.isAdmin).toBe(true)
    })
  })

  describe('login', () => {
    it('should login successfully and store tokens', async () => {
      const authStore = useAuthStore()

      const mockLoginResponse = {
        data: {
          access: 'new-access-token',
          refresh: 'new-refresh-token',
          mustChangePassword: false
        }
      }

      const mockUserResponse = {
        data: {
          id: '1',
          email: 'test@example.com',
          name: 'Test User',
          role: 'RESIDENT',
          permissions: ['bills.create', 'bills.read']
        }
      }

      api.post.mockResolvedValueOnce(mockLoginResponse)
      api.get.mockResolvedValueOnce(mockUserResponse)

      await authStore.login('test@example.com', 'password123')

      expect(api.post).toHaveBeenCalledWith('/auth/login', {
        email: 'test@example.com',
        password: 'password123'
      })
      expect(api.get).toHaveBeenCalledWith('/users/me', {
        headers: { Authorization: 'Bearer new-access-token' }
      })

      expect(authStore.accessToken).toBe('new-access-token')
      expect(authStore.refreshToken).toBe('new-refresh-token')
      expect(authStore.user).toEqual(mockUserResponse.data)
      expect(authStore.permissions).toEqual(['bills.create', 'bills.read'])
      expect(authStore.isAuthenticated).toBe(true)

      expect(localStorage.getItem('accessToken')).toBe('new-access-token')
      expect(localStorage.getItem('refreshToken')).toBe('new-refresh-token')
      expect(JSON.parse(localStorage.getItem('user'))).toEqual(mockUserResponse.data)
      expect(JSON.parse(localStorage.getItem('permissions'))).toEqual(['bills.create', 'bills.read'])
    })
  })

  describe('logout', () => {
    it('should clear all auth data', () => {
      const authStore = useAuthStore()

      // Set up authenticated state
      authStore.accessToken = 'test-token'
      authStore.refreshToken = 'test-refresh'
      authStore.user = { id: '1', email: 'test@example.com' }
      authStore.permissions = ['bills.create']
      localStorage.setItem('accessToken', 'test-token')
      localStorage.setItem('refreshToken', 'test-refresh')
      localStorage.setItem('user', JSON.stringify({ id: '1' }))
      localStorage.setItem('permissions', JSON.stringify(['bills.create']))

      authStore.logout()

      expect(authStore.accessToken).toBeNull()
      expect(authStore.refreshToken).toBeNull()
      expect(authStore.user).toBeNull()
      expect(authStore.permissions).toEqual([])
      expect(authStore.isAuthenticated).toBe(false)

      expect(localStorage.getItem('accessToken')).toBeNull()
      expect(localStorage.getItem('refreshToken')).toBeNull()
      expect(localStorage.getItem('user')).toBeNull()
      expect(localStorage.getItem('permissions')).toBeNull()
    })
  })

  describe('hasPermission', () => {
    it('should return true if user has the permission', () => {
      const authStore = useAuthStore()
      authStore.permissions = ['bills.create', 'bills.read', 'users.update']

      expect(authStore.hasPermission('bills.create')).toBe(true)
      expect(authStore.hasPermission('bills.read')).toBe(true)
      expect(authStore.hasPermission('users.update')).toBe(true)
    })

    it('should return false if user does not have the permission', () => {
      const authStore = useAuthStore()
      authStore.permissions = ['bills.read']

      expect(authStore.hasPermission('bills.create')).toBe(false)
      expect(authStore.hasPermission('users.delete')).toBe(false)
    })
  })

  describe('isAdmin computed', () => {
    it('should return true for ADMIN role', () => {
      const authStore = useAuthStore()
      authStore.user = { id: '1', role: 'ADMIN' }

      expect(authStore.isAdmin).toBe(true)
    })

    it('should return false for non-ADMIN roles', () => {
      const authStore = useAuthStore()
      authStore.user = { id: '1', role: 'RESIDENT' }

      expect(authStore.isAdmin).toBe(false)
    })

    it('should return false when user is null', () => {
      const authStore = useAuthStore()
      authStore.user = null

      expect(authStore.isAdmin).toBe(false)
    })
  })

  describe('refresh', () => {
    it('should refresh tokens successfully', async () => {
      const authStore = useAuthStore()
      authStore.refreshToken = 'old-refresh-token'

      const mockResponse = {
        data: {
          access: 'new-access-token',
          refresh: 'new-refresh-token'
        }
      }

      api.post.mockResolvedValueOnce(mockResponse)

      const result = await authStore.refresh()

      expect(result).toBe(true)
      expect(api.post).toHaveBeenCalledWith('/auth/refresh', {
        refreshToken: 'old-refresh-token'
      })
      expect(authStore.accessToken).toBe('new-access-token')
      expect(authStore.refreshToken).toBe('new-refresh-token')
      expect(localStorage.getItem('accessToken')).toBe('new-access-token')
      expect(localStorage.getItem('refreshToken')).toBe('new-refresh-token')
    })

    it('should logout on refresh failure', async () => {
      const authStore = useAuthStore()
      authStore.accessToken = 'old-token'
      authStore.refreshToken = 'old-refresh-token'
      authStore.user = { id: '1' }

      api.post.mockRejectedValueOnce(new Error('Invalid refresh token'))

      const result = await authStore.refresh()

      expect(result).toBe(false)
      expect(authStore.accessToken).toBeNull()
      expect(authStore.refreshToken).toBeNull()
      expect(authStore.user).toBeNull()
    })
  })
})
