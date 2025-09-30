import { ref, onMounted, onUnmounted } from 'vue'

/**
 * Composable for checking if a new version of the app is available
 */
export function useVersionCheck() {
  const hasNewVersion = ref(false)
  const isUpdating = ref(false)
  let checkInterval = null
  let currentVersion = null

  async function checkVersion() {
    try {
      // Fetch version.json with cache busting
      const response = await fetch(`/version.json?t=${Date.now()}`)
      const data = await response.json()

      if (!currentVersion) {
        // First check - store current version
        currentVersion = data.timestamp
        console.log('[Version] Current version:', currentVersion)
      } else if (data.timestamp !== currentVersion) {
        // New version detected
        console.log('[Version] New version detected:', data.timestamp, 'Current:', currentVersion)
        hasNewVersion.value = true

        // Stop checking
        if (checkInterval) {
          clearInterval(checkInterval)
          checkInterval = null
        }

        // Automatically reload after showing message
        setTimeout(() => {
          reloadApp()
        }, 1500)
      }
    } catch (error) {
      console.warn('[Version] Failed to check version:', error)
    }
  }

  function startChecking(intervalMs = 60000) {
    // Initial check
    checkVersion()

    // Check periodically
    checkInterval = setInterval(checkVersion, intervalMs)
  }

  function stopChecking() {
    if (checkInterval) {
      clearInterval(checkInterval)
      checkInterval = null
    }
  }

  async function reloadApp() {
    isUpdating.value = true

    // Clear all caches
    if ('caches' in window) {
      const names = await caches.keys()
      await Promise.all(names.map(name => caches.delete(name)))
    }

    // Clear session storage
    sessionStorage.clear()

    // Wait a bit for the message to show, then force reload
    setTimeout(() => {
      // Force hard reload by adding timestamp
      window.location.href = window.location.href.split('?')[0] + '?t=' + Date.now()
    }, 1500)
  }

  onMounted(() => {
    startChecking()
  })

  onUnmounted(() => {
    stopChecking()
  })

  return {
    hasNewVersion,
    isUpdating,
    reloadApp
  }
}
