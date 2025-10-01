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
    path: '/readings',
    name: 'Readings',
    component: () => import('../views/Readings.vue'),
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
    path: '/predictions',
    name: 'Predictions',
    component: () => import('../views/Predictions.vue'),
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

router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()

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