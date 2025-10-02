import { ref, onUnmounted } from 'vue'

// Simple event bus for cross-component data synchronization
class EventBus {
  constructor() {
    this.listeners = new Map()
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event).add(callback)
  }

  off(event, callback) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).delete(callback)
    }
  }

  emit(event, data) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).forEach(callback => callback(data))
    }
  }
}

const eventBus = new EventBus()

// Event types
export const DATA_EVENTS = {
  USER_UPDATED: 'user:updated',
  USER_CREATED: 'user:created',
  USER_DELETED: 'user:deleted',
  GROUP_UPDATED: 'group:updated',
  GROUP_CREATED: 'group:created',
  GROUP_DELETED: 'group:deleted',
  CHORE_CREATED: 'chore:created',
  CHORE_UPDATED: 'chore:updated',
  CHORE_DELETED: 'chore:deleted',
  CHORE_ASSIGNED: 'chore:assigned',
  CHORE_ASSIGNMENT_UPDATED: 'chore_assignment:updated',
  BILL_CREATED: 'bill:created',
  BILL_UPDATED: 'bill:updated',
  BILL_DELETED: 'bill:deleted',
  CONSUMPTION_CREATED: 'consumption:created',
  CONSUMPTION_DELETED: 'consumption:deleted',
  LOAN_CREATED: 'loan:created',
  LOAN_DELETED: 'loan:deleted',
  LOAN_PAYMENT_CREATED: 'loan_payment:created',
  SUPPLY_ITEM_CREATED: 'supply_item:created',
  SUPPLY_ITEM_UPDATED: 'supply_item:updated',
  SUPPLY_ITEM_DELETED: 'supply_item:deleted',
  SUPPLY_CONTRIBUTION_CREATED: 'supply_contribution:created'
}

/**
 * Composable for event-driven data synchronization
 * Usage:
 *   const { emit, on } = useDataEvents()
 *   emit(DATA_EVENTS.USER_UPDATED, { userId: '123' })
 *   on(DATA_EVENTS.USER_UPDATED, async () => { await loadUsers() })
 */
export function useDataEvents() {
  const listeners = ref([])

  const emit = (event, data) => {
    eventBus.emit(event, data)
  }

  const on = (event, callback) => {
    eventBus.on(event, callback)
    listeners.value.push({ event, callback })
  }

  // Auto-cleanup on unmount
  onUnmounted(() => {
    listeners.value.forEach(({ event, callback }) => {
      eventBus.off(event, callback)
    })
  })

  return {
    emit,
    on
  }
}
