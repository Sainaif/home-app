import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import apiClient from '../api/client';
import { useAuthStore } from './auth';

const MAX_HISTORY_LENGTH = 100; // Limit notification history to prevent memory leaks

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
    // Add timestamp if not present
    const notificationWithTimestamp = {
      ...notification,
      timestamp: notification.timestamp || new Date().toISOString(),
    };
    history.value.unshift(notificationWithTimestamp);

    // Enforce history limit to prevent memory leaks
    if (history.value.length > MAX_HISTORY_LENGTH) {
      history.value = history.value.slice(0, MAX_HISTORY_LENGTH);
    }
  }

  // Check if a notification should be shown based on user preferences
  function shouldShowNotification(notification) {
    if (!notification || !notification.type) return true;

    // Extract category from notification type (e.g., 'bill.created' -> 'bill')
    const category = notification.type.split('.')[0];

    // Check if user has enabled notifications for this category
    return preferences.value[category] !== false;
  }

  async function subscribe() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
      return;
    }

    // Get VAPID public key from auth config
    const authStore = useAuthStore();
    const vapidPublicKey = authStore.authConfig.vapidPublicKey;
    if (!vapidPublicKey) {
      console.warn('VAPID public key not configured, push notifications disabled');
      return;
    }

    try {
      const registration = await navigator.serviceWorker.ready;

      // Check for existing subscription first
      let subscription = await registration.pushManager.getSubscription();

      // If no subscription exists, create one
      if (!subscription) {
        subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: vapidPublicKey,
        });
      }

      // Transform PushSubscription to match backend model
      // Browser returns { endpoint, expirationTime, keys: { p256dh, auth } }
      // Backend expects { endpoint, expirationTime, p256dh, auth }
      const subscriptionJson = subscription.toJSON();
      await apiClient.post('/web-push/subscribe', {
        endpoint: subscriptionJson.endpoint,
        expirationTime: subscriptionJson.expirationTime,
        p256dh: subscriptionJson.keys.p256dh,
        auth: subscriptionJson.keys.auth,
      });
    } catch (error) {
      console.error('Failed to subscribe to push notifications:', error);
    }
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

  // Note: fetchPreferences() is NOT called automatically on store init.
  // It should be called after successful login to avoid 401 errors on login page.

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
    shouldShowNotification,
  };
});
