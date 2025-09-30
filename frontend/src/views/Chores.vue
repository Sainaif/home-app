<template>
  <div>
    <h1 class="text-3xl font-bold mb-8">{{ $t('chores.title') }}</h1>

    <!-- Create Chore Form (Admin Only) -->
    <div v-if="authStore.isAdmin" class="card mb-6">
      <h2 class="text-xl font-semibold mb-4">Dodaj nowy obowiązek</h2>
      <form @submit.prevent="createChore" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">Nazwa obowiązku</label>
            <input v-model="choreForm.name" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Opis (opcjonalnie)</label>
            <input v-model="choreForm.description" class="input" />
          </div>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">Częstotliwość</label>
            <select v-model="choreForm.frequency" required class="input">
              <option value="daily">Dziennie</option>
              <option value="weekly">Tygodniowo</option>
              <option value="monthly">Miesięcznie</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Tryb rotacji</label>
            <select v-model="choreForm.rotationMode" required class="input">
              <option value="round_robin">Kolejno (round robin)</option>
              <option value="random">Losowo</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Przypisz do</label>
            <select v-model="choreForm.initialUserId" required class="input">
              <option value="">Wybierz użytkownika</option>
              <option v-for="user in users" :key="user.id" :value="user.id">{{ user.name }}</option>
            </select>
          </div>
        </div>
        <button type="submit" :disabled="creatingChore" class="btn btn-primary">
          {{ creatingChore ? 'Dodawanie...' : 'Dodaj obowiązek' }}
        </button>
      </form>
    </div>

    <div class="card">
      <div v-if="loading" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else-if="!assignments || assignments.length === 0" class="text-center py-8 text-gray-400">Brak obowiązków</div>
      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead class="border-b border-gray-700">
            <tr class="text-left">
              <th class="pb-3">Obowiązek</th>
              <th class="pb-3">{{ $t('chores.assigned') }}</th>
              <th class="pb-3">{{ $t('chores.dueDate') }}</th>
              <th class="pb-3">{{ $t('chores.status') }}</th>
              <th class="pb-3">{{ $t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="assignment in assignments" :key="assignment.id" class="border-b border-gray-700">
              <td class="py-3">{{ assignment.choreName }}</td>
              <td class="py-3">{{ assignment.userName }}</td>
              <td class="py-3">{{ formatDate(assignment.dueDate) }}</td>
              <td class="py-3">
                <span :class="statusClass(assignment.status)">
                  {{ $t(`chores.${assignment.status}`) }}
                </span>
              </td>
              <td class="py-3">
                <button
                  v-if="assignment.status === 'pending' && assignment.userId === authStore.user?.id"
                  @click="markDone(assignment.id)"
                  class="btn btn-primary text-sm">
                  {{ $t('chores.markDone') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'

const authStore = useAuthStore()
const assignments = ref([])
const users = ref([])
const loading = ref(false)
const creatingChore = ref(false)

const choreForm = ref({
  name: '',
  description: '',
  frequency: 'weekly',
  rotationMode: 'round_robin',
  initialUserId: ''
})

onMounted(async () => {
  await loadAssignments()
  if (authStore.isAdmin) {
    await loadUsers()
  }
})

async function loadAssignments() {
  loading.value = true
  try {
    const response = await api.get('/chore-assignments')
    assignments.value = response.data || []
  } catch (err) {
    console.error('Failed to load chores:', err)
    assignments.value = []
  } finally {
    loading.value = false
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

async function createChore() {
  creatingChore.value = true
  try {
    // Create chore
    const choreRes = await api.post('/chores', {
      name: choreForm.value.name,
      description: choreForm.value.description || undefined,
      frequency: choreForm.value.frequency,
      rotation_mode: choreForm.value.rotationMode
    })

    // Assign to initial user
    await api.post('/chores/assign', {
      chore_id: choreRes.data.id,
      user_id: choreForm.value.initialUserId,
      due_date: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString() // 7 days from now
    })

    // Reset form
    choreForm.value = {
      name: '',
      description: '',
      frequency: 'weekly',
      rotationMode: 'round_robin',
      initialUserId: ''
    }

    // Reload assignments
    await loadAssignments()
  } catch (err) {
    console.error('Failed to create chore:', err)
    alert('Błąd tworzenia obowiązku: ' + (err.response?.data?.error || err.message))
  } finally {
    creatingChore.value = false
  }
}

async function markDone(assignmentId) {
  try {
    await api.patch(`/chore-assignments/${assignmentId}`, { status: 'done' })
    await loadAssignments()
  } catch (err) {
    console.error('Failed to mark chore as done:', err)
  }
}

function formatDate(date) {
  return new Date(date).toLocaleDateString('pl-PL')
}

function statusClass(status) {
  return status === 'done' ? 'text-green-400' : 'text-yellow-400'
}
</script>