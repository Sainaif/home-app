import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import apiClient from '../api';

export const useNotificationStore = defineStore('notification', () => {
  const history = ref([]);
  const preferences = ref({
    bill: true,
    chore: true,
    supply: true,
    loan: true,
  });

  const unreadCount = computed(() => history.value.filter((n) => !n.read).length);

  async function fetchNotifications() {
    try {
      const response = await apiClient.get('/notifications');
      history.value = response.data;
    } catch (error) {
      console.error('Failed to fetch notifications:', error);
    }
  }

  async function markAsRead(id) {
    try {
      await apiClient.post(`/notifications/${id}/read`);
      const notification = history.value.find((n) => n.id === id);
      if (notification) {
        notification.read = true;
      }
    } catch (error) {
      console.error('Failed to mark notification as read:', error);
    }
  }

  async function markAllAsRead() {
    try {
      await apiClient.post('/notifications/read-all');
      history.value.forEach((n) => (n.read = true));
    } catch (error) {
      console.error('Failed to mark all notifications as read:', error);
    }
  }

  function addNotification(notification) {
    history.value.unshift(notification);
  }

  async function subscribe() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
      return;
    }

    const registration = await navigator.serviceWorker.ready;
    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: import.meta.env.VITE_VAPID_PUBLIC_KEY,
    });

    await apiClient.post('/web-push/subscribe', subscription);
  }

  async function fetchPreferences() {
    try {
      const response = await apiClient.get('/notifications/preferences');
      preferences.value = response.data;
    } catch (error) {
      console.error('Failed to fetch preferences:', error);
    }
  }

  async function updatePreferences(newPreferences) {
    try {
      const response = await apiClient.put('/notifications/preferences', newPreferences);
      preferences.value = response.data;
    } catch (error) {
      console.error('Failed to update preferences:', error);
    }
  }

  fetchPreferences();

  return {
    history,
    preferences,
    unreadCount,
    fetchNotifications,
    addNotification,
    markAsRead,
    markAllAsRead,
    subscribe,
    fetchPreferences,
    updatePreferences,
  };
});
