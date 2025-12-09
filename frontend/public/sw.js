// Holy Home - Service Worker for PWA with Push Notifications
import { precacheAndRoute, cleanupOutdatedCaches } from 'workbox-precaching'

// Precache and route assets injected by VitePWA
precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()

// Listen for skip waiting message
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting()
  }
})

// Activate event - claim clients
self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
})

// Push event - handle incoming push notifications
self.addEventListener('push', (event) => {
  let data = { title: 'Holy Home', body: 'Nowe powiadomienie' }

  if (event.data) {
    try {
      data = event.data.json()
    } catch (e) {
      data.body = event.data.text()
    }
  }

  const options = {
    body: data.body || data.Body || 'Nowe powiadomienie',
    icon: '/pwa-192x192.png',
    badge: '/pwa-192x192.png',
    vibrate: [100, 50, 100],
    data: data,
    actions: [
      { action: 'open', title: 'Otwórz' },
      { action: 'close', title: 'Zamknij' }
    ]
  }

  event.waitUntil(
    self.registration.showNotification(data.title || data.Title || 'Holy Home', options)
  )
})

// Notification click - open the app
self.addEventListener('notificationclick', (event) => {
  event.notification.close()

  if (event.action === 'close') {
    return
  }

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true })
      .then((clientList) => {
        // Focus existing window if available
        for (const client of clientList) {
          if (client.url.includes(self.location.origin) && 'focus' in client) {
            return client.focus()
          }
        }
        // Open new window
        if (clients.openWindow) {
          return clients.openWindow('/')
        }
      })
  )
})
