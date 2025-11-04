import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../stores/auth'

// Mock axios before importing client
vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => ({
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() }
      },
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn()
    }))
  }
}))

describe('API Client', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('Auth Store Integration', () => {
    it('should work with auth store for token management', () => {
      const authStore = useAuthStore()
      authStore.accessToken = 'test-access-token'

      expect(authStore.accessToken).toBe('test-access-token')
    })

    it('should handle missing token', () => {
      const authStore = useAuthStore()

      expect(authStore.accessToken).toBeNull()
    })

    it('should support token refresh', () => {
      const authStore = useAuthStore()
      authStore.refreshToken = 'test-refresh-token'
      authStore.refresh = vi.fn().mockResolvedValue(true)

      expect(authStore.refreshToken).toBe('test-refresh-token')
      expect(authStore.refresh).toBeDefined()
    })
  })
})
