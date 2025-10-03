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
            <span class="text-xl font-bold gradient-text">Holy Home</span>
          </router-link>

          <div class="flex items-center space-x-1">
            <router-link to="/" class="nav-link">
              <LayoutDashboard class="w-4 h-4" />
              Home
            </router-link>
            <router-link to="/bills" class="nav-link">
              <Receipt class="w-4 h-4" />
              Rachunki
            </router-link>
            <router-link to="/balance" class="nav-link">
              <Wallet class="w-4 h-4" />
              Bilans
            </router-link>
            <router-link to="/chores" class="nav-link">
              <CheckSquare class="w-4 h-4" />
              Obowiązki
            </router-link>
            <router-link to="/supplies" class="nav-link">
              <ShoppingCart class="w-4 h-4" />
              Zakupy
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
            <span class="text-lg font-bold gradient-text">Holy Home</span>
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
        <router-link to="/balance" class="mobile-nav-link" aria-label="Balance overview">
          <Wallet class="w-6 h-6" aria-hidden="true" />
          <span>Bilans</span>
        </router-link>
        <router-link to="/chores" class="mobile-nav-link" aria-label="Chores">
          <CheckSquare class="w-6 h-6" aria-hidden="true" />
          <span>Obowiązki</span>
        </router-link>
        <router-link to="/supplies" class="mobile-nav-link" aria-label="Supplies">
          <ShoppingCart class="w-6 h-6" aria-hidden="true" />
          <span>Zakupy</span>
        </router-link>
      </div>
    </nav>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useNotificationStore } from './stores/notification'
import { useVersionCheck } from './composables/useVersionCheck'
import { useEventStream } from './composables/useEventStream'
import UpdateBanner from './components/UpdateBanner.vue'
import NotificationToast from './components/NotificationToast.vue'
import NotificationHistory from './components/NotificationHistory.vue'
import NotificationPreferences from './components/NotificationPreferences.vue'
import { Home, LayoutDashboard, Receipt, Wallet, CheckSquare, ShoppingCart, Settings, LogOut, Bell } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const notificationStore = useNotificationStore()
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

// Translate event types to Polish messages
function translateEvent(eventType, data) {
  const translations = {
    'bill.created': {
      type: 'bill',
      title: 'Nowy rachunek',
      message: `${data.createdBy} dodał rachunek: ${translateBillType(data.type)} - ${data.amount} PLN`,
      resourceId: data.billId,
      resourceType: 'bill'
    },
    'bill.posted': {
      type: 'bill',
      title: 'Opublikowano rachunek',
      message: `${translateBillType(data.type)} (${data.periodEnd}) - ${formatAmount(data.amount)} PLN - gotowy do rozliczenia`,
      resourceId: data.billId,
      resourceType: 'bill'
    },
    'consumption.created': {
      type: 'bill',
      title: 'Nowy odczyt',
      message: `${data.createdBy} dodał odczyt dla ${translateBillType(data.billType)}: ${data.meterValue}`,
      resourceId: data.billId,
      resourceType: 'bill'
    },
    'chore.updated': {
      type: 'chore',
      title: data.action === 'created' ? 'Nowy obowiązek' : 'Zaktualizowano obowiązek',
      message: data.name,
      resourceId: data.choreId,
      resourceType: 'chore'
    },
    'chore.assigned': {
      type: 'chore',
      title: 'Przydzielono obowiązek',
      message: `Masz nowy obowiązek: ${data.choreName}. Termin: ${formatDueDate(data.dueDate)}`,
      resourceId: data.choreId,
      resourceType: 'chore'
    },
    'supply.item.added': {
      type: 'supply',
      title: 'Nowa pozycja',
      message: `${data.addedBy} dodał: ${data.name} (${data.category})`,
      resourceId: data.itemId,
      resourceType: 'supply'
    },
    'loan.created': {
      type: 'loan',
      title: 'Nowa pożyczka',
      message: data.message || 'Dodano nową pożyczkę',
      resourceId: data.loanId,
      resourceType: 'loan'
    },
    'loan.payment.created': {
      type: 'loan',
      title: 'Spłata pożyczki',
      message: data.message || 'Zarejestrowano spłatę',
      resourceId: data.loanId,
      resourceType: 'loan'
    },
    'loan.deleted': {
      type: 'loan',
      title: 'Usunięto pożyczkę',
      message: data.message || 'Pożyczka została usunięta',
      resourceType: 'loan'
    },
    'supply.item.bought': {
      type: 'supply',
      title: 'Zakupiono produkt',
      message: `${data.boughtBy} kupił: ${data.name}`,
      resourceId: data.itemId,
      resourceType: 'supply'
    },
    'supply.budget.contributed': {
      type: 'supply',
      title: 'Wpłata na budżet',
      message: `${data.contributedBy} wpłacił ${data.amount} PLN`,
      resourceType: 'supply'
    },
    'supply.budget.low': {
      type: 'supply',
      title: 'Niski budżet',
      message: `Budżet wynosi ${data.currentBudget} PLN`,
      resourceType: 'supply'
    },
    'payment.created': {
      type: 'bill',
      title: 'Nowa wpłata',
      message: data.message || 'Zarejestrowano wpłatę',
      resourceId: data.billId,
      resourceType: 'bill'
    },
    'balance.updated': {
      type: 'loan',
      title: 'Zaktualizowano bilans',
      message: data.message || 'Bilans został zaktualizowany',
      resourceType: 'loan'
    },
    'permissions.updated': {
      type: 'system',
      title: 'System',
      message: data.message || 'Zaktualizowano uprawnienia',
      skipNotification: true // Don't show this as a notification
    }
  }

  return translations[eventType] || {
    type: 'info',
    title: 'Nowe zdarzenie',
    message: eventType
  }
}

function translateBillType(type) {
  const types = {
    'electricity': 'Prąd',
    'gas': 'Gaz',
    'internet': 'Internet',
    'water': 'Woda',
    'inne': 'Inne'
  }
  return types[type] || type
}

function formatDueDate(dateString) {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleDateString('pl-PL', { day: 'numeric', month: 'short', year: 'numeric' })
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
  // Refresh permissions when window gains focus
  window.addEventListener('focus', refreshPermissionsIfNeeded)

  // Also refresh on initial mount if authenticated
  if (authStore.isAuthenticated) {
    refreshPermissionsIfNeeded()

    // Request browser notification permission
    await requestNotificationPermission()

    // Connect to event stream and listen for all events
    eventStream.connect()

    // Permission updates trigger page reload
    eventStream.on('permissions.updated', handlePermissionUpdate)

    // Other events show toast notifications
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