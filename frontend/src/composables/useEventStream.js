import { ref, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'

/**
 * Composable for managing SSE (Server-Sent Events) connections
 * Handles authentication, reconnection, and event dispatching
 * @version 2.0.0 - SSE disabled
 */
export function useEventStream() {
  const authStore = useAuthStore()
  const isConnected = ref(false)
  const isConnecting = ref(false)
  const error = ref(null)

  let eventSource = null
  let reconnectTimer = null
  let reconnectAttempts = 0
  const maxReconnectAttempts = 10
  const baseReconnectDelay = 1000 // 1 second

  const eventHandlers = new Map()

  /**
   * Connect to SSE stream
   */
  function connect() {
    if (!authStore.accessToken) {
      error.value = 'Brak tokenu uwierzytelniania'
      return
    }

    if (eventSource) {
      disconnect()
    }

    isConnecting.value = true
    error.value = null

    try {
      // EventSource doesn't support custom headers, so we pass token as query param
      const baseURL = import.meta.env.VITE_API_URL || 'http://localhost:3000'
      const url = `${baseURL}/events/stream?token=${authStore.accessToken}`

      eventSource = new EventSource(url)

      eventSource.onopen = () => {
        console.log('[SSE] Connected to event stream')
        isConnected.value = true
        isConnecting.value = false
        reconnectAttempts = 0
        error.value = null
      }

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          console.log('[SSE] Message received:', data)
          handleEvent(data)
        } catch (err) {
          console.warn('[SSE] Failed to parse message:', err)
        }
      }

      eventSource.onerror = (err) => {
        console.error('[SSE] Connection error:', err)
        isConnected.value = false
        isConnecting.value = false
        error.value = 'Błąd połączenia'

        // Cleanup and attempt reconnect
        if (eventSource) {
          eventSource.close()
          eventSource = null
        }

        scheduleReconnect()
      }
    } catch (err) {
      console.error('[SSE] Failed to create EventSource:', err)
      isConnecting.value = false
      error.value = err.message
      scheduleReconnect()
    }
  }

  /**
   * Schedule reconnection with exponential backoff
   */
  function scheduleReconnect() {
    if (reconnectAttempts >= maxReconnectAttempts) {
      console.warn('[SSE] Max reconnection attempts reached')
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

    console.log(`[SSE] Reconnecting in ${delay}ms (attempt ${reconnectAttempts + 1})`)

    reconnectTimer = setTimeout(() => {
      reconnectAttempts++
      connect()
    }, delay)
  }

  /**
   * Disconnect from SSE stream
   */
  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }

    if (eventSource) {
      eventSource.close()
      eventSource = null
    }

    isConnected.value = false
    isConnecting.value = false
    reconnectAttempts = 0
  }

  /**
   * Register event handler
   * @param {string} eventType - Type of event (e.g., 'prediction.updated')
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
   * @param {Object} data - Event data from SSE
   */
  function handleEvent(data) {
    const { type } = data

    if (!type) {
      console.warn('[SSE] Event has no type:', data)
      return
    }

    // Handle special connection event
    if (type === 'connected') {
      console.log('[SSE] Connection confirmed')
      return
    }

    // Call all registered handlers for this event type
    if (eventHandlers.has(type)) {
      const handlers = eventHandlers.get(type)
      handlers.forEach(handler => {
        try {
          handler(data)
        } catch (err) {
          console.error(`[SSE] Error in handler for ${type}:`, err)
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
          console.error('[SSE] Error in wildcard handler:', err)
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
    error,
    connect,
    disconnect,
    on,
    off,
  }
}