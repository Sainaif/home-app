import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useNotificationStore = defineStore('notification', () => {
  // Load from localStorage
  const history = ref(JSON.parse(localStorage.getItem('notificationHistory') || '[]'))
  const preferences = ref(JSON.parse(localStorage.getItem('notificationPreferences') || JSON.stringify({
    bill: true,
    chore: true,
    supply: true,
    loan: true
  })))

  // Computed
  const unreadCount = computed(() => history.value.filter(n => !n.read).length)

  const unreadNotifications = computed(() => history.value.filter(n => !n.read))

  const readNotifications = computed(() => history.value.filter(n => n.read))

  // Actions
  function addNotification(notification) {
    const newNotification = {
      id: Date.now() + Math.random(),
      timestamp: new Date().toISOString(),
      read: false,
      ...notification
    }

    // Add to beginning of array
    history.value.unshift(newNotification)

    // Limit to 50 notifications
    if (history.value.length > 50) {
      history.value = history.value.slice(0, 50)
    }

    // Cleanup old notifications (older than 7 days)
    const sevenDaysAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
    history.value = history.value.filter(n =>
      new Date(n.timestamp) > sevenDaysAgo
    )

    saveHistory()
    return newNotification
  }

  function markAsRead(id) {
    const notification = history.value.find(n => n.id === id)
    if (notification) {
      notification.read = true
      saveHistory()
    }
  }

  function markAllAsRead() {
    history.value.forEach(n => n.read = true)
    saveHistory()
  }

  function clearHistory() {
    history.value = []
    saveHistory()
  }

  function togglePreference(type) {
    preferences.value[type] = !preferences.value[type]
    savePreferences()
  }

  function setPreference(type, value) {
    preferences.value[type] = value
    savePreferences()
  }

  function enableAll() {
    Object.keys(preferences.value).forEach(key => {
      preferences.value[key] = true
    })
    savePreferences()
  }

  function disableAll() {
    Object.keys(preferences.value).forEach(key => {
      preferences.value[key] = false
    })
    savePreferences()
  }

  function shouldShowNotification(type) {
    return preferences.value[type] !== false
  }

  function saveHistory() {
    localStorage.setItem('notificationHistory', JSON.stringify(history.value))
  }

  function savePreferences() {
    localStorage.setItem('notificationPreferences', JSON.stringify(preferences.value))
  }

  function getNotificationById(id) {
    return history.value.find(n => n.id === id)
  }

  function removeNotification(id) {
    const index = history.value.findIndex(n => n.id === id)
    if (index !== -1) {
      history.value.splice(index, 1)
      saveHistory()
    }
  }

  return {
    // State
    history,
    preferences,

    // Computed
    unreadCount,
    unreadNotifications,
    readNotifications,

    // Actions
    addNotification,
    markAsRead,
    markAllAsRead,
    clearHistory,
    togglePreference,
    setPreference,
    enableAll,
    disableAll,
    shouldShowNotification,
    getNotificationById,
    removeNotification
  }
})
