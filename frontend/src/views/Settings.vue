<template>
  <div>
    <h1 class="text-3xl font-bold mb-8">{{ $t('settings.title') }}</h1>

    <div class="card">
      <h2 class="text-xl font-semibold mb-4">{{ $t('settings.profile') }}</h2>
      <form @submit.prevent="updateProfile" class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-2">{{ $t('settings.name') }}</label>
          <input v-model="profileForm.name" required class="input" />
        </div>

        <div>
          <label class="block text-sm font-medium mb-2">{{ $t('settings.email') }}</label>
          <input v-model="profileForm.email" type="email" required class="input" />
        </div>

        <div>
          <label class="block text-sm font-medium mb-2">Rola</label>
          <input :value="authStore.user.role" disabled class="input bg-gray-700" />
        </div>

        <div v-if="authStore.user.groupId">
          <label class="block text-sm font-medium mb-2">Grupa</label>
          <input :value="authStore.user.groupName || authStore.user.groupId" disabled class="input bg-gray-700" />
        </div>

        <div v-if="profileError" class="text-red-500 text-sm">{{ profileError }}</div>
        <div v-if="profileSuccess" class="text-green-500 text-sm">{{ profileSuccess }}</div>

        <button type="submit" :disabled="updatingProfile" class="btn btn-primary">
          {{ updatingProfile ? 'Zapisywanie...' : 'Zapisz profil' }}
        </button>
      </form>
    </div>

    <div class="card mt-6">
      <h2 class="text-xl font-semibold mb-4">{{ $t('settings.changePassword') }}</h2>
      <form @submit.prevent="changePassword" class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-2">{{ $t('settings.currentPassword') }}</label>
          <input v-model="passwordForm.currentPassword" type="password" required class="input" />
        </div>

        <div>
          <label class="block text-sm font-medium mb-2">{{ $t('settings.newPassword') }}</label>
          <input v-model="passwordForm.newPassword" type="password" required class="input" />
        </div>

        <div v-if="error" class="text-red-500 text-sm">{{ error }}</div>
        <div v-if="success" class="text-green-500 text-sm">{{ success }}</div>

        <button type="submit" :disabled="loading" class="btn btn-primary">
          {{ $t('settings.save') }}
        </button>
      </form>
    </div>

    <!-- Admin Section -->
    <div v-if="authStore.isAdmin" class="mt-8 space-y-6">
      <!-- Users Management -->
      <div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold">Zarządzanie użytkownikami</h2>
          <button @click="showCreateUserModal = true" class="btn btn-primary flex items-center gap-2">
            <UserPlus class="w-4 h-4" />
            Dodaj użytkownika
          </button>
        </div>

        <div v-if="loadingUsers" class="text-center py-8">Ładowanie...</div>
        <div v-else-if="users.length === 0" class="text-center py-8 text-gray-400">Brak użytkowników</div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b border-gray-700">
              <tr class="text-left">
                <th class="pb-3">Nazwa</th>
                <th class="pb-3">Email</th>
                <th class="pb-3">Rola</th>
                <th class="pb-3">Grupa</th>
                <th class="pb-3">Status</th>
                <th class="pb-3">Akcje</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in users" :key="user.id" class="border-b border-gray-700">
                <td class="py-3">{{ user.name }}</td>
                <td class="py-3">{{ user.email }}</td>
                <td class="py-3">
                  <span :class="user.role === 'ADMIN' ? 'text-purple-400' : 'text-gray-400'">
                    {{ user.role }}
                  </span>
                </td>
                <td class="py-3">{{ user.groupName || '-' }}</td>
                <td class="py-3">
                  <span :class="user.isActive ? 'text-green-400' : 'text-red-400'">
                    {{ user.isActive ? 'Aktywny' : 'Nieaktywny' }}
                  </span>
                </td>
                <td class="py-3">
                  <button
                    @click="editUser(user)"
                    class="btn btn-sm btn-outline mr-2"
                    :disabled="user.id === authStore.user.id">
                    <Edit class="w-3 h-3" />
                  </button>
                  <button
                    @click="toggleUserStatus(user)"
                    class="btn btn-sm"
                    :class="user.isActive ? 'btn-secondary' : 'btn-primary'"
                    :disabled="user.id === authStore.user.id">
                    {{ user.isActive ? 'Dezaktywuj' : 'Aktywuj' }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Groups Management -->
      <div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold">Zarządzanie grupami</h2>
          <button @click="showCreateGroupModal = true" class="btn btn-primary flex items-center gap-2">
            <Users class="w-4 h-4" />
            Dodaj grupę
          </button>
        </div>

        <div v-if="loadingGroups" class="text-center py-8">Ładowanie...</div>
        <div v-else-if="groups.length === 0" class="text-center py-8 text-gray-400">Brak grup</div>
        <div v-else class="space-y-3">
          <div v-for="group in groups" :key="group.id"
               class="flex justify-between items-center p-4 bg-gray-700/30 rounded-xl">
            <div>
              <p class="font-medium">{{ group.name }}</p>
              <p class="text-sm text-gray-400">Waga: {{ parseFloat(group.weight.$numberDecimal || group.weight || 1).toFixed(2) }}</p>
            </div>
            <div class="flex gap-2">
              <button @click="editGroup(group)" class="btn btn-sm btn-outline">
                <Edit class="w-3 h-3" />
              </button>
              <button @click="deleteGroup(group.id)" class="btn btn-sm btn-secondary">
                <Trash class="w-3 h-3" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create User Modal -->
    <div v-if="showCreateUserModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showCreateUserModal = false">
      <div class="card max-w-lg w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold">Nowy użytkownik</h2>
          <button @click="showCreateUserModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="createUser" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Nazwa</label>
            <input v-model="newUser.name" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Email</label>
            <input v-model="newUser.email" type="email" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Hasło</label>
            <input v-model="newUser.password" type="password" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Rola</label>
            <select v-model="newUser.role" required class="input">
              <option value="RESIDENT">Mieszkaniec</option>
              <option value="ADMIN">Administrator</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Grupa</label>
            <select v-model="newUser.groupId" class="input">
              <option value="">Brak grupy</option>
              <option v-for="group in groups" :key="group.id" :value="group.id">
                {{ group.name }}
              </option>
            </select>
          </div>

          <div v-if="userError" class="text-red-500 text-sm">{{ userError }}</div>

          <div class="flex gap-3">
            <button type="submit" :disabled="creatingUser" class="btn btn-primary flex-1">
              {{ creatingUser ? 'Tworzenie...' : 'Utwórz' }}
            </button>
            <button type="button" @click="showCreateUserModal = false" class="btn btn-outline">
              Anuluj
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Create/Edit Group Modal -->
    <div v-if="showCreateGroupModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="closeGroupModal">
      <div class="card max-w-lg w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold">{{ editingGroup ? 'Edytuj grupę' : 'Nowa grupa' }}</h2>
          <button @click="closeGroupModal" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="saveGroup" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Nazwa</label>
            <input v-model="groupForm.name" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Waga (domyślnie 1.0)</label>
            <input v-model.number="groupForm.weight" type="number" step="0.01" required class="input" />
          </div>

          <div v-if="groupError" class="text-red-500 text-sm">{{ groupError }}</div>

          <div class="flex gap-3">
            <button type="submit" :disabled="savingGroup" class="btn btn-primary flex-1">
              {{ savingGroup ? 'Zapisywanie...' : 'Zapisz' }}
            </button>
            <button type="button" @click="closeGroupModal" class="btn btn-outline">
              Anuluj
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit User Modal -->
    <div v-if="showEditUserModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showEditUserModal = false">
      <div class="card max-w-lg w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold">Edytuj użytkownika</h2>
          <button @click="showEditUserModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="updateUser" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Nazwa</label>
            <input v-model="editUserForm.name" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Email</label>
            <input v-model="editUserForm.email" type="email" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Rola</label>
            <select v-model="editUserForm.role" required class="input">
              <option value="RESIDENT">Mieszkaniec</option>
              <option value="ADMIN">Administrator</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Grupa</label>
            <select v-model="editUserForm.groupId" class="input">
              <option value="">Brak grupy</option>
              <option v-for="group in groups" :key="group.id" :value="group.id">
                {{ group.name }}
              </option>
            </select>
          </div>

          <div v-if="userError" class="text-red-500 text-sm">{{ userError }}</div>

          <div class="flex gap-3">
            <button type="submit" :disabled="updatingUser" class="btn btn-primary flex-1">
              {{ updatingUser ? 'Zapisywanie...' : 'Zapisz' }}
            </button>
            <button type="button" @click="showEditUserModal = false" class="btn btn-outline">
              Anuluj
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'
import { UserPlus, Users, Edit, Trash, X } from 'lucide-vue-next'

const authStore = useAuthStore()

const profileForm = ref({
  name: authStore.user?.name || '',
  email: authStore.user?.email || ''
})

const passwordForm = ref({
  currentPassword: '',
  newPassword: ''
})

const loading = ref(false)
const error = ref('')
const success = ref('')
const updatingProfile = ref(false)
const profileError = ref('')
const profileSuccess = ref('')

// Admin state
const users = ref([])
const groups = ref([])
const loadingUsers = ref(false)
const loadingGroups = ref(false)

// Modals
const showCreateUserModal = ref(false)
const showEditUserModal = ref(false)
const showCreateGroupModal = ref(false)

// Forms
const newUser = ref({
  name: '',
  email: '',
  password: '',
  role: 'RESIDENT',
  groupId: ''
})

const editUserForm = ref({
  id: '',
  name: '',
  email: '',
  role: '',
  groupId: ''
})

const groupForm = ref({
  name: '',
  weight: 1.0
})

const editingGroup = ref(null)
const creatingUser = ref(false)
const updatingUser = ref(false)
const savingGroup = ref(false)
const userError = ref('')
const groupError = ref('')

onMounted(() => {
  // Initialize profile form with current user data
  profileForm.value = {
    name: authStore.user?.name || '',
    email: authStore.user?.email || ''
  }

  if (authStore.isAdmin) {
    loadUsers()
    loadGroups()
  }
})

async function updateProfile() {
  updatingProfile.value = true
  profileError.value = ''
  profileSuccess.value = ''

  try {
    const response = await api.patch(`/users/${authStore.user.id}`, {
      name: profileForm.value.name,
      email: profileForm.value.email
    })

    // Update auth store with new user data
    authStore.user.name = profileForm.value.name
    authStore.user.email = profileForm.value.email

    profileSuccess.value = 'Profil zaktualizowany pomyślnie'
  } catch (err) {
    profileError.value = err.response?.data?.error || 'Błąd aktualizacji profilu'
  } finally {
    updatingProfile.value = false
  }
}

async function changePassword() {
  loading.value = true
  error.value = ''
  success.value = ''

  try {
    await api.post('/users/change-password', {
      current_password: passwordForm.value.currentPassword,
      new_password: passwordForm.value.newPassword
    })

    success.value = 'Hasło zostało zmienione'
    passwordForm.value.currentPassword = ''
    passwordForm.value.newPassword = ''
  } catch (err) {
    error.value = err.response?.data?.error || 'Błąd zmiany hasła'
  } finally {
    loading.value = false
  }
}

// Admin functions
async function loadUsers() {
  loadingUsers.value = true
  try {
    const response = await api.get('/users')
    users.value = response.data || []
  } catch (err) {
    console.error('Failed to load users:', err)
    users.value = []
  } finally {
    loadingUsers.value = false
  }
}

async function loadGroups() {
  loadingGroups.value = true
  try {
    const response = await api.get('/groups')
    groups.value = response.data || []
  } catch (err) {
    console.error('Failed to load groups:', err)
    groups.value = []
  } finally {
    loadingGroups.value = false
  }
}

async function createUser() {
  creatingUser.value = true
  userError.value = ''

  try {
    await api.post('/users', {
      name: newUser.value.name,
      email: newUser.value.email,
      password: newUser.value.password,
      role: newUser.value.role,
      group_id: newUser.value.groupId || undefined
    })

    showCreateUserModal.value = false
    newUser.value = { name: '', email: '', password: '', role: 'RESIDENT', groupId: '' }
    await loadUsers()
  } catch (err) {
    userError.value = err.response?.data?.error || 'Nie udało się utworzyć użytkownika'
  } finally {
    creatingUser.value = false
  }
}

function editUser(user) {
  editUserForm.value = {
    id: user.id,
    name: user.name,
    email: user.email,
    role: user.role,
    groupId: user.groupId || ''
  }
  showEditUserModal.value = true
}

async function updateUser() {
  updatingUser.value = true
  userError.value = ''

  try {
    await api.patch(`/users/${editUserForm.value.id}`, {
      name: editUserForm.value.name,
      email: editUserForm.value.email,
      role: editUserForm.value.role,
      group_id: editUserForm.value.groupId || undefined
    })

    showEditUserModal.value = false
    await loadUsers()
  } catch (err) {
    userError.value = err.response?.data?.error || 'Nie udało się zaktualizować użytkownika'
  } finally {
    updatingUser.value = false
  }
}

async function toggleUserStatus(user) {
  try {
    await api.patch(`/users/${user.id}`, {
      is_active: !user.isActive
    })
    await loadUsers()
  } catch (err) {
    console.error('Failed to toggle user status:', err)
  }
}

function editGroup(group) {
  editingGroup.value = group
  groupForm.value = {
    name: group.name,
    weight: parseFloat(group.weight.$numberDecimal || group.weight || 1)
  }
  showCreateGroupModal.value = true
}

async function saveGroup() {
  savingGroup.value = true
  groupError.value = ''

  try {
    if (editingGroup.value) {
      // Update existing group
      await api.patch(`/groups/${editingGroup.value.id}`, {
        name: groupForm.value.name,
        weight: groupForm.value.weight
      })
    } else {
      // Create new group
      await api.post('/groups', {
        name: groupForm.value.name,
        weight: groupForm.value.weight
      })
    }

    closeGroupModal()
    await loadGroups()
  } catch (err) {
    groupError.value = err.response?.data?.error || 'Nie udało się zapisać grupy'
  } finally {
    savingGroup.value = false
  }
}

async function deleteGroup(groupId) {
  if (!confirm('Czy na pewno chcesz usunąć tę grupę?')) return

  try {
    await api.delete(`/groups/${groupId}`)
    await loadGroups()
  } catch (err) {
    console.error('Failed to delete group:', err)
  }
}

function closeGroupModal() {
  showCreateGroupModal.value = false
  editingGroup.value = null
  groupForm.value = { name: '', weight: 1.0 }
  groupError.value = ''
}
</script>