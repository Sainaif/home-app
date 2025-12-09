<template>
  <UpdateBanner :show="hasNewVersion || isUpdating || hasPermissionUpdate" :isUpdating="isUpdating || isRefreshingPermissions" />
  <NotificationToast ref="notificationToast" />
  <NotificationHistory
    :isOpen="showNotificationHistory"
    @close="showNotificationHistory = false"
    @openPreferences="showNotificationPreferences = true" />
  <NotificationPreferences
    :isOpen="showNotificationPreferences"
    @close="showNotificationPreferences = false" />

  <div class="min-h-screen pb-20 md:pb-0">
    <!-- Desktop Navigation -->
    <nav v-if="authStore.isAuthenticated" class="glass sticky top-0 z-50 backdrop-blur-xl border-b border-gray-700/50 hidden md:block">
      <div class="max-w-7xl mx-auto px-4">
        <div class="flex items-center justify-between h-16">
          <router-link to="/" class="flex items-center space-x-2 hover:opacity-80 transition-opacity">
            <Home class="w-6 h-6 text-purple-400" />
            <span class="text-xl font-bold gradient-text">{{ appSettingsStore.appName }}</span>
          </router-link>

          <div class="flex items-center space-x-1">
            <router-link to="/" class="nav-link">
              <LayoutDashboard class="w-4 h-4" />
              {{ $t('nav.home') }}
            </router-link>
            <router-link to="/bills" class="nav-link">
              <Receipt class="w-4 h-4" />
              {{ $t('nav.bills') }}
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
          </div>

          <div class="flex items-center gap-2">
            <button
              @click="showNotificationHistory = true"
              class="btn btn-outline btn-sm relative flex items-center gap-2">
              <Bell class="w-4 h-4" />
              <span v-if="notificationStore.unreadCount > 0" class="absolute -top-1 -right-1 bg-purple-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
                {{ notificationStore.unreadCount > 9 ? '9+' : notificationStore.unreadCount }}
              </span>
            </button>
            <router-link to="/settings" class="btn btn-outline btn-sm flex items-center gap-2">
              <Settings class="w-4 h-4" />
              {{ $t('nav.settings') }}
            </router-link>
            <button @click="handleLogout" class="btn btn-secondary btn-sm flex items-center gap-2">
              <LogOut class="w-4 h-4" />
              {{ $t('nav.logout') }}
            </button>
          </div>
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
            <span class="text-lg font-bold gradient-text">{{ appSettingsStore.appName }}</span>
          </router-link>
          <div class="flex items-center gap-2">
            <button
              @click="showNotificationHistory = true"
              class="p-2.5 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 active:scale-95 transition-all relative">
              <Bell class="w-5 h-5 text-gray-300" />
              <span v-if="notificationStore.unreadCount > 0" class="absolute -top-0.5 -right-0.5 bg-purple-500 text-white text-xs rounded-full w-4 h-4 flex items-center justify-center text-[10px]">
                {{ notificationStore.unreadCount > 9 ? '9+' : notificationStore.unreadCount }}
              </span>
            </button>
            <router-link to="/settings" class="p-2.5 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 active:scale-95 transition-all">
              <Settings class="w-5 h-5 text-gray-300" />
            </router-link>
            <button @click="handleLogout" class="p-2.5 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 active:scale-95 transition-all">
              <LogOut class="w-5 h-5 text-gray-300" />
            </button>
          </div>
        </div>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-3 py-3 md:px-4 md:py-8">
      <router-view />
    </main>

    <!-- Footer -->
    <AppFooter v-if="authStore.isAuthenticated" class="mb-16 md:mb-0" />

    <!-- Mobile Bottom Navigation -->
    <nav
      v-if="authStore.isAuthenticated"
      class="fixed bottom-0 left-0 right-0 glass backdrop-blur-xl border-t border-gray-700/50 md:hidden z-50 safe-area-bottom"
      role="navigation"
      aria-label="Primary navigation">
      <div class="grid grid-cols-5 px-1 py-1.5">
        <router-link to="/" class="mobile-nav-link" :aria-label="$t('nav.dashboard')">
          <LayoutDashboard class="w-6 h-6" aria-hidden="true" />
          <span>{{ $t('nav.home') }}</span>
        </router-link>
        <router-link to="/bills" class="mobile-nav-link" :aria-label="$t('nav.bills')">
          <Receipt class="w-6 h-6" aria-hidden="true" />
          <span>{{ $t('nav.bills') }}</span>
        </router-link>
        <router-link to="/balance" class="mobile-nav-link" :aria-label="$t('nav.balance')">
          <Wallet class="w-6 h-6" aria-hidden="true" />
          <span>{{ $t('nav.balance') }}</span>
        </router-link>
        <router-link to="/chores" class="mobile-nav-link" :aria-label="$t('nav.chores')">
          <CheckSquare class="w-6 h-6" aria-hidden="true" />
          <span>{{ $t('nav.chores') }}</span>
        </router-link>
        <router-link to="/supplies" class="mobile-nav-link" :aria-label="$t('nav.supplies')">
          <ShoppingCart class="w-6 h-6" aria-hidden="true" />
          <span>{{ $t('nav.supplies') }}</span>
        </router-link>
      </div>
    </nav>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from './stores/auth'
import { useNotificationStore } from './stores/notification'
import { useAppSettingsStore } from './stores/appSettings'
import { useVersionCheck } from './composables/useVersionCheck'
import { useEventStream } from './composables/useEventStream'
import UpdateBanner from './components/UpdateBanner.vue'
import NotificationToast from './components/NotificationToast.vue'
import NotificationHistory from './components/NotificationHistory.vue'
import NotificationPreferences from './components/NotificationPreferences.vue'
import AppFooter from './components/AppFooter.vue'
import { Home, LayoutDashboard, Receipt, Wallet, CheckSquare, ShoppingCart, Settings, LogOut, Bell } from 'lucide-vue-next'

const router = useRouter()
const { t, locale } = useI18n()
const authStore = useAuthStore()
const notificationStore = useNotificationStore()
const appSettingsStore = useAppSettingsStore()
const { hasNewVersion, isUpdating } = useVersionCheck()
const eventStream = useEventStream()

const hasPermissionUpdate = ref(false)
const isRefreshingPermissions = ref(false)
const notificationToast = ref(null)
const showNotificationHistory = ref(false)
const showNotificationPreferences = ref(false)

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

// Auto-refresh permissions when window gains focus (no cooldown needed with SSE)
async function refreshPermissionsIfNeeded() {
  if (!authStore.isAuthenticated) return

  try {
    await authStore.loadUser()
    console.log('Permissions refreshed on focus')
  } catch (error) {
    console.error('Failed to refresh permissions:', error)
  }
}

// Handle permission update events
async function handlePermissionUpdate() {
  hasPermissionUpdate.value = true
  isRefreshingPermissions.value = true

  try {
    // Wait a moment to show the banner
    await new Promise(resolve => setTimeout(resolve, 1000))

    // Reload permissions
    await authStore.loadUser()

    // Wait another moment
    await new Promise(resolve => setTimeout(resolve, 500))

    // Reload the page to refresh all components with new permissions
    window.location.reload()
  } catch (error) {
    console.error('Failed to refresh permissions:', error)
    hasPermissionUpdate.value = false
    isRefreshingPermissions.value = false
  }
}

// Translate event types using i18n
function translateEvent(eventType, data) {
  const billType = translateBillType(data.type || data.billType)

  const translations = {
    'bill.created': {
      type: 'bill',
      title: t('events.billCreatedTitle'),
      message: t('events.billCreatedMessage', { createdBy: data.createdBy, type: billType, amount: data.amount }),
      resourceId: data.billId,
      resourceType: 'bill'
    },
    'bill.posted': {
      type: 'bill',
      title: t('events.billPostedTitle'),
      message: t('events.billPostedMessage', { type: billType, periodEnd: data.periodEnd, amount: formatAmount(data.amount) }),
      resourceId: data.billId,
      resourceType: 'bill'
    },
    'consumption.created': {
      type: 'bill',
      title: t('events.consumptionCreatedTitle'),
      message: t('events.consumptionCreatedMessage', { createdBy: data.createdBy, billType: translateBillType(data.billType), meterValue: data.meterValue }),
      resourceId: data.billId,
      resourceType: 'bill'
    },
    'chore.updated': {
      type: 'chore',
      title: data.action === 'created' ? t('events.choreCreatedTitle') : t('events.choreUpdatedTitle'),
      message: data.name,
      resourceId: data.choreId,
      resourceType: 'chore'
    },
    'chore.assigned': {
      type: 'chore',
      title: t('events.choreAssignedTitle'),
      message: t('events.choreAssignedMessage', { choreName: data.choreName, dueDate: formatDueDate(data.dueDate) }),
      resourceId: data.choreId,
      resourceType: 'chore'
    },
    'supply.item.added': {
      type: 'supply',
      title: t('events.supplyItemAddedTitle'),
      message: t('events.supplyItemAddedMessage', { addedBy: data.addedBy, name: data.name, category: data.category }),
      resourceId: data.itemId,
      resourceType: 'supply'
    },
    'loan.created': {
      type: 'loan',
      title: t('events.loanCreatedTitle'),
      message: data.message || t('events.loanCreatedMessage'),
      resourceId: data.loanId,
      resourceType: 'loan'
    },
    'loan.payment.created': {
      type: 'loan',
      title: t('events.loanPaymentCreatedTitle'),
      message: data.message || t('events.loanPaymentCreatedMessage'),
      resourceId: data.loanId,
      resourceType: 'loan'
    },
    'loan.deleted': {
      type: 'loan',
      title: t('events.loanDeletedTitle'),
      message: data.message || t('events.loanDeletedMessage'),
      resourceType: 'loan'
    },
    'supply.item.bought': {
      type: 'supply',
      title: t('events.supplyItemBoughtTitle'),
      message: t('events.supplyItemBoughtMessage', { boughtBy: data.boughtBy, name: data.name }),
      resourceId: data.itemId,
      resourceType: 'supply'
    },
    'supply.budget.contributed': {
      type: 'supply',
      title: t('events.supplyBudgetContributedTitle'),
      message: t('events.supplyBudgetContributedMessage', { contributedBy: data.contributedBy, amount: data.amount }),
      resourceType: 'supply'
    },
    'supply.budget.low': {
      type: 'supply',
      title: t('events.supplyBudgetLowTitle'),
      message: t('events.supplyBudgetLowMessage', { currentBudget: data.currentBudget }),
      resourceType: 'supply'
    },
    'payment.created': {
      type: 'bill',
      title: t('events.paymentCreatedTitle'),
      message: data.message || t('events.paymentCreatedMessage'),
      resourceId: data.billId,
      resourceType: 'bill'
    },
    'balance.updated': {
      type: 'loan',
      title: t('events.balanceUpdatedTitle'),
      message: data.message || t('events.balanceUpdatedMessage'),
      resourceType: 'loan'
    },
    'permissions.updated': {
      type: 'system',
      title: t('events.permissionsTitle'),
      message: data.message || t('events.permissionsUpdatedMessage'),
      skipNotification: true // Don't show this as a notification
    }
  }

  return translations[eventType] || {
    type: 'info',
    title: t('events.unknownTitle'),
    message: eventType
  }
}

function translateBillType(type) {
  if (!type) return ''
  const key = type === 'inne' ? 'other' : type
  return t(`billTypes.${key}`, type)
}

function formatDueDate(dateString) {
  if (!dateString) return ''
  const date = new Date(dateString)
  const localeMap = { 'pl': 'pl-PL', 'en': 'en-US' }
  const dateLocale = localeMap[locale.value] || 'en-US'
  return date.toLocaleDateString(dateLocale, { day: 'numeric', month: 'short', year: 'numeric' })
}

function formatAmount(amountString) {
  if (!amountString) return '0.00'
  // Parse Decimal128 JSON format or plain number
  const cleaned = amountString.replace(/["{}$numberDecimal:]/g, '')
  const amount = parseFloat(cleaned)
  return amount.toFixed(2)
}

// Request browser notification permission
async function requestNotificationPermission() {
  if (!('Notification' in window)) {
    console.log('This browser does not support notifications')
    return false
  }

  if (Notification.permission === 'granted') {
    return true
  }

  if (Notification.permission !== 'denied') {
    const permission = await Notification.requestPermission()
    return permission === 'granted'
  }

  return false
}

// Show browser notification
function showBrowserNotification(notification) {
  if (Notification.permission !== 'granted') return

  const options = {
    body: notification.message,
    icon: '/icon-192.png',
    badge: '/icon-192.png',
    tag: notification.resourceId || 'general',
    requireInteraction: false,
    silent: false
  }

  const notif = new Notification(notification.title, options)

  // Handle click - focus window and navigate
  notif.onclick = () => {
    window.focus()
    notif.close()

    // Navigate to the resource if possible
    if (notification.resourceId && notification.resourceType) {
      const routes = {
        bill: `/bills?id=${notification.resourceId}`,
        chore: `/chores?id=${notification.resourceId}`,
        supply: `/supplies?id=${notification.resourceId}`,
        loan: `/balance?loanId=${notification.resourceId}`
      }
      const route = routes[notification.resourceType]
      if (route) {
        router.push(route)
      }
    }
  }
}

// Handle all SSE events
function handleSSEEvent(event) {
  if (!notificationToast.value) return

  const notification = translateEvent(event.type, event.data || {})

  // Skip notifications marked as skipNotification
  if (notification.skipNotification) {
    return
  }

  // Check user preferences - should we show this type?
  if (!notificationStore.shouldShowNotification(notification.type)) {
    return
  }

  // Add to notification history
  notificationStore.addNotification(notification)

  // Show in-app toast notification
  notificationToast.value.show(notification)

  // Show browser notification if permission granted
  showBrowserNotification(notification)
}

onMounted(async () => {
  // Fetch app settings (public, works even when not logged in)
  await appSettingsStore.fetchSettings()

  // Refresh permissions when window gains focus
  window.addEventListener('focus', refreshPermissionsIfNeeded)

  // Also refresh on initial mount if authenticated
  if (authStore.isAuthenticated) {
    refreshPermissionsIfNeeded()

    // Fetch auth config (needed for VAPID key on page refresh)
    await authStore.fetchAuthConfig()

    // Fetch notification preferences (only when authenticated)
    await notificationStore.fetchPreferences()

    // Request browser notification permission
    await requestNotificationPermission()
    await notificationStore.subscribe()

    // Connect to event stream and listen for all events
    eventStream.connect()

    // Permission updates trigger page reload
    eventStream.on('permissions.updated', handlePermissionUpdate)

    // Other events show toast notifications
    eventStream.on('notification.created', handleSSEEvent)
    eventStream.on('bill.created', handleSSEEvent)
    eventStream.on('bill.posted', handleSSEEvent)
    eventStream.on('consumption.created', handleSSEEvent)
    eventStream.on('chore.updated', handleSSEEvent)
    eventStream.on('chore.assigned', handleSSEEvent)
    eventStream.on('supply.item.added', handleSSEEvent)
    eventStream.on('supply.item.bought', handleSSEEvent)
    eventStream.on('supply.budget.contributed', handleSSEEvent)
    eventStream.on('supply.budget.low', handleSSEEvent)
    eventStream.on('loan.created', handleSSEEvent)
    eventStream.on('loan.payment.created', handleSSEEvent)
    eventStream.on('loan.deleted', handleSSEEvent)
    eventStream.on('payment.created', handleSSEEvent)
    eventStream.on('balance.updated', handleSSEEvent)
  }
})

onUnmounted(() => {
  window.removeEventListener('focus', refreshPermissionsIfNeeded)
  eventStream.disconnect()
})
</script>