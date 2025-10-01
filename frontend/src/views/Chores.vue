<template>
  <div>
    <div class="flex justify-between items-center mb-8">
      <h1 class="text-3xl font-bold">Obowiązki domowe</h1>
      <div class="flex gap-4">
        <button @click="showLeaderboard = !showLeaderboard" class="btn btn-outline">
          {{ showLeaderboard ? 'Ukryj ranking' : 'Pokaż ranking' }}
        </button>
        <button v-if="authStore.isAdmin" @click="showCreateForm = !showCreateForm" class="btn btn-primary">
          {{ showCreateForm ? 'Anuluj' : '+ Dodaj obowiązek' }}
        </button>
      </div>
    </div>

    <!-- Statistics Cards -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
      <div class="stat-card">
        <div class="text-gray-400 text-sm mb-1">Twoje punkty</div>
        <div class="text-3xl font-bold text-purple-400">{{ userStats?.totalPoints || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="text-gray-400 text-sm mb-1">Ukończone</div>
        <div class="text-3xl font-bold text-green-400">{{ userStats?.completedChores || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="text-gray-400 text-sm mb-1">Oczekujące</div>
        <div class="text-3xl font-bold text-yellow-400">{{ userStats?.pendingChores || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="text-gray-400 text-sm mb-1">Terminowość</div>
        <div class="text-3xl font-bold text-blue-400">{{ userStats?.onTimeRate?.toFixed(0) || 0 }}%</div>
      </div>
    </div>

    <!-- Leaderboard -->
    <div v-if="showLeaderboard" class="card mb-6">
      <h2 class="text-xl font-semibold mb-4 flex items-center gap-2">
        <span>🏆</span> Ranking użytkowników
      </h2>
      <div v-if="loadingLeaderboard" class="text-center py-8">Ładowanie...</div>
      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead class="border-b border-gray-700">
            <tr class="text-left">
              <th class="pb-3">Pozycja</th>
              <th class="pb-3">Użytkownik</th>
              <th class="pb-3">Punkty</th>
              <th class="pb-3">Ukończone</th>
              <th class="pb-3">Terminowość</th>
              <th class="pb-3">Oczekujące</th>
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

    <!-- Create Chore Form (Admin Only) -->
    <div v-if="authStore.isAdmin && showCreateForm" class="card mb-6">
      <h2 class="text-xl font-semibold mb-4">Dodaj nowy obowiązek</h2>
      <form @submit.prevent="createChore" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">Nazwa obowiązku *</label>
            <input v-model="choreForm.name" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Opis (opcjonalnie)</label>
            <input v-model="choreForm.description" class="input" />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">Częstotliwość *</label>
            <select v-model="choreForm.frequency" required class="input">
              <option value="daily">Codziennie</option>
              <option value="weekly">Tygodniowo</option>
              <option value="monthly">Miesięcznie</option>
              <option value="custom">Niestandardowa</option>
              <option value="irregular">Nieregularna</option>
            </select>
          </div>
          <div v-if="choreForm.frequency === 'custom'">
            <label class="block text-sm font-medium mb-2">Interwał (dni)</label>
            <input v-model.number="choreForm.customInterval" type="number" min="1" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Trudność (1-5) *</label>
            <select v-model.number="choreForm.difficulty" required class="input">
              <option :value="1">⭐ Bardzo łatwa</option>
              <option :value="2">⭐⭐ Łatwa</option>
              <option :value="3">⭐⭐⭐ Średnia</option>
              <option :value="4">⭐⭐⭐⭐ Trudna</option>
              <option :value="5">⭐⭐⭐⭐⭐ Bardzo trudna</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Priorytet (1-5) *</label>
            <select v-model.number="choreForm.priority" required class="input">
              <option :value="1">Bardzo niski</option>
              <option :value="2">Niski</option>
              <option :value="3">Średni</option>
              <option :value="4">Wysoki</option>
              <option :value="5">Bardzo wysoki</option>
            </select>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">Tryb przypisywania *</label>
            <select v-model="choreForm.assignmentMode" required class="input">
              <option value="auto">Automatycznie (najmniej obciążony)</option>
              <option value="manual">Ręcznie</option>
              <option value="round_robin">Kolejno (round robin)</option>
              <option value="random">Losowo</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Przypomnienie (godziny przed)</label>
            <input v-model.number="choreForm.reminderHours" type="number" min="0" max="168" class="input" placeholder="np. 24" />
          </div>
        </div>

        <div class="flex items-center gap-2">
          <input v-model="choreForm.notificationsEnabled" type="checkbox" id="notifications" class="w-4 h-4" />
          <label for="notifications" class="text-sm">Włącz powiadomienia dla tego obowiązku</label>
        </div>

        <button type="submit" :disabled="creatingChore" class="btn btn-primary">
          {{ creatingChore ? 'Dodawanie...' : 'Dodaj obowiązek' }}
        </button>
      </form>
    </div>

    <!-- Filters -->
    <div class="card mb-6">
      <div class="flex flex-wrap gap-4">
        <div>
          <label class="block text-sm font-medium mb-2">Status</label>
          <select v-model="filters.status" @change="loadAssignments" class="input">
            <option value="">Wszystkie</option>
            <option value="pending">Oczekujące</option>
            <option value="in_progress">W trakcie</option>
            <option value="done">Ukończone</option>
            <option value="overdue">Zaległe</option>
          </select>
        </div>
        <div v-if="authStore.isAdmin">
          <label class="block text-sm font-medium mb-2">Użytkownik</label>
          <select v-model="filters.userId" @change="loadAssignments" class="input">
            <option value="">Wszyscy</option>
            <option v-for="user in users" :key="user.id" :value="user.id">{{ user.name }}</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-2">Sortuj według</label>
          <select v-model="filters.sortBy" @change="applySorting" class="input">
            <option value="dueDate">Termin</option>
            <option value="priority">Priorytet</option>
            <option value="difficulty">Trudność</option>
            <option value="points">Punkty</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Chores List -->
    <div class="card">
      <div v-if="loading" class="text-center py-8">Ładowanie...</div>
      <div v-else-if="!sortedAssignments || sortedAssignments.length === 0" class="text-center py-8 text-gray-400">
        Brak obowiązków
      </div>
      <div v-else class="space-y-4">
        <div v-for="assignment in sortedAssignments" :key="assignment.id"
             class="p-4 rounded-xl bg-gray-700/30 hover:bg-gray-700/50 transition-colors">
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <div class="flex items-center gap-2 mb-2">
                <h3 class="text-lg font-semibold">{{ assignment.chore?.name || 'Nieznany obowiązek' }}</h3>
                <span v-if="assignment.chore?.difficulty" class="text-sm">
                  {{ '⭐'.repeat(assignment.chore.difficulty) }}
                </span>
                <span v-if="assignment.chore?.priority >= 4" class="px-2 py-1 text-xs rounded bg-red-600/20 text-red-400">
                  Wysoki priorytet
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
                  💎 {{ assignment.points }} pkt
                </span>
                <span v-if="assignment.status === 'done' && assignment.isOnTime" class="text-green-400">
                  ⚡ Na czas!
                </span>
              </div>
            </div>

            <div class="flex gap-2">
              <button
                v-if="assignment.status === 'pending' && assignment.assigneeUserId === authStore.user?.id"
                @click="updateStatus(assignment.id, 'in_progress')"
                class="btn btn-sm btn-outline">
                Rozpocznij
              </button>
              <button
                v-if="assignment.status === 'in_progress' && assignment.assigneeUserId === authStore.user?.id"
                @click="updateStatus(assignment.id, 'done')"
                class="btn btn-sm btn-primary">
                Oznacz jako ukończone
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
import { useAuthStore } from '../stores/auth'
import api from '../api/client'

const authStore = useAuthStore()
const assignments = ref([])
const chores = ref([])
const users = ref([])
const leaderboard = ref([])
const userStats = ref(null)
const loading = ref(false)
const loadingLeaderboard = ref(false)
const creatingChore = ref(false)
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

onMounted(async () => {
  await Promise.all([
    loadAssignments(),
    loadChores(),
    loadLeaderboard()
  ])

  if (authStore.isAdmin) {
    await loadUsers()
  }

  // Find user stats from leaderboard
  userStats.value = leaderboard.value.find(u => u.userId === authStore.user?.id)
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
    if (choreForm.value.assignmentMode === 'auto') {
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
      notificationsEnabled: true,
      reminderHours: 24
    }

    showCreateForm.value = false

    await Promise.all([
      loadAssignments(),
      loadChores()
    ])
  } catch (err) {
    console.error('Failed to create chore:', err)
    alert('Błąd tworzenia obowiązku: ' + (err.response?.data?.error || err.message))
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
  } catch (err) {
    console.error('Failed to update chore status:', err)
    alert('Błąd aktualizacji statusu: ' + (err.response?.data?.error || err.message))
  }
}

function applySorting() {
  // Trigger computed property recalculation by updating ref
  assignments.value = [...assignments.value]
}

function formatDate(date) {
  return new Date(date).toLocaleDateString('pl-PL', {
    day: 'numeric',
    month: 'short',
    year: 'numeric'
  })
}

function statusLabel(status) {
  const labels = {
    pending: 'Oczekujące',
    in_progress: 'W trakcie',
    done: 'Ukończone',
    overdue: 'Zaległe'
  }
  return labels[status] || status
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
</script>
