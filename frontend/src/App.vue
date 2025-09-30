<template>
  <UpdateBanner :show="hasNewVersion || isUpdating" :isUpdating="isUpdating" />

  <div class="min-h-screen">
    <nav v-if="authStore.isAuthenticated" class="glass sticky top-0 z-50 backdrop-blur-xl border-b border-gray-700/50">
      <div class="max-w-7xl mx-auto px-4">
        <div class="flex items-center justify-between h-16">
          <div class="flex items-center space-x-2">
            <Home class="w-6 h-6 text-purple-400" />
            <span class="text-xl font-bold gradient-text">Holy Home</span>
          </div>

          <div class="flex items-center space-x-1">
            <router-link to="/" class="nav-link">
              <LayoutDashboard class="w-4 h-4" />
              {{ $t('nav.dashboard') }}
            </router-link>
            <router-link to="/bills" class="nav-link">
              <Receipt class="w-4 h-4" />
              {{ $t('nav.bills') }}
            </router-link>
            <router-link to="/readings" class="nav-link">
              <Gauge class="w-4 h-4" />
              {{ $t('nav.readings') }}
            </router-link>
            <router-link to="/balance" class="nav-link">
              <Wallet class="w-4 h-4" />
              {{ $t('nav.balance') }}
            </router-link>
            <router-link to="/chores" class="nav-link">
              <CheckSquare class="w-4 h-4" />
              {{ $t('nav.chores') }}
            </router-link>
            <router-link to="/predictions" class="nav-link">
              <TrendingUp class="w-4 h-4" />
              {{ $t('nav.predictions') }}
            </router-link>
            <router-link to="/settings" class="nav-link">
              <Settings class="w-4 h-4" />
              {{ $t('nav.settings') }}
            </router-link>
          </div>

          <button @click="handleLogout" class="btn btn-secondary btn-sm flex items-center gap-2">
            <LogOut class="w-4 h-4" />
            {{ $t('nav.logout') }}
          </button>
        </div>
      </div>
    </nav>

    <main class="max-w-7xl mx-auto px-4 py-8">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useVersionCheck } from './composables/useVersionCheck'
import { watch } from 'vue'
import UpdateBanner from './components/UpdateBanner.vue'
import { Home, LayoutDashboard, Receipt, Gauge, Wallet, CheckSquare, TrendingUp, Settings, LogOut } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const { hasNewVersion, isUpdating, reloadApp } = useVersionCheck()

// Auto-reload when new version detected
watch(hasNewVersion, (newVal) => {
  if (newVal) {
    reloadApp()
  }
})

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>