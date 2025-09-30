/**
 * Service Worker Registration
 * Registers the service worker for PWA offline capability
 */

export function register() {
  if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
      const swUrl = '/sw.js'

      navigator.serviceWorker
        .register(swUrl)
        .then((registration) => {
          console.log('[SW] Service Worker registered:', registration.scope)

          // Check for updates every hour
          setInterval(() => {
            registration.update()
          }, 60 * 60 * 1000)

          // Handle updates - auto-update without prompt
          registration.addEventListener('updatefound', () => {
            const newWorker = registration.installing

            newWorker.addEventListener('statechange', () => {
              if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                // New service worker available - auto-update
                console.log('[SW] New content available, auto-updating...')

                // Skip waiting and activate immediately
                newWorker.postMessage({ type: 'SKIP_WAITING' })

                // The controllerchange event will trigger reload
              }
            })
          })
        })
        .catch((error) => {
          console.error('[SW] Service Worker registration failed:', error)
        })

      // Reload page when new service worker takes control
      navigator.serviceWorker.addEventListener('controllerchange', () => {
        console.log('[SW] Controller changed, reloading page')
        window.location.reload()
      })
    })
  } else {
    console.log('[SW] Service Workers not supported')
  }
}

export function unregister() {
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.ready
      .then((registration) => {
        return registration.unregister()
      })
      .catch((error) => {
        console.error('[SW] Service Worker unregistration failed:', error)
      })
  }
}