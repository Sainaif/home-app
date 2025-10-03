<template>
  <UpdateBanner :show="hasNewVersion || isUpdating" :isUpdating="isUpdating" />

  <div class="min-h-screen pb-20 md:pb-0">
    <!-- Desktop Navigation -->
    <nav v-if="authStore.isAuthenticated" class="glass sticky top-0 z-50 backdrop-blur-xl border-b border-gray-700/50 hidden md:block">
      <div class="max-w-7xl mx-auto px-4">
        <div class="flex items-center justify-between h-16">
          <router-link to="/" class="flex items-center space-x-2 hover:opacity-80 transition-opacity">
            <Home class="w-6 h-6 text-purple-400" />
            <span class="text-xl font-bold gradient-text">Holy Home</span>
          </router-link>

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
            <router-link to="/supplies" class="nav-link">
              <ShoppingCart class="w-4 h-4" />
              {{ $t('nav.supplies') }}
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

    <!-- Mobile Header -->
    <header v-if="authStore.isAuthenticated" class="glass sticky top-0 z-50 backdrop-blur-xl border-b border-gray-700/50 md:hidden">
      <div class="px-4 py-4">
        <div class="flex items-center justify-between">
          <router-link to="/" class="flex items-center gap-2.5 active:opacity-70 transition-opacity">
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-purple-600 to-pink-600 flex items-center justify-center shadow-lg">
              <Home class="w-5 h-5 text-white" />
            </div>
            <span class="text-lg font-bold gradient-text">Holy Home</span>
          </router-link>
          <button @click="handleLogout" class="p-2.5 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 active:scale-95 transition-all">
            <LogOut class="w-5 h-5 text-gray-300" />
          </button>
        </div>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-3 py-3 md:px-4 md:py-8">
      <router-view />
    </main>

    <!-- Mobile Bottom Navigation -->
    <nav
      v-if="authStore.isAuthenticated"
      class="fixed bottom-0 left-0 right-0 glass backdrop-blur-xl border-t border-gray-700/50 md:hidden z-50 safe-area-bottom"
      role="navigation"
      aria-label="Primary navigation">
      <div class="grid grid-cols-5 px-1 py-1.5">
        <router-link to="/" class="mobile-nav-link" aria-label="Dashboard">
          <LayoutDashboard class="w-6 h-6" aria-hidden="true" />
          <span>Home</span>
        </router-link>
        <router-link to="/bills" class="mobile-nav-link" aria-label="Bills">
          <Receipt class="w-6 h-6" aria-hidden="true" />
          <span>Rachunki</span>
        </router-link>
        <router-link to="/readings" class="mobile-nav-link" aria-label="Meter readings">
          <Gauge class="w-6 h-6" aria-hidden="true" />
          <span>Odczyty</span>
        </router-link>
        <router-link to="/balance" class="mobile-nav-link" aria-label="Balance overview">
          <Wallet class="w-6 h-6" aria-hidden="true" />
          <span>Bilans</span>
        </router-link>
        <router-link to="/settings" class="mobile-nav-link" aria-label="Settings and more">
          <Settings class="w-6 h-6" aria-hidden="true" />
          <span>Więcej</span>
        </router-link>
      </div>
    </nav>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useVersionCheck } from './composables/useVersionCheck'
import UpdateBanner from './components/UpdateBanner.vue'
import { Home, LayoutDashboard, Receipt, Gauge, Wallet, CheckSquare, ShoppingCart, Settings, LogOut } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const { hasNewVersion, isUpdating } = useVersionCheck()

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>