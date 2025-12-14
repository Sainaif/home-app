import { ref, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'

/**
 * Composable for managing WebSocket connections for real-time events
 * Handles authentication, reconnection, and event dispatching
 * @version 3.0.0 - WebSocket implementation (secure - no token in URL)
 */
export function useEventStream() {
  const authStore = useAuthStore()
  const isConnected = ref(false)
  const isConnecting = ref(false)
  const isAuthenticated = ref(false)
  const error = ref(null)

  let socket = null
  let reconnectTimer = null
  let reconnectAttempts = 0
  const maxReconnectAttempts = 10
  const baseReconnectDelay = 1000 // 1 second

  const eventHandlers = new Map()

  /**
   * Build WebSocket URL from API URL
   */
  function getWebSocketURL() {
    const baseURL = import.meta.env.VITE_API_URL || '/api'

    // If baseURL is absolute, convert protocol
    if (baseURL.startsWith('http://') || baseURL.startsWith('https://')) {
      return baseURL.replace(/^http/, 'ws') + '/ws/events'
    }

    // Relative URL - construct from window.location
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host

    // Handle /api prefix
    if (baseURL.startsWith('/')) {
      return `${protocol}//${host}${baseURL}/ws/events`
    }

    return `${protocol}//${host}/api/ws/events`
  }

  /**
   * Connect to WebSocket stream
   */
  function connect() {
    if (!authStore.accessToken) {
      error.value = 'Brak tokenu uwierzytelniania'
      return
    }

    if (socket) {
      disconnect()
    }

    isConnecting.value = true
    isAuthenticated.value = false
    error.value = null

    try {
      const url = getWebSocketURL()
      socket = new WebSocket(url)

      socket.onopen = () => {
        console.log('[WS] Connected, sending auth...')

        // Send auth message immediately after connection
        socket.send(JSON.stringify({
          type: 'auth',
          token: authStore.accessToken
        }))
      }

      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data)

          switch (message.type) {
            case 'authenticated':
              console.log('[WS] Authentication successful')
              isConnected.value = true
              isConnecting.value = false
              isAuthenticated.value = true
              reconnectAttempts = 0
              error.value = null
              break

            case 'error':
              console.error('[WS] Server error:', message.data)
              error.value = typeof message.data === 'string'
                ? message.data
                : 'Błąd serwera'
              // If auth/token error, try to refresh token and reconnect
              if (message.data && (message.data.includes('token') || message.data.includes('expired'))) {
                handleTokenError()
              }
              break

            case 'event':
              // Parse the nested event data
              if (message.data) {
                const eventData = typeof message.data === 'string'
                  ? JSON.parse(message.data)
                  : message.data
                handleEvent(eventData)
              }
              break

            case 'heartbeat':
              // Heartbeat received - connection is alive
              break

            default:
              console.log('[WS] Unknown message type:', message.type)
          }
        } catch (err) {
          console.warn('[WS] Failed to parse message:', err)
        }
      }

      socket.onerror = (err) => {
        console.error('[WS] Connection error:', err)
        isConnected.value = false
        isConnecting.value = false
        isAuthenticated.value = false
        error.value = 'Błąd połączenia'
      }

      socket.onclose = (event) => {
        console.log('[WS] Connection closed:', event.code, event.reason)
        isConnected.value = false
        isConnecting.value = false
        isAuthenticated.value = false
        socket = null

        // Don't reconnect on normal close (1000) or auth failure
        if (event.code !== 1000 && authStore.accessToken) {
          scheduleReconnect()
        }
      }
    } catch (err) {
      console.error('[WS] Failed to create WebSocket:', err)
      isConnecting.value = false
      error.value = err.message
      scheduleReconnect()
    }
  }

  /**
   * Handle token error by refreshing token and reconnecting
   */
  async function handleTokenError() {
    console.log('[WS] Token error, attempting to refresh...')
    disconnect()

    try {
      const refreshed = await authStore.refresh()
      if (refreshed) {
        console.log('[WS] Token refreshed, reconnecting...')
        // Reset reconnect attempts since we have a fresh token
        reconnectAttempts = 0
        connect()
      } else {
        console.log('[WS] Token refresh failed, user needs to login again')
        error.value = 'Sesja wygasła, zaloguj się ponownie'
      }
    } catch (err) {
      console.error('[WS] Token refresh error:', err)
      error.value = 'Sesja wygasła, zaloguj się ponownie'
    }
  }

  /**
   * Schedule reconnection with exponential backoff
   */
  function scheduleReconnect() {
    if (reconnectAttempts >= maxReconnectAttempts) {
      console.warn('[WS] Max reconnection attempts reached')
      error.value = 'Osiągnięto maksymalną liczbę prób ponownego połączenia'
      return
    }

    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
    }

    const delay = Math.min(
      baseReconnectDelay * Math.pow(2, reconnectAttempts),
      30000 // Max 30 seconds
    )

    console.log(`[WS] Reconnecting in ${delay}ms (attempt ${reconnectAttempts + 1})`)

    reconnectTimer = setTimeout(() => {
      reconnectAttempts++
      connect()
    }, delay)
  }

  /**
   * Disconnect from WebSocket stream
   */
  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }

    if (socket) {
      socket.close(1000, 'Client disconnect')
      socket = null
    }

    isConnected.value = false
    isConnecting.value = false
    isAuthenticated.value = false
    reconnectAttempts = 0
  }

  /**
   * Register event handler
   * @param {string} eventType - Type of event (e.g., 'bill.created')
   * @param {Function} handler - Handler function to call when event received
   */
  function on(eventType, handler) {
    if (!eventHandlers.has(eventType)) {
      eventHandlers.set(eventType, [])
    }
    eventHandlers.get(eventType).push(handler)
  }

  /**
   * Unregister event handler
   * @param {string} eventType - Type of event
   * @param {Function} handler - Handler function to remove
   */
  function off(eventType, handler) {
    if (eventHandlers.has(eventType)) {
      const handlers = eventHandlers.get(eventType)
      const index = handlers.indexOf(handler)
      if (index > -1) {
        handlers.splice(index, 1)
      }
    }
  }

  /**
   * Handle incoming event
   * @param {Object} data - Event data from WebSocket
   */
  function handleEvent(data) {
    const { type } = data

    if (!type) {
      console.warn('[WS] Event has no type:', data)
      return
    }

    // Handle special connection event (legacy compatibility)
    if (type === 'connected') {
      console.log('[WS] Connection confirmed')
      return
    }

    // Call all registered handlers for this event type
    if (eventHandlers.has(type)) {
      const handlers = eventHandlers.get(type)
      handlers.forEach(handler => {
        try {
          handler(data)
        } catch (err) {
          console.error(`[WS] Error in handler for ${type}:`, err)
        }
      })
    }

    // Also call wildcard handlers
    if (eventHandlers.has('*')) {
      const handlers = eventHandlers.get('*')
      handlers.forEach(handler => {
        try {
          handler(data)
        } catch (err) {
          console.error('[WS] Error in wildcard handler:', err)
        }
      })
    }
  }

  /**
   * Cleanup on unmount
   */
  onUnmounted(() => {
    disconnect()
    eventHandlers.clear()
  })

  return {
    isConnected,
    isConnecting,
    isAuthenticated,
    error,
    connect,
    disconnect,
    on,
    off,
  }
}
