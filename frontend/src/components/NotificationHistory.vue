<template>
  <Teleport to="body">
    <Transition name="slide">
      <div v-if="isOpen" class="fixed inset-0 z-50 flex justify-end" @click.self="close">
        <div class="w-full max-w-md bg-gray-800 shadow-2xl flex flex-col h-full">
          <!-- Header -->
          <div class="p-4 border-b border-gray-700 flex items-center justify-between">
            <h2 class="text-xl font-bold text-white">Powiadomienia</h2>
            <div class="flex items-center gap-2">
              <button
                @click="$emit('openPreferences')"
                class="p-2 hover:bg-gray-700 rounded-lg transition-colors"
                title="Ustawienia">
                <Settings class="w-5 h-5 text-gray-400" />
              </button>
              <button
                @click="close"
                class="p-2 hover:bg-gray-700 rounded-lg transition-colors">
                <X class="w-5 h-5 text-gray-400" />
              </button>
            </div>
          </div>

          <!-- Actions Bar -->
          <div class="p-4 border-b border-gray-700 space-y-3">
            <div class="flex items-center gap-2">
              <button
                @click="notificationStore.markAllAsRead()"
                :disabled="notificationStore.unreadCount === 0"
                class="btn btn-secondary btn-sm flex-1 disabled:opacity-50 disabled:cursor-not-allowed">
                <CheckCheck class="w-4 h-4" />
                Oznacz wszystkie jako przeczytane
              </button>
            </div>

            <div class="flex items-center gap-2">
              <select
                v-model="filterType"
                class="flex-1 bg-gray-700 text-white rounded-lg px-3 py-2 text-sm border border-gray-600">
                <option value="all">Wszystkie</option>
                <option value="bill">Rachunki</option>
                <option value="chore">Obowiązki</option>
                <option value="supply">Zaopatrzenie</option>
                <option value="loan">Pożyczki</option>
                <option value="permission">Uprawnienia</option>
              </select>

            </div>
          </div>

          <!-- Notifications List -->
          <div class="flex-1 overflow-y-auto">
            <div v-if="filteredNotifications.length === 0" class="p-8 text-center text-gray-400">
              <Bell class="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>Brak powiadomień</p>
            </div>

            <div
              v-for="notification in filteredNotifications"
              :key="notification.id"
              :class="[
                'p-4 border-b border-gray-700 cursor-pointer transition-colors hover:bg-gray-700/50',
                !notification.read && 'bg-gray-700/30'
              ]"
              @click="handleNotificationClick(notification)">
              <div class="flex items-start gap-3">
                <!-- Read/Unread Indicator -->
                <div class="mt-1">
                  <div
                    v-if="!notification.read"
                    :class="[
                      'w-2 h-2 rounded-full',
                      getIndicatorColor(notification.type)
                    ]" />
                  <div v-else class="w-2 h-2 rounded-full bg-gray-600" />
                </div>

                <!-- Icon -->
                <component
                  :is="getIcon(notification.type)"
                  :class="[
                    'w-5 h-5 flex-shrink-0 mt-0.5',
                    getIconColor(notification.type)
                  ]" />

                <!-- Content -->
                <div class="flex-1 min-w-0">
                  <p :class="['text-sm', !notification.read && 'font-semibold text-white']">
                    {{ notification.title }}
                  </p>
                  <p v-if="notification.message" class="text-sm text-gray-400 mt-1">
                    {{ notification.message }}
                  </p>
                  <p class="text-xs text-gray-500 mt-2">
                    {{ formatTimestamp(notification.timestamp) }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useNotificationStore } from '../stores/notification'
import { X, Bell, CheckCheck, Trash2, Settings, FileText, CheckSquare, ShoppingCart, DollarSign, UserPlus } from 'lucide-vue-next'

const props = defineProps({
  isOpen: Boolean
})

const emit = defineEmits(['close', 'openPreferences', 'navigate'])

const router = useRouter()
const notificationStore = useNotificationStore()

const filterType = ref('all')

const filteredNotifications = computed(() => {
  if (filterType.value === 'all') {
    return notificationStore.history
  }
  return notificationStore.history.filter(n => n.type === filterType.value)
})

watch(() => props.isOpen, (isOpen) => {
  if (isOpen) {
    notificationStore.fetchNotifications()
  }
})

function close() {
  emit('close')
}

function handleNotificationClick(notification) {
  // Mark as read
  notificationStore.markAsRead(notification.id)

  // Navigate if has resource info
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
      close()
    }
  }
}

function formatTimestamp(timestamp) {
  const now = new Date()
  const then = new Date(timestamp)
  const diffInSeconds = Math.floor((now - then) / 1000)

  if (diffInSeconds < 60) return 'Przed chwilą'
  if (diffInSeconds < 3600) {
    const mins = Math.floor(diffInSeconds / 60)
    return `${mins} ${mins === 1 ? 'minutę' : mins < 5 ? 'minuty' : 'minut'} temu`
  }
  if (diffInSeconds < 86400) {
    const hours = Math.floor(diffInSeconds / 3600)
    return `${hours} ${hours === 1 ? 'godzinę' : hours < 5 ? 'godziny' : 'godzin'} temu`
  }

  const days = Math.floor(diffInSeconds / 86400)
  if (days < 7) {
    return `${days} ${days === 1 ? 'dzień' : 'dni'} temu`
  }

  return then.toLocaleDateString('pl-PL')
}

function getIcon(type) {
  const icons = {
    bill: FileText,
    chore: CheckSquare,
    supply: ShoppingCart,
    loan: DollarSign,
    permission: UserPlus,
    info: Bell
  }
  return icons[type] || icons.info
}

function getIndicatorColor(type) {
  const colors = {
    bill: 'bg-blue-400',
    chore: 'bg-purple-400',
    supply: 'bg-green-400',
    loan: 'bg-yellow-400',
    permission: 'bg-pink-400',
    info: 'bg-gray-400'
  }
  return colors[type] || colors.info
}

function getIconColor(type) {
  const colors = {
    bill: 'text-blue-400',
    chore: 'text-purple-400',
    supply: 'text-green-400',
    loan: 'text-yellow-400',
    permission: 'text-pink-400',
    info: 'text-gray-400'
  }
  return colors[type] || colors.info
}
</script>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from {
  transform: translateX(100%);
}

.slide-leave-to {
  transform: translateX(100%);
}

.slide-enter-from ~ *,
.slide-leave-to ~ * {
  opacity: 0;
}
</style>
