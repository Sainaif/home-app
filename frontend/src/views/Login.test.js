import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createI18n } from 'vue-i18n'
import Login from './Login.vue'
import { useAuthStore } from '../stores/auth'

// Mock the passkey composable
vi.mock('../composables/usePasskey', () => ({
  usePasskey: () => ({
    checkSupport: vi.fn(() => Promise.resolve({ supported: false })),
    loginWithConditionalUI: vi.fn()
  })
}))

// Create mock i18n
const i18n = createI18n({
  legacy: false,
  locale: 'pl',
  messages: {
    pl: {
      auth: {
        email: 'Email',
        password: 'Hasło',
        loginButton: 'Zaloguj'
      },
      common: {
        loading: 'Ładowanie'
      }
    }
  }
})

// Create mock router
const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/', component: { template: '<div>Dashboard</div>' } },
    { path: '/login', component: Login }
  ]
})

describe('Login Component', () => {
  let wrapper
  let authStore

  beforeEach(() => {
    setActivePinia(createPinia())
    authStore = useAuthStore()
    vi.clearAllMocks()
    localStorage.clear()
  })

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
    }
  })

  it('should render login form', () => {
    wrapper = mount(Login, {
      global: {
        plugins: [router, i18n],
        stubs: {
          'lucide-vue-next': true
        }
      }
    })

    expect(wrapper.find('input[type="email"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
  })

  it('should bind email and password inputs correctly', async () => {
    wrapper = mount(Login, {
      global: {
        plugins: [router, i18n],
        stubs: {
          'lucide-vue-next': true
        }
      }
    })

    const emailInput = wrapper.find('input[type="email"]')
    const passwordInput = wrapper.find('input[type="password"]')

    await emailInput.setValue('test@example.com')
    await passwordInput.setValue('password123')

    expect(emailInput.element.value).toBe('test@example.com')
    expect(passwordInput.element.value).toBe('password123')
  })

  it('should call authStore.login on form submit', async () => {
    wrapper = mount(Login, {
      global: {
        plugins: [router, i18n],
        stubs: {
          'lucide-vue-next': true
        }
      }
    })

    authStore.login = vi.fn().mockResolvedValue({})

    const emailInput = wrapper.find('input[type="email"]')
    const passwordInput = wrapper.find('input[type="password"]')
    const form = wrapper.find('form')

    await emailInput.setValue('test@example.com')
    await passwordInput.setValue('password123')
    await form.trigger('submit.prevent')

    expect(authStore.login).toHaveBeenCalledWith('test@example.com', 'password123')
  })

  it('should navigate to home on successful login', async () => {
    wrapper = mount(Login, {
      global: {
        plugins: [router, i18n],
        stubs: {
          'lucide-vue-next': true
        }
      }
    })

    authStore.login = vi.fn().mockResolvedValue({})
    const pushSpy = vi.spyOn(router, 'push')

    const emailInput = wrapper.find('input[type="email"]')
    const passwordInput = wrapper.find('input[type="password"]')
    const form = wrapper.find('form')

    await emailInput.setValue('test@example.com')
    await passwordInput.setValue('password123')
    await form.trigger('submit.prevent')

    // Wait for async operations
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    expect(pushSpy).toHaveBeenCalledWith('/')
  })

  it('should display error message on login failure', async () => {
    wrapper = mount(Login, {
      global: {
        plugins: [router, i18n],
        stubs: {
          'lucide-vue-next': true
        }
      }
    })

    const errorMessage = 'Invalid credentials'
    authStore.login = vi.fn().mockRejectedValue({
      response: { data: { error: errorMessage } }
    })

    const emailInput = wrapper.find('input[type="email"]')
    const passwordInput = wrapper.find('input[type="password"]')
    const form = wrapper.find('form')

    await emailInput.setValue('test@example.com')
    await passwordInput.setValue('wrongpassword')
    await form.trigger('submit.prevent')

    // Wait for async operations
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    expect(wrapper.text()).toContain(errorMessage)
  })

  it('should disable submit button while loading', async () => {
    wrapper = mount(Login, {
      global: {
        plugins: [router, i18n],
        stubs: {
          'lucide-vue-next': true
        }
      }
    })

    // Mock login to return a promise that doesn't resolve immediately
    authStore.login = vi.fn(() => new Promise(() => {}))

    const submitButton = wrapper.find('button[type="submit"]')
    expect(submitButton.attributes('disabled')).toBeUndefined()

    const form = wrapper.find('form')
    await form.trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(submitButton.attributes('disabled')).toBeDefined()
  })
})
