<template>
  <div class="fixed top-4 right-4 z-50 space-y-2 max-w-sm">
    <TransitionGroup name="toast">
      <div
        v-for="notification in notifications"
        :key="notification.id"
        :class="[
          'p-4 rounded-lg shadow-lg cursor-pointer transition-all',
          'border-l-4 backdrop-blur-sm',
          getNotificationClass(notification.type)
        ]"
        @click="handleClick(notification)">
        <div class="flex items-start gap-3">
          <component :is="getIcon(notification.type)" class="w-5 h-5 flex-shrink-0 mt-0.5" />
          <div class="flex-1 min-w-0">
            <p class="font-semibold text-sm">{{ notification.title }}</p>
            <p v-if="notification.message" class="text-sm opacity-90 mt-1">{{ notification.message }}</p>
          </div>
          <button
            @click.stop="dismiss(notification.id)"
            class="text-white/60 hover:text-white/90 transition-colors">
            <X class="w-4 h-4" />
          </button>
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { X, Bell, FileText, CheckSquare, ShoppingCart, DollarSign, UserPlus } from 'lucide-vue-next'

const router = useRouter()
const notifications = ref([])

function show(notification) {
  const id = Date.now() + Math.random()
  const newNotification = {
    id,
    ...notification
  }

  notifications.value.unshift(newNotification)

  // Auto-dismiss after 5 seconds
  setTimeout(() => {
    dismiss(id)
  }, 5000)
}

function dismiss(id) {
  const index = notifications.value.findIndex(n => n.id === id)
  if (index !== -1) {
    notifications.value.splice(index, 1)
  }
}

function handleClick(notification) {
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
      dismiss(notification.id)
      return
    }
  }

  // Otherwise just dismiss
  dismiss(notification.id)
}

function getNotificationClass(type) {
  const classes = {
    bill: 'bg-blue-600/90 border-blue-400 text-white',
    chore: 'bg-purple-600/90 border-purple-400 text-white',
    supply: 'bg-green-600/90 border-green-400 text-white',
    loan: 'bg-yellow-600/90 border-yellow-400 text-white',
    permission: 'bg-pink-600/90 border-pink-400 text-white',
    info: 'bg-gray-600/90 border-gray-400 text-white'
  }
  return classes[type] || classes.info
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

defineExpose({ show, dismiss })
</script>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(2rem);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(2rem) scale(0.95);
}
</style>
