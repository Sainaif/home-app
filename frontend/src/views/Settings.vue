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

    <!-- Passkeys Section -->
    <div class="card mt-6">
      <div class="flex justify-between items-center mb-4">
        <div>
          <h2 class="text-xl font-semibold">Klucze dostępu (Passkeys)</h2>
          <p class="text-sm text-gray-400 mt-1">Szybsze i bezpieczniejsze logowanie bez hasła</p>
        </div>
        <button
          v-if="passkeySupported"
          @click="showAddPasskeyModal = true"
          class="btn btn-primary flex items-center gap-2">
          <Key class="w-4 h-4" />
          Dodaj passkey
        </button>
      </div>

      <div v-if="!passkeySupported" class="text-center py-8 text-gray-400">
        <ShieldOff class="w-12 h-12 mx-auto mb-2 opacity-50" />
        <p>Passkeys nie są obsługiwane w tej przeglądarce</p>
      </div>

      <div v-else-if="loadingPasskeys" class="text-center py-8">Ładowanie...</div>

      <div v-else-if="passkeys.length === 0" class="text-center py-8 text-gray-400">
        <Shield class="w-12 h-12 mx-auto mb-2 opacity-50" />
        <p>Nie masz jeszcze żadnych passkeys</p>
        <p class="text-sm mt-2">Dodaj passkey, aby logować się bez hasła</p>
      </div>

      <div v-else class="space-y-3">
        <div v-for="(passkey, index) in passkeys" :key="index"
             class="flex justify-between items-center p-4 bg-gray-700/30 rounded-xl">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg bg-purple-600/20 flex items-center justify-center">
              <Key class="w-5 h-5 text-purple-400" />
            </div>
            <div>
              <p class="font-medium">{{ passkey.name }}</p>
              <p class="text-xs text-gray-400">Utworzony: {{ passkey.createdAt }}</p>
              <p class="text-xs text-gray-400">Ostatnio użyty: {{ passkey.lastUsedAt }}</p>
            </div>
          </div>
          <button @click="confirmDeletePasskey(index)" class="btn btn-sm btn-secondary">
            <Trash class="w-3 h-3" />
          </button>
        </div>
      </div>

      <div v-if="passkeyError" class="mt-4 text-red-500 text-sm">{{ passkeyError }}</div>
      <div v-if="passkeySuccess" class="mt-4 text-green-500 text-sm">{{ passkeySuccess }}</div>
    </div>

    <!-- Add Passkey Modal -->
    <div v-if="showAddPasskeyModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showAddPasskeyModal = false">
      <div class="card max-w-lg w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold">Dodaj passkey</h2>
          <button @click="showAddPasskeyModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="addPasskey" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Nazwa passkey (opcjonalnie)</label>
            <input
              v-model="passkeyName"
              type="text"
              class="input"
              placeholder="np. MacBook Pro, iPhone" />
            <p class="text-xs text-gray-400 mt-1">Pomaga rozpoznać urządzenie</p>
          </div>

          <div v-if="passkeyError" class="text-red-500 text-sm">{{ passkeyError }}</div>

          <div class="flex gap-3">
            <button type="submit" :disabled="addingPasskey" class="btn btn-primary flex-1">
              {{ addingPasskey ? 'Dodawanie...' : 'Dodaj' }}
            </button>
            <button type="button" @click="showAddPasskeyModal = false" class="btn btn-outline">
              Anuluj
            </button>
          </div>
        </form>
      </div>
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
                  <div class="flex flex-wrap gap-2">
                    <button @click="viewUserDashboard(user.id)" class="text-blue-400 hover:text-blue-300 text-sm">
                      Dashboard
                    </button>
                    <button
                      @click="editUser(user)"
                      class="btn btn-sm btn-outline"
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
                    <button
                      @click="forcePasswordChange(user)"
                      class="btn btn-sm btn-outline"
                      :disabled="user.id === authStore.user.id"
                      title="Wymuś zmianę hasła">
                      Zmień hasło
                    </button>
                    <button
                      @click="deleteUser(user)"
                      class="btn btn-sm btn-secondary"
                      :disabled="user.id === authStore.user.id || user.isActive"
                      title="Usuń użytkownika (tylko nieaktywnych)">
                      <Trash class="w-3 h-3" />
                    </button>
                  </div>
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

      <!-- Backup & Restore -->
      <div class="card">
        <div class="flex justify-between items-center mb-4">
          <div>
            <h2 class="text-xl font-semibold">Kopia zapasowa i przywracanie</h2>
            <p class="text-sm text-gray-400 mt-1">Eksportuj lub importuj wszystkie dane systemu</p>
          </div>
        </div>

        <div class="space-y-4">
          <div class="p-4 bg-gray-700/30 rounded-xl">
            <div class="flex items-start gap-3">
              <div class="w-10 h-10 rounded-lg bg-green-600/20 flex items-center justify-center flex-shrink-0">
                <Download class="w-5 h-5 text-green-400" />
              </div>
              <div class="flex-1">
                <h3 class="font-medium mb-1">Eksportuj dane</h3>
                <p class="text-sm text-gray-400 mb-3">
                  Pobierz pełną kopię zapasową wszystkich danych (użytkownicy, rachunki, historia, passkeys)
                </p>
                <button @click="exportBackup" :disabled="exportingBackup" class="btn btn-primary flex items-center gap-2">
                  <Download class="w-4 h-4" />
                  {{ exportingBackup ? 'Eksportowanie...' : 'Pobierz kopię zapasową' }}
                </button>
              </div>
            </div>
          </div>

          <div class="p-4 bg-gray-700/30 rounded-xl border-2 border-red-600/20">
            <div class="flex items-start gap-3">
              <div class="w-10 h-10 rounded-lg bg-red-600/20 flex items-center justify-center flex-shrink-0">
                <Upload class="w-5 h-5 text-red-400" />
              </div>
              <div class="flex-1">
                <h3 class="font-medium mb-1 text-red-400">Importuj dane (NIEBEZPIECZNE)</h3>
                <p class="text-sm text-gray-400 mb-3">
                  <strong class="text-red-400">OSTRZEŻENIE:</strong> Operacja usunie WSZYSTKIE obecne dane i zastąpi je danymi z kopii zapasowej. Tej operacji nie można cofnąć!
                </p>
                <div class="space-y-2">
                  <input
                    type="file"
                    ref="backupFileInput"
                    accept=".json"
                    @change="handleBackupFileSelect"
                    class="hidden" />
                  <button
                    @click="$refs.backupFileInput.click()"
                    :disabled="importingBackup"
                    class="btn btn-secondary flex items-center gap-2">
                    <Upload class="w-4 h-4" />
                    {{ importingBackup ? 'Importowanie...' : 'Wybierz plik kopii zapasowej' }}
                  </button>
                  <p v-if="selectedBackupFile" class="text-sm text-gray-400">
                    Wybrany plik: {{ selectedBackupFile.name }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div v-if="backupError" class="p-4 bg-red-600/20 rounded-xl border border-red-600/50">
            <p class="text-red-400 text-sm">{{ backupError }}</p>
          </div>
          <div v-if="backupSuccess" class="p-4 bg-green-600/20 rounded-xl border border-green-600/50">
            <p class="text-green-400 text-sm">{{ backupSuccess }}</p>
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
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { usePasskey } from '../composables/usePasskey'
import { useDataEvents, DATA_EVENTS } from '../composables/useDataEvents'
import api from '../api/client'
import { UserPlus, Users, Edit, Trash, X, Key, Shield, ShieldOff, Download, Upload } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const { checkSupport, listPasskeys, register: registerPasskey, deletePasskey: removePasskey } = usePasskey()
const { emit } = useDataEvents()

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

// Backup state
const exportingBackup = ref(false)
const importingBackup = ref(false)
const selectedBackupFile = ref(null)
const backupFileInput = ref(null)
const backupError = ref('')
const backupSuccess = ref('')

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

// Passkey state
const passkeySupported = ref(false)
const passkeys = ref([])
const loadingPasskeys = ref(false)
const addingPasskey = ref(false)
const showAddPasskeyModal = ref(false)
const passkeyName = ref('')
const passkeyError = ref('')
const passkeySuccess = ref('')

onMounted(async () => {
  // Initialize profile form with current user data
  profileForm.value = {
    name: authStore.user?.name || '',
    email: authStore.user?.email || ''
  }

  if (authStore.isAdmin) {
    await Promise.all([loadUsers(), loadGroups()])
  }

  // Check passkey support
  const support = await checkSupport()
  passkeySupported.value = support.supported

  // Load passkeys if supported
  if (passkeySupported.value) {
    await loadPasskeyList()
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

    emit(DATA_EVENTS.USER_UPDATED, { userId: authStore.user.id })
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
      groupId: newUser.value.groupId || undefined
    })

    showCreateUserModal.value = false
    newUser.value = { name: '', email: '', password: '', role: 'RESIDENT', groupId: '' }
    await loadUsers()
    emit(DATA_EVENTS.USER_CREATED)
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
    const payload = {
      name: editUserForm.value.name,
      email: editUserForm.value.email,
      role: editUserForm.value.role
    }

    // If groupId is empty string, send zero ObjectID to remove group
    // If groupId has a value, send it
    // If groupId is undefined, don't include it (don't update)
    if (editUserForm.value.groupId === '') {
      payload.groupId = '000000000000000000000000'
    } else if (editUserForm.value.groupId) {
      payload.groupId = editUserForm.value.groupId
    }

    await api.patch(`/users/${editUserForm.value.id}`, payload)

    showEditUserModal.value = false
    await loadUsers()
    emit(DATA_EVENTS.USER_UPDATED, { userId: editUserForm.value.id })
  } catch (err) {
    userError.value = err.response?.data?.error || 'Nie udało się zaktualizować użytkownika'
  } finally {
    updatingUser.value = false
  }
}

async function toggleUserStatus(user) {
  try {
    await api.patch(`/users/${user.id}`, {
      isActive: !user.isActive
    })
    await loadUsers()
    emit(DATA_EVENTS.USER_UPDATED, { userId: user.id })
  } catch (err) {
    console.error('Failed to toggle user status:', err)
    alert('Nie udało się zmienić statusu użytkownika: ' + (err.response?.data?.error || err.message))
  }
}

async function forcePasswordChange(user) {
  if (!confirm(`Czy na pewno chcesz wymusić zmianę hasła dla użytkownika ${user.name}?\n\nUżytkownik będzie musiał zmienić hasło przy następnym logowaniu TYLKO przy użyciu hasła (nie passkey).`)) {
    return
  }

  try {
    await api.post(`/users/${user.id}/force-password-change`)
    await loadUsers()
    alert('Wymuszono zmianę hasła dla użytkownika')
  } catch (err) {
    console.error('Failed to force password change:', err)
    alert('Nie udało się wymusić zmiany hasła: ' + (err.response?.data?.error || err.message))
  }
}

async function deleteUser(user) {
  if (user.isActive) {
    alert('Nie można usunąć aktywnego użytkownika. Najpierw dezaktywuj użytkownika.')
    return
  }

  if (!confirm(`Czy na pewno chcesz USUNĄĆ użytkownika ${user.name}?\n\nTej operacji NIE MOŻNA cofnąć!\n\nWszystkie dane użytkownika (historia, rachunki, obowiązki) zostaną zachowane, ale użytkownik nie będzie mógł się zalogować.`)) {
    return
  }

  try {
    await api.delete(`/users/${user.id}`)
    await loadUsers()
    emit(DATA_EVENTS.USER_DELETED, { userId: user.id })
    alert('Użytkownik został usunięty')
  } catch (err) {
    console.error('Failed to delete user:', err)
    alert('Nie udało się usunąć użytkownika: ' + (err.response?.data?.error || err.message))
  }
}

function viewUserDashboard(userId) {
  router.push(`/dashboard/${userId}`)
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
      emit(DATA_EVENTS.GROUP_UPDATED, { groupId: editingGroup.value.id })
    } else {
      // Create new group
      await api.post('/groups', {
        name: groupForm.value.name,
        weight: groupForm.value.weight
      })
      emit(DATA_EVENTS.GROUP_CREATED)
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
  if (!confirm('Czy na pewno chcesz usunąć tę grupę?\n\nUWAGA: Nie można usunąć grupy, która ma przypisanych użytkowników.\nNajpierw usuń wszystkich użytkowników z tej grupy.')) return

  try {
    await api.delete(`/groups/${groupId}`)
    await loadGroups()
    await loadUsers() // Refresh users to update group assignments
    emit(DATA_EVENTS.GROUP_DELETED, { groupId })
    alert('Grupa została usunięta pomyślnie')
  } catch (err) {
    console.error('Failed to delete group:', err)
    console.error('Error response:', err.response)
    console.error('Error data:', err.response?.data)
    const errorMsg = err.response?.data?.error || err.message
    console.error('Parsed error message:', errorMsg)

    if (errorMsg && errorMsg.includes('users are still assigned')) {
      alert('Nie można usunąć grupy: Grupa ma przypisanych użytkowników.\n\nNajpierw usuń wszystkich użytkowników z tej grupy (zmień ich grupę na "Brak grupy" w edycji użytkownika).')
    } else {
      alert('Nie udało się usunąć grupy: ' + errorMsg)
    }
  }
}

function closeGroupModal() {
  showCreateGroupModal.value = false
  editingGroup.value = null
  groupForm.value = { name: '', weight: 1.0 }
  groupError.value = ''
}

// Passkey functions
async function loadPasskeyList() {
  loadingPasskeys.value = true
  passkeyError.value = ''
  try {
    const list = await listPasskeys()
    passkeys.value = list || []
  } catch (err) {
    console.error('Failed to load passkeys:', err)
    passkeyError.value = 'Nie udało się załadować passkeys'
    passkeys.value = []
  } finally {
    loadingPasskeys.value = false
  }
}

async function addPasskey() {
  addingPasskey.value = true
  passkeyError.value = ''
  passkeySuccess.value = ''

  try {
    const name = passkeyName.value.trim() || `Passkey ${new Date().toLocaleDateString('pl-PL')}`
    await registerPasskey(name)

    passkeySuccess.value = 'Passkey dodany pomyślnie'
    showAddPasskeyModal.value = false
    passkeyName.value = ''

    // Reload passkey list
    await loadPasskeyList()
  } catch (err) {
    console.error('Failed to add passkey:', err)
    passkeyError.value = err.message || 'Nie udało się dodać passkey'
  } finally {
    addingPasskey.value = false
  }
}

async function confirmDeletePasskey(index) {
  if (!confirm('Czy na pewno chcesz usunąć ten passkey?')) return

  passkeyError.value = ''
  passkeySuccess.value = ''

  try {
    const passkey = passkeys.value[index]

    if (!passkey.credentialId) {
      passkeyError.value = 'Brak ID passkey'
      return
    }

    await removePasskey(passkey.credentialId)

    passkeySuccess.value = 'Passkey usunięty pomyślnie'

    // Reload passkey list
    await loadPasskeyList()
  } catch (err) {
    console.error('Failed to delete passkey:', err)
    passkeyError.value = err.message || 'Nie udało się usunąć passkey'
  }
}

// Backup functions
async function exportBackup() {
  exportingBackup.value = true
  backupError.value = ''
  backupSuccess.value = ''

  try {
    const response = await api.get('/backup/export', {
      responseType: 'blob'
    })

    // Create download link
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', `holy-home-backup-${new Date().toISOString().split('T')[0]}.json`)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)

    backupSuccess.value = 'Kopia zapasowa została pobrana pomyślnie'
  } catch (err) {
    console.error('Failed to export backup:', err)
    backupError.value = err.response?.data?.error || 'Nie udało się wyeksportować kopii zapasowej'
  } finally {
    exportingBackup.value = false
  }
}

function handleBackupFileSelect(event) {
  const file = event.target.files[0]
  if (!file) {
    selectedBackupFile.value = null
    return
  }

  selectedBackupFile.value = file

  // Show confirmation dialog
  if (confirm(`OSTRZEŻENIE: Czy na pewno chcesz zaimportować kopię zapasową?\n\nOperacja:\n- Usunie WSZYSTKIE obecne dane\n- Zastąpi je danymi z pliku: ${file.name}\n- Tej operacji NIE MOŻNA cofnąć\n\nKliknij OK, aby kontynuować lub Anuluj, aby przerwać.`)) {
    importBackup(file)
  } else {
    selectedBackupFile.value = null
    event.target.value = ''
  }
}

async function importBackup(file) {
  importingBackup.value = true
  backupError.value = ''
  backupSuccess.value = ''

  try {
    const response = await api.post('/backup/import', file, {
      headers: {
        'Content-Type': 'application/json'
      }
    })

    backupSuccess.value = 'Kopia zapasowa została zaimportowana pomyślnie. Odśwież stronę, aby zobaczyć zmiany.'

    // Clear file selection
    selectedBackupFile.value = null
    if (backupFileInput.value) {
      backupFileInput.value.value = ''
    }

    // Optionally reload page after 3 seconds
    setTimeout(() => {
      window.location.reload()
    }, 3000)
  } catch (err) {
    console.error('Failed to import backup:', err)
    backupError.value = err.response?.data?.error || 'Nie udało się zaimportować kopii zapasowej'
    selectedBackupFile.value = null
    if (backupFileInput.value) {
      backupFileInput.value.value = ''
    }
  } finally {
    importingBackup.value = false
  }
}
</script>