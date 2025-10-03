import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/dashboard/:userId',
    name: 'UserDashboard',
    component: () => import('../views/Dashboard.vue'),
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/bills',
    name: 'Bills',
    component: () => import('../views/Bills.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/bills/:id',
    name: 'BillDetail',
    component: () => import('../views/BillDetail.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/balance',
    name: 'Balance',
    component: () => import('../views/Balance.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/chores',
    name: 'Chores',
    component: () => import('../views/Chores.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/supplies',
    name: 'Supplies',
    component: () => import('../views/Supplies.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('../views/Settings.vue'),
    meta: { requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // If user has tokens but is navigating to a protected route, validate them
  if (to.meta.requiresAuth && authStore.isAuthenticated) {
    // Try to refresh the token to ensure it's valid
    // This will use the refresh token if the access token is expired
    try {
      // If access token is still valid, this will just verify it works
      // If expired, the API interceptor will refresh it automatically
      // We just need to trigger any authenticated request or explicitly refresh
      await authStore.validateSession()
    } catch (error) {
      // If validation/refresh fails, clear tokens and redirect to login
      authStore.logout()
      next('/login')
      return
    }
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/')
  } else if (to.meta.requiresAdmin && !authStore.isAdmin) {
    next('/')
  } else {
    next()
  }
})

export default router