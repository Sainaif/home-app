<template>
  <div>
    <div class="page-header">
      <h1 class="text-3xl font-bold">{{ $t('chores.title') }}</h1>
      <div class="button-group">
        <button @click="showLeaderboard = !showLeaderboard" class="btn btn-outline">
          {{ showLeaderboard ? $t('chores.hideLeaderboard') : $t('chores.showLeaderboard') }}
        </button>
        <button v-if="authStore.hasPermission('chores.create')" @click="showCreateForm = !showCreateForm" class="btn btn-primary">
          {{ showCreateForm ? $t('common.cancel') : '+ ' + $t('chores.addChore') }}
        </button>
      </div>
    </div>

    <!-- Statistics Cards -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
      <div class="stat-card">
        <div class="text-gray-400 text-sm mb-1">{{ $t('chores.yourPoints') }}</div>
        <div class="text-3xl font-bold text-purple-400">{{ userStats?.totalPoints || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="text-gray-400 text-sm mb-1">{{ $t('chores.completed') }}</div>
        <div class="text-3xl font-bold text-green-400">{{ userStats?.completedChores || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="text-gray-400 text-sm mb-1">{{ $t('chores.pending') }}</div>
        <div class="text-3xl font-bold text-yellow-400">{{ userStats?.pendingChores || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="text-gray-400 text-sm mb-1">{{ $t('chores.punctuality') }}</div>
        <div class="text-3xl font-bold text-blue-400">{{ userStats?.onTimeRate?.toFixed(0) || 0 }}%</div>
      </div>
    </div>

    <!-- Leaderboard -->
    <div v-if="showLeaderboard" class="card mb-6">
      <h2 class="text-xl font-semibold mb-4 flex items-center gap-2">
        <span>🏆</span> {{ $t('chores.leaderboard') }}
      </h2>
      <div v-if="loadingLeaderboard" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead class="border-b border-gray-700">
            <tr class="text-left">
              <th class="pb-3">{{ $t('chores.rank') }}</th>
              <th class="pb-3">{{ $t('common.user') }}</th>
              <th class="pb-3">{{ $t('chores.points') }}</th>
              <th class="pb-3">{{ $t('chores.completed') }}</th>
              <th class="pb-3">{{ $t('chores.punctuality') }}</th>
              <th class="pb-3">{{ $t('chores.pending') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(user, index) in leaderboard" :key="user.userId"
                :class="user.userId === authStore.user?.id ? 'bg-purple-600/20' : ''"
                class="border-b border-gray-700">
              <td class="py-3">
                <span v-if="index === 0" class="text-2xl">🥇</span>
                <span v-else-if="index === 1" class="text-2xl">🥈</span>
                <span v-else-if="index === 2" class="text-2xl">🥉</span>
                <span v-else class="text-gray-400">#{{ index + 1 }}</span>
              </td>
              <td class="py-3 font-medium">{{ user.userName }}</td>
              <td class="py-3 text-purple-400 font-bold">{{ user.totalPoints }}</td>
              <td class="py-3 text-green-400">{{ user.completedChores }}</td>
              <td class="py-3">{{ user.onTimeRate.toFixed(0) }}%</td>
              <td class="py-3 text-yellow-400">{{ user.pendingChores }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Create Chore Form -->
    <div v-if="showCreateForm && authStore.hasPermission('chores.create')" class="card mb-6">
      <h2 class="text-xl font-semibold mb-4">{{ $t('chores.addNew') }}</h2>
      <form @submit.prevent="createChore" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('chores.name') }}</label>
            <input v-model="choreForm.name" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('chores.descriptionOptional') }}</label>
            <input v-model="choreForm.description" class="input" />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('chores.frequency') }}</label>
            <select v-model="choreForm.frequency" required class="input">
              <option value="daily">{{ $t('chores.daily') }}</option>
              <option value="weekly">{{ $t('chores.weekly') }}</option>
              <option value="monthly">{{ $t('chores.monthly') }}</option>
              <option value="custom">{{ $t('chores.custom') }}</option>
              <option value="irregular">{{ $t('chores.irregular') }}</option>
            </select>
          </div>
          <div v-if="choreForm.frequency === 'custom'">
            <label class="block text-sm font-medium mb-2">{{ $t('chores.interval') }}</label>
            <input v-model.number="choreForm.customInterval" type="number" min="1" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('chores.difficulty') }}</label>
            <select v-model.number="choreForm.difficulty" required class="input">
              <option :value="1">⭐ {{ $t('chores.veryEasy') }}</option>
              <option :value="2">⭐⭐ {{ $t('chores.easy') }}</option>
              <option :value="3">⭐⭐⭐ {{ $t('chores.medium') }}</option>
              <option :value="4">⭐⭐⭐⭐ {{ $t('chores.hard') }}</option>
              <option :value="5">⭐⭐⭐⭐⭐ {{ $t('chores.veryHard') }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('chores.priority') }}</label>
            <select v-model.number="choreForm.priority" required class="input">
              <option :value="1">{{ $t('chores.veryLow') }}</option>
              <option :value="2">{{ $t('chores.low') }}</option>
              <option :value="3">{{ $t('chores.medium') }}</option>
              <option :value="4">{{ $t('chores.high') }}</option>
              <option :value="5">{{ $t('chores.veryHigh') }}</option>
            </select>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('chores.assignmentMode') }}</label>
            <select v-model="choreForm.assignmentMode" required class="input">
              <option value="auto">{{ $t('chores.autoLeastLoaded') }}</option>
              <option value="manual">{{ $t('chores.manual') }}</option>
              <option value="round_robin">{{ $t('chores.roundRobin') }}</option>
              <option value="random">{{ $t('chores.random') }}</option>
            </select>
          </div>
          <div v-if="choreForm.assignmentMode === 'manual'">
            <label class="block text-sm font-medium mb-2">{{ $t('chores.assignTo') }}</label>
            <select v-model="choreForm.manualAssigneeId" required class="input">
              <option value="">{{ $t('chores.selectUser') }}</option>
              <option v-for="user in users" :key="user.id" :value="user.id">{{ user.name }}</option>
            </select>
          </div>
          <div v-else>
            <label class="block text-sm font-medium mb-2">{{ $t('chores.reminderHours') }}</label>
            <input v-model.number="choreForm.reminderHours" type="number" min="0" max="168" class="input" placeholder="24" />
          </div>
        </div>

        <div class="flex items-center gap-2">
          <input v-model="choreForm.notificationsEnabled" type="checkbox" id="notifications" class="w-4 h-4" />
          <label for="notifications" class="text-sm">{{ $t('chores.enableNotifications') }}</label>
        </div>

        <button type="submit" :disabled="creatingChore" class="btn btn-primary">
          {{ creatingChore ? $t('chores.creating') : $t('chores.addChore') }}
        </button>
      </form>
    </div>

    <!-- Filters -->
    <div class="card mb-6">
      <div class="flex flex-wrap gap-4">
        <div>
          <label class="block text-sm font-medium mb-2">{{ $t('chores.status') }}</label>
          <select v-model="filters.status" @change="loadAssignments" class="input">
            <option value="">{{ $t('chores.all') }}</option>
            <option value="pending">{{ $t('chores.pending') }}</option>
            <option value="in_progress">{{ $t('chores.inProgress') }}</option>
            <option value="done">{{ $t('chores.done') }}</option>
            <option value="overdue">{{ $t('chores.overdue') }}</option>
          </select>
        </div>
        <div v-if="authStore.hasPermission('chores.read')">
          <label class="block text-sm font-medium mb-2">{{ $t('common.user') }}</label>
          <select v-model="filters.userId" @change="loadAssignments" class="input">
            <option value="">{{ $t('chores.everyone') }}</option>
            <option v-for="user in users" :key="user.id" :value="user.id">{{ user.name }}</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-2">{{ $t('chores.sortByLabel') }}</label>
          <select v-model="filters.sortBy" @change="applySorting" class="input">
            <option value="dueDate">{{ $t('chores.deadline') }}</option>
            <option value="priority">{{ $t('chores.priority') }}</option>
            <option value="difficulty">{{ $t('chores.difficulty') }}</option>
            <option value="points">{{ $t('chores.points') }}</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Chores List -->
    <div class="card">
      <div v-if="loading" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else-if="!sortedAssignments || sortedAssignments.length === 0" class="text-center py-8 text-gray-400">
        {{ $t('chores.noChores') }}
      </div>
      <div v-else class="space-y-4">
        <div v-for="assignment in sortedAssignments" :key="assignment.id"
             class="p-4 rounded-xl bg-gray-700/30 hover:bg-gray-700/50 transition-colors">
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <div class="flex items-center gap-2 mb-2">
                <h3 class="text-lg font-semibold">{{ assignment.chore?.name || $t('chores.unknownChore') }}</h3>
                <span v-if="assignment.chore?.difficulty" class="text-sm">
                  {{ '⭐'.repeat(assignment.chore.difficulty) }}
                </span>
                <span v-if="assignment.chore?.priority >= 4" class="px-2 py-1 text-xs rounded bg-red-600/20 text-red-400">
                  {{ $t('chores.highPriority') }}
                </span>
              </div>

              <p v-if="assignment.chore?.description" class="text-sm text-gray-400 mb-2">
                {{ assignment.chore.description }}
              </p>

              <div class="flex flex-wrap gap-4 text-sm">
                <span class="text-gray-400">
                  👤 {{ assignment.userName }}
                </span>
                <span class="text-gray-400">
                  📅 {{ formatDate(assignment.dueDate) }}
                </span>
                <span :class="statusColor(assignment.status)">
                  {{ statusLabel(assignment.status) }}
                </span>
                <span class="text-purple-400 font-bold">
                  💎 {{ assignment.points }} {{ $t('chores.pts') }}
                </span>
                <span v-if="assignment.status === 'done' && assignment.isOnTime" class="text-green-400">
                  ⚡ {{ $t('chores.onTime') }}
                </span>
              </div>
            </div>

            <div class="flex gap-2">
              <button
                v-if="assignment.status === 'pending' && assignment.assigneeUserId === authStore.user?.id"
                @click="updateStatus(assignment.id, 'in_progress')"
                class="btn btn-sm btn-outline">
                {{ $t('chores.start') }}
              </button>
              <button
                v-if="assignment.status === 'in_progress' && assignment.assigneeUserId === authStore.user?.id"
                @click="updateStatus(assignment.id, 'done')"
                class="btn btn-sm btn-primary">
                {{ $t('chores.markDone') }}
              </button>
              <button
                v-if="authStore.hasPermission('reminders.send') && (assignment.status === 'pending' || assignment.status === 'in_progress') && assignment.assigneeUserId !== authStore.user?.id"
                @click="sendChoreReminder(assignment.id)"
                :disabled="sendingChoreReminder === assignment.id"
                class="btn btn-sm btn-secondary p-1"
                :title="$t('chores.sendReminder')">
                <svg v-if="sendingChoreReminder !== assignment.id" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/>
                  <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>
                </svg>
                <svg v-else class="w-4 h-4 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              </button>
              <button
                v-if="authStore.hasPermission('chores.delete')"
                @click="deleteChore(assignment.chore?.id)"
                :disabled="deletingChoreId === assignment.chore?.id"
                class="btn btn-sm btn-error">
                {{ deletingChoreId === assignment.chore?.id ? $t('chores.deleting') : $t('chores.delete') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useEventStream } from '../composables/useEventStream'
import { useDataEvents, DATA_EVENTS } from '../composables/useDataEvents'
import api from '../api/client'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const { on: onEvent } = useEventStream()
const { on: onDataEvent, emit } = useDataEvents()
const assignments = ref([])
const chores = ref([])
const users = ref([])
const leaderboard = ref([])
const userStats = ref(null)
const loading = ref(false)
const loadingLeaderboard = ref(false)
const creatingChore = ref(false)
const deletingChoreId = ref(null)
const sendingChoreReminder = ref(null)
const showLeaderboard = ref(false)
const showCreateForm = ref(false)

const choreForm = ref({
  name: '',
  description: '',
  frequency: 'weekly',
  customInterval: null,
  difficulty: 3,
  priority: 3,
  assignmentMode: 'auto',
  manualAssigneeId: '',
  notificationsEnabled: true,
  reminderHours: 24
})

const filters = ref({
  status: '',
  userId: '',
  sortBy: 'dueDate'
})

const sortedAssignments = computed(() => {
  if (!assignments.value) return []

  let result = [...assignments.value]

  switch (filters.value.sortBy) {
    case 'priority':
      result.sort((a, b) => (b.chore?.priority || 0) - (a.chore?.priority || 0))
      break
    case 'difficulty':
      result.sort((a, b) => (b.chore?.difficulty || 0) - (a.chore?.difficulty || 0))
      break
    case 'points':
      result.sort((a, b) => (b.points || 0) - (a.points || 0))
      break
    case 'dueDate':
    default:
      result.sort((a, b) => new Date(a.dueDate) - new Date(b.dueDate))
      break
  }

  return result
})

// Helper functions for event handlers to reduce duplication
function refreshChoresAndAssignments() {
  loadChores()
  loadAssignments()
}

function refreshAssignmentsAndLeaderboard() {
  loadAssignments()
  loadLeaderboard()
}

onMounted(async () => {
  // Load chores and users first (needed for enrichment)
  await Promise.all([
    loadChores(),
    loadLeaderboard(),
    loadUsers()
  ])

  // Then load assignments (which enriches with chore/user data)
  await loadAssignments()

  // Find user stats from leaderboard
  userStats.value = leaderboard.value.find(u => u.userId === authStore.user?.id)

  // Listen for chore-related WebSocket events
  onEvent('chore.updated', () => {
    console.log('[Chores] Chore updated event received, refreshing...')
    refreshChoresAndAssignments()
  })

  onEvent('chore.assigned', () => {
    console.log('[Chores] Chore assigned event received, refreshing...')
    refreshAssignmentsAndLeaderboard()
  })

  // Listen for local data events
  onDataEvent(DATA_EVENTS.CHORE_CREATED, refreshChoresAndAssignments)
  onDataEvent(DATA_EVENTS.CHORE_UPDATED, refreshChoresAndAssignments)
  onDataEvent(DATA_EVENTS.CHORE_DELETED, refreshChoresAndAssignments)
  onDataEvent(DATA_EVENTS.CHORE_ASSIGNED, refreshAssignmentsAndLeaderboard)
  onDataEvent(DATA_EVENTS.CHORE_ASSIGNMENT_UPDATED, refreshAssignmentsAndLeaderboard)
  onDataEvent(DATA_EVENTS.USER_UPDATED, loadUsers)
})

async function loadAssignments() {
  loading.value = true
  try {
    let url = '/chore-assignments'
    const params = []

    if (filters.value.status) {
      params.push(`status=${filters.value.status}`)
    }
    if (filters.value.userId) {
      params.push(`userId=${filters.value.userId}`)
    }

    if (params.length > 0) {
      url += '?' + params.join('&')
    }

    const response = await api.get(url)
    const assignmentsData = response.data || []

    // Enrich with chore and user details
    for (let assignment of assignmentsData) {
      const chore = chores.value.find(c => c.id === assignment.choreId)
      if (chore) {
        assignment.chore = chore
      }

      const user = users.value.find(u => u.id === assignment.assigneeUserId)
      if (user) {
        assignment.userName = user.name
      }
    }

    assignments.value = assignmentsData
  } catch (err) {
    console.error('Failed to load chore assignments:', err)
    assignments.value = []
  } finally {
    loading.value = false
  }
}

async function loadChores() {
  try {
    const response = await api.get('/chores')
    chores.value = response.data || []
  } catch (err) {
    console.error('Failed to load chores:', err)
  }
}

async function loadUsers() {
  try {
    const response = await api.get('/users')
    users.value = response.data || []
  } catch (err) {
    console.error('Failed to load users:', err)
  }
}

async function loadLeaderboard() {
  loadingLeaderboard.value = true
  try {
    const response = await api.get('/chores/leaderboard')
    leaderboard.value = response.data || []
  } catch (err) {
    console.error('Failed to load leaderboard:', err)
    leaderboard.value = []
  } finally {
    loadingLeaderboard.value = false
  }
}

async function createChore() {
  creatingChore.value = true
  try {
    const choreRes = await api.post('/chores', {
      name: choreForm.value.name,
      description: choreForm.value.description || undefined,
      frequency: choreForm.value.frequency,
      customInterval: choreForm.value.customInterval || undefined,
      difficulty: choreForm.value.difficulty,
      priority: choreForm.value.priority,
      assignmentMode: choreForm.value.assignmentMode,
      notificationsEnabled: choreForm.value.notificationsEnabled,
      reminderHours: choreForm.value.reminderHours || undefined
    })

    // Calculate due date based on frequency
    let dueDate = new Date()
    switch (choreForm.value.frequency) {
      case 'daily':
        dueDate.setDate(dueDate.getDate() + 1)
        break
      case 'weekly':
        dueDate.setDate(dueDate.getDate() + 7)
        break
      case 'monthly':
        dueDate.setMonth(dueDate.getMonth() + 1)
        break
      case 'custom':
        dueDate.setDate(dueDate.getDate() + (choreForm.value.customInterval || 7))
        break
      default:
        dueDate.setDate(dueDate.getDate() + 7)
    }

    // Auto-assign the chore based on assignment mode
    if (choreForm.value.assignmentMode === 'manual' && choreForm.value.manualAssigneeId) {
      // Manual assignment - assign to selected user
      await api.post('/chores/assign', {
        choreId: choreRes.data.id,
        assigneeUserId: choreForm.value.manualAssigneeId,
        dueDate: dueDate.toISOString()
      })
    } else if (choreForm.value.assignmentMode === 'auto') {
      await api.post(`/chores/${choreRes.data.id}/auto-assign`, {
        dueDate: dueDate.toISOString()
      })
    } else if (choreForm.value.assignmentMode === 'round_robin') {
      await api.post(`/chores/${choreRes.data.id}/rotate`, {
        dueDate: dueDate.toISOString()
      })
    } else if (choreForm.value.assignmentMode === 'random') {
      // Random assignment - let backend handle it via auto-assign for now
      await api.post(`/chores/${choreRes.data.id}/auto-assign`, {
        dueDate: dueDate.toISOString()
      })
    }

    // Reset form
    choreForm.value = {
      name: '',
      description: '',
      frequency: 'weekly',
      customInterval: null,
      difficulty: 3,
      priority: 3,
      assignmentMode: 'auto',
      manualAssigneeId: '',
      notificationsEnabled: true,
      reminderHours: 24
    }

    showCreateForm.value = false

    // Load chores first, then assignments (so enrichment works)
    await loadChores()
    await loadAssignments()
    emit(DATA_EVENTS.CHORE_CREATED)
  } catch (err) {
    console.error('Failed to create chore:', err)
    alert(t('chores.createError') + ' ' + (err.response?.data?.error || err.message))
  } finally {
    creatingChore.value = false
  }
}

async function updateStatus(assignmentId, status) {
  try {
    await api.patch(`/chore-assignments/${assignmentId}`, { status })
    await Promise.all([
      loadAssignments(),
      loadLeaderboard()
    ])

    // Update user stats
    userStats.value = leaderboard.value.find(u => u.userId === authStore.user?.id)
    emit(DATA_EVENTS.CHORE_ASSIGNMENT_UPDATED, { assignmentId })
  } catch (err) {
    console.error('Failed to update chore status:', err)
    alert(t('chores.updateStatusError') + ' ' + (err.response?.data?.error || err.message))
  }
}

async function deleteChore(choreId) {
  if (!choreId) return
  if (!confirm(t('chores.confirmDelete'))) return

  deletingChoreId.value = choreId
  try {
    const response = await api.delete(`/chores/${choreId}`)
    if (response.data?.requiresApproval) {
      alert(t('chores.deletionRequested'))
    } else {
      await loadChores()
      await loadAssignments()
      emit(DATA_EVENTS.CHORE_DELETED, { choreId })
    }
  } catch (err) {
    console.error('Failed to delete chore:', err)
    alert(t('chores.deleteError') + ' ' + (err.response?.data?.error || err.message))
  } finally {
    deletingChoreId.value = null
  }
}

function applySorting() {
  // Trigger computed property recalculation by updating ref
  assignments.value = [...assignments.value]
}

function formatDate(date) {
  const localeMap = { 'pl': 'pl-PL', 'en': 'en-US' }
  const dateLocale = localeMap[locale.value] || 'en-US'
  return new Date(date).toLocaleDateString(dateLocale, {
    day: 'numeric',
    month: 'short',
    year: 'numeric'
  })
}

function statusLabel(status) {
  const statusMap = {
    pending: 'chores.pending',
    in_progress: 'chores.inProgress',
    done: 'chores.done',
    overdue: 'chores.overdue'
  }
  return statusMap[status] ? t(statusMap[status]) : status
}

function statusColor(status) {
  const colors = {
    pending: 'text-yellow-400',
    in_progress: 'text-blue-400',
    done: 'text-green-400',
    overdue: 'text-red-400'
  }
  return colors[status] || 'text-gray-400'
}

async function sendChoreReminder(assignmentId) {
  sendingChoreReminder.value = assignmentId
  try {
    await api.post(`/reminders/chore/${assignmentId}`)
    alert(t('chores.reminderSent'))
  } catch (err) {
    console.error('Failed to send chore reminder:', err)
    alert(t('errors.sendReminderFailed') + ' ' + (err.response?.data?.error || err.message))
  } finally {
    sendingChoreReminder.value = null
  }
}
</script>
