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
          <input :value="getUserRoleDisplayName(authStore.user?.role)" disabled class="input bg-gray-700" />
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

    <!-- Sessions Section -->
    <div class="card mt-6">
      <h2 class="text-xl font-semibold mb-4">{{ $t('settings.sessions') }}</h2>

      <div v-if="loadingSessions" class="text-center py-8">
        {{ $t('common.loading') }}
      </div>

      <div v-else-if="sessions.length === 0" class="text-center py-8 text-gray-400">
        {{ $t('settings.noSessions') }}
      </div>

      <div v-else class="space-y-3">
        <div v-for="session in sessions" :key="session.id" class="border border-gray-700 rounded-lg p-4">
          <div class="flex items-start justify-between gap-4">
            <div class="flex-1">
              <div class="flex items-center gap-2 mb-2">
                <h3 class="font-semibold">{{ session.name }}</h3>
                <span v-if="isCurrentSession(session)" class="text-xs px-2 py-0.5 rounded-full bg-purple-600 text-white">
                  {{ $t('settings.currentSession') }}
                </span>
              </div>
              <div class="text-sm text-gray-400 space-y-1">
                <p>{{ $t('settings.createdAt') }}: {{ formatDate(session.createdAt) }}</p>
                <p>{{ $t('settings.lastUsed') }}: {{ formatDate(session.lastUsedAt) }}</p>
                <p v-if="session.ipAddress">IP: {{ session.ipAddress }}</p>
              </div>
            </div>
            <div class="flex gap-2">
              <button @click="openRenameSessionModal(session)" class="btn btn-sm btn-outline" :title="$t('settings.renameSession')">
                <Edit class="w-4 h-4" />
              </button>
              <button @click="deleteSession(session.id)" class="btn btn-sm btn-secondary" :title="$t('settings.deleteSession')">
                <Trash class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Rename Session Modal -->
    <div v-if="showRenameSessionModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showRenameSessionModal = false">
      <div class="card max-w-md w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-xl font-bold">{{ $t('settings.renameSession') }}</h2>
          <button @click="showRenameSessionModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="confirmRenameSession" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('settings.sessionName') }}</label>
            <input
              v-model="renameSessionForm.name"
              type="text"
              required
              class="input"
              placeholder="Chrome on Windows" />
          </div>

          <div class="flex gap-3">
            <button type="submit" :disabled="renamingSession" class="btn btn-primary flex-1">
              {{ renamingSession ? $t('common.saving') : $t('common.save') }}
            </button>
            <button type="button" @click="showRenameSessionModal = false" class="btn btn-outline">
              {{ $t('common.cancel') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Admin Section -->
    <div class="mt-8 space-y-6">
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
        <div v-else class="overflow-x-auto -mx-4 sm:mx-0">
          <table class="w-full min-w-max">
            <thead class="border-b border-gray-700">
              <tr class="text-left">
                <th class="pb-3 px-2 sm:px-0">Nazwa</th>
                <th class="pb-3 px-2 sm:px-0 hidden md:table-cell">Email</th>
                <th class="pb-3 px-2 sm:px-0">Rola</th>
                <th class="pb-3 px-2 sm:px-0 hidden lg:table-cell">Grupa</th>
                <th class="pb-3 px-2 sm:px-0 hidden sm:table-cell">Status</th>
                <th class="pb-3 px-2 sm:px-0">Akcje</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in users" :key="user.id" class="border-b border-gray-700">
                <td class="py-3 px-2 sm:px-0">
                  <div>{{ user.name }}</div>
                  <div class="text-xs text-gray-400 md:hidden">{{ user.email }}</div>
                </td>
                <td class="py-3 px-2 sm:px-0 hidden md:table-cell">{{ user.email }}</td>
                <td class="py-3 px-2 sm:px-0">
                  <span :class="user.role === 'ADMIN' ? 'text-purple-400' : 'text-gray-400'" class="text-xs sm:text-sm">
                    {{ getUserRoleDisplayName(user.role) }}
                  </span>
                </td>
                <td class="py-3 px-2 sm:px-0 hidden lg:table-cell">{{ user.groupName || '-' }}</td>
                <td class="py-3 px-2 sm:px-0 hidden sm:table-cell">
                  <span :class="user.isActive ? 'text-green-400' : 'text-red-400'" class="text-xs sm:text-sm">
                    {{ user.isActive ? 'Aktywny' : 'Nieaktywny' }}
                  </span>
                </td>
                <td class="py-3 px-2 sm:px-0">
                  <div class="flex flex-col sm:flex-row flex-wrap gap-1 sm:gap-2">
                    <button @click="viewUserDashboard(user.id)" class="text-blue-400 hover:text-blue-300 text-xs sm:text-sm whitespace-nowrap">
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
                      class="btn btn-sm text-xs"
                      :class="user.isActive ? 'btn-secondary' : 'btn-primary'"
                      :disabled="user.id === authStore.user.id">
                      {{ user.isActive ? 'Dezaktywuj' : 'Aktywuj' }}
                    </button>
                    <button
                      @click="forcePasswordChange(user)"
                      class="btn btn-sm btn-outline text-xs"
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
               class="p-4 bg-gray-700/30 rounded-xl">
            <div class="flex flex-col sm:flex-row justify-between items-start gap-3 mb-3">
              <div class="flex-1">
                <p class="font-medium">{{ group.name }}</p>
                <p class="text-sm text-gray-400">Waga: {{ parseFloat(group.weight.$numberDecimal || group.weight || 1).toFixed(2) }}</p>
              </div>
              <div class="flex flex-wrap gap-2 w-full sm:w-auto">
                <button @click="manageGroupUsers(group)" class="btn btn-sm btn-outline flex-1 sm:flex-initial">
                  <Users class="w-3 h-3 mr-1" />
                  <span class="text-xs">Użytkownicy</span>
                </button>
                <button @click="editGroup(group)" class="btn btn-sm btn-outline">
                  <Edit class="w-3 h-3" />
                </button>
                <button @click="deleteGroup(group.id)" class="btn btn-sm btn-secondary">
                  <Trash class="w-3 h-3" />
                </button>
              </div>
            </div>
            <div v-if="getUsersInGroup(group.id).length > 0" class="mt-2">
              <p class="text-xs text-gray-400 mb-1">Członkowie:</p>
              <div class="flex flex-wrap gap-1">
                <span v-for="user in getUsersInGroup(group.id)" :key="user.id"
                      class="text-xs px-2 py-1 bg-gray-800 rounded">
                  {{ user.name }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Audit Logs -->
      <div class="card">
        <div class="flex justify-between items-center mb-4">
          <div>
            <h2 class="text-xl font-semibold">Logi audytu</h2>
            <p class="text-sm text-gray-400 mt-1">Historia wszystkich akcji użytkowników i administratorów</p>
          </div>
          <button @click="loadAuditLogs" :disabled="loadingAuditLogs" class="btn btn-outline btn-sm">
            <RefreshCw class="w-4 h-4" />
          </button>
        </div>

        <!-- Comprehensive Filters -->
        <div class="space-y-3 mb-4">
          <!-- Search Bar -->
          <div class="relative">
            <input
              v-model="auditFilters.search"
              @input="debouncedLoadAuditLogs"
              type="text"
              placeholder="Szukaj w logach (użytkownik, akcja, zasób...)..."
              class="input pl-10" />
            <Search class="w-4 h-4 absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
          </div>

          <!-- Row 1: User, Action, Resource Type, Status -->
          <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
            <select v-model="auditFilters.userEmail" @change="loadAuditLogs" class="input">
              <option value="">Wszyscy użytkownicy</option>
              <option v-for="user in users" :key="user.id" :value="user.email">{{ user.name }} ({{ user.email }})</option>
            </select>

            <select v-model="auditFilters.action" @change="loadAuditLogs" class="input">
              <option value="">Wszystkie akcje</option>
              <optgroup label="Rachunki">
                <option value="create_bill">Tworzenie rachunku</option>
                <option value="post_bill">Publikacja rachunku</option>
                <option value="close_bill">Zamknięcie rachunku</option>
                <option value="delete_bill">Usunięcie rachunku</option>
              </optgroup>
              <optgroup label="Odczyty">
                <option value="create_reading">Dodanie odczytu</option>
                <option value="delete_consumption">Usunięcie odczytu</option>
              </optgroup>
              <optgroup label="Zakupy">
                <option value="create_supply">Dodanie artykułu</option>
                <option value="update_supply">Aktualizacja artykułu</option>
                <option value="delete_supply">Usunięcie artykułu</option>
                <option value="purchase_supply">Zakup artykułu</option>
              </optgroup>
              <optgroup label="Pożyczki">
                <option value="create_loan">Tworzenie pożyczki</option>
                <option value="payment_loan">Spłata pożyczki</option>
              </optgroup>
              <optgroup label="Obowiązki">
                <option value="create_chore">Tworzenie obowiązku</option>
                <option value="update_chore">Aktualizacja obowiązku</option>
                <option value="delete_chore">Usunięcie obowiązku</option>
                <option value="complete_chore">Wykonanie obowiązku</option>
              </optgroup>
              <optgroup label="Użytkownicy i role">
                <option value="user.create">Tworzenie użytkownika</option>
                <option value="user.update">Aktualizacja użytkownika</option>
                <option value="user.delete">Usunięcie użytkownika</option>
                <option value="role.create">Tworzenie roli</option>
                <option value="role.update">Aktualizacja roli</option>
                <option value="role.delete">Usunięcie roli</option>
              </optgroup>
            </select>

            <select v-model="auditFilters.resourceType" @change="loadAuditLogs" class="input">
              <option value="">Wszystkie zasoby</option>
              <option value="bill">Rachunek</option>
              <option value="consumption">Odczyt</option>
              <option value="supply">Zakup</option>
              <option value="loan">Pożyczka</option>
              <option value="chore">Obowiązek</option>
              <option value="user">Użytkownik</option>
              <option value="role">Rola</option>
            </select>

            <select v-model="auditFilters.status" @change="loadAuditLogs" class="input">
              <option value="">Wszystkie statusy</option>
              <option value="success">✓ Sukces</option>
              <option value="failure">✗ Błąd</option>
            </select>
          </div>

          <!-- Row 2: Date Range, Limit -->
          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input
              v-model="auditFilters.dateFrom"
              @change="loadAuditLogs"
              type="date"
              placeholder="Data od"
              class="input" />

            <input
              v-model="auditFilters.dateTo"
              @change="loadAuditLogs"
              type="date"
              placeholder="Data do"
              class="input" />

            <select v-model="auditLimit" @change="loadAuditLogs" class="input">
              <option :value="10">10 wpisów</option>
              <option :value="25">25 wpisów</option>
              <option :value="50">50 wpisów</option>
              <option :value="100">100 wpisów</option>
              <option :value="200">200 wpisów</option>
            </select>
          </div>

          <!-- Clear Filters Button -->
          <div class="flex justify-end">
            <button @click="clearAuditFilters" class="btn btn-outline btn-sm">
              Wyczyść filtry
            </button>
          </div>
        </div>

        <div v-if="loadingAuditLogs" class="text-center py-8">Ładowanie...</div>
        <div v-else-if="auditLogs.length === 0" class="text-center py-8 text-gray-400">Brak logów</div>
        <div v-else class="space-y-2 max-h-[600px] overflow-y-auto">
          <div v-for="log in auditLogs" :key="log.id"
               class="p-4 rounded-lg bg-gray-700/30 hover:bg-gray-700/50 transition-colors cursor-pointer"
               @click="toggleAuditLogDetails(log.id)">
            <div class="flex items-start justify-between">
              <div class="flex-1">
                <!-- Title with status -->
                <div class="flex items-center gap-2 mb-1">
                  <span class="font-medium">{{ getAuditLogTitle(log) }}</span>
                  <span :class="log.status === 'success' ? 'text-green-400' : 'text-red-400'" class="text-xs">
                    {{ log.status === 'success' ? '✓' : '✗' }}
                  </span>
                </div>

                <!-- Summary -->
                <div class="text-sm text-gray-300 mb-2" v-if="getAuditLogSummary(log)">
                  {{ getAuditLogSummary(log) }}
                </div>

                <!-- Metadata -->
                <div class="text-xs text-gray-500 flex flex-wrap gap-x-3 gap-y-1">
                  <span>{{ log.userName || log.userEmail }}</span>
                  <span>•</span>
                  <span>{{ formatDate(log.createdAt) }}</span>
                  <span v-if="log.resourceType && getResourceRoute(log)">•</span>
                  <router-link
                    v-if="log.resourceType && log.resourceId && log.status === 'success' && getResourceRoute(log)"
                    :to="getResourceRoute(log)"
                    @click.stop
                    class="text-purple-400 hover:text-purple-300 underline">
                    Zobacz {{ translateResourceType(log.resourceType) }}
                  </router-link>
                </div>

                <!-- Expandable Details -->
                <div v-if="expandedAuditLogs.has(log.id) && log.details && Object.keys(log.details).length > 0"
                     class="mt-3 pt-3 border-t border-gray-600">
                  <div class="text-sm space-y-2">
                    <div v-for="(value, key) in formatAuditDetails(log)" :key="key">
                      <div class="text-gray-400 font-medium mb-1">{{ translateDetailKey(key) }}</div>
                      <div class="text-gray-300 pl-3 break-words" style="white-space: pre-wrap;">{{ value }}</div>
                    </div>
                  </div>
                </div>
              </div>
              <svg
                class="w-5 h-5 text-gray-400 transition-transform flex-shrink-0"
                :class="{ 'rotate-180': expandedAuditLogs.has(log.id) }"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </div>
          </div>
          <div class="mt-3 text-sm text-gray-400 text-center">
            Pokazano {{ auditLogs.length }} z ostatnich wpisów
          </div>
        </div>
      </div>

      <!-- Role Management -->
      <div class="card">
        <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-4">
          <div class="flex-1">
            <h2 class="text-xl font-semibold">Zarządzanie rolami i uprawnieniami</h2>
            <p class="text-sm text-gray-400 mt-1">Konfiguruj role użytkowników i ich uprawnienia</p>
          </div>
          <button @click="showCreateRoleModal = true" class="btn btn-primary btn-sm flex items-center gap-2 w-full sm:w-auto">
            <UserPlus class="w-4 h-4" />
            Nowa rola
          </button>
        </div>

        <div v-if="loadingRoles" class="text-center py-8">Ładowanie...</div>
        <div v-else class="space-y-3">
          <div v-for="role in roles" :key="role.id" class="p-4 bg-gray-700/30 rounded-xl">
            <div class="flex flex-col sm:flex-row justify-between items-start gap-3 mb-3">
              <div class="flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <h3 class="font-semibold">{{ role.displayName }}</h3>
                  <span v-if="role.name === 'ADMIN'" class="text-xs px-2 py-0.5 bg-purple-600/20 text-purple-400 rounded whitespace-nowrap">Niezmienne</span>
                </div>
                <p class="text-sm text-gray-400 mt-1">{{ role.permissions.length }} uprawnień</p>
              </div>
              <div class="flex flex-col sm:flex-row gap-2 w-full sm:w-auto">
                <button v-if="role.name !== 'ADMIN'" @click="openEditRoleModal(role)" class="btn btn-sm btn-outline flex items-center justify-center gap-1">
                  <Edit class="w-3 h-3" />
                  <span class="text-xs">Edytuj</span>
                </button>
                <button v-if="role.name !== 'ADMIN'" @click="deleteRole(role.id)" class="btn btn-sm btn-secondary flex items-center justify-center gap-1">
                  <Trash class="w-3 h-3" />
                  <span class="text-xs">Usuń</span>
                </button>
                <span v-if="role.name === 'ADMIN'" class="text-xs sm:text-sm text-gray-500 italic text-center sm:text-left">Rola administratora (chroniona)</span>
              </div>
            </div>
            <div class="flex flex-wrap gap-2">
              <span v-for="perm in role.permissions.slice(0, 5)" :key="perm" class="text-xs px-2 py-1 bg-gray-800 rounded whitespace-nowrap">
                {{ translatePermissionName(perm) }}
              </span>
              <span v-if="role.permissions.length > 5" class="text-xs px-2 py-1 text-gray-400 whitespace-nowrap">
                +{{ role.permissions.length - 5 }} więcej
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Approval Requests -->
      <div class="card">
        <div class="flex justify-between items-center mb-4">
          <div>
            <h2 class="text-xl font-semibold">Oczekujące zatwierdzenia</h2>
            <p class="text-sm text-gray-400 mt-1">Wnioski użytkowników wymagające zgody administratora</p>
          </div>
          <button @click="loadApprovals" :disabled="loadingApprovals" class="btn btn-outline btn-sm">
            <RefreshCw class="w-4 h-4" />
          </button>
        </div>

        <div v-if="loadingApprovals" class="text-center py-8">Ładowanie...</div>
        <div v-else-if="pendingApprovals.length === 0" class="text-center py-8 text-gray-400">Brak oczekujących wniosków</div>
        <div v-else class="space-y-3">
          <div v-for="approval in pendingApprovals" :key="approval.id" class="p-4 bg-gray-700/30 rounded-xl">
            <div class="flex justify-between items-start">
              <div class="flex-1">
                <div class="flex items-center gap-2 mb-2">
                  <span class="font-semibold">{{ approval.userName || approval.userEmail }}</span>
                  <span class="text-sm text-gray-400">•</span>
                  <span class="text-sm font-mono text-gray-400">{{ approval.action }}</span>
                </div>
                <p class="text-sm text-gray-400">{{ formatDate(approval.createdAt) }}</p>
              </div>
              <div class="flex gap-2">
                <button @click="approveRequest(approval.id)" class="btn btn-sm btn-primary">
                  Zatwierdź
                </button>
                <button @click="rejectRequest(approval.id)" class="btn btn-sm btn-secondary">
                  Odrzuć
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Backup & Restore -->
      <div class="card">
        <div class="mb-4">
          <h2 class="text-xl font-semibold mb-1">Kopia zapasowa i przywracanie</h2>
          <p class="text-sm text-gray-400">Eksportuj lub importuj wszystkie dane systemu</p>
        </div>

        <div class="space-y-3">
          <!-- Export Section -->
          <div class="border border-gray-700 rounded-xl overflow-hidden">
            <button
              @click="showExportBackup = !showExportBackup"
              class="w-full p-4 bg-gray-700/30 hover:bg-gray-700/50 transition-colors flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-green-600/20 flex items-center justify-center flex-shrink-0">
                  <Download class="w-5 h-5 text-green-400" />
                </div>
                <div class="text-left">
                  <h3 class="font-medium">Eksportuj dane</h3>
                  <p class="text-xs text-gray-400">Pobierz pełną kopię zapasową</p>
                </div>
              </div>
              <svg class="w-5 h-5 transition-transform" :class="{ 'rotate-180': showExportBackup }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            <div v-if="showExportBackup" class="p-4 border-t border-gray-700">
              <p class="text-sm text-gray-400 mb-3">
                Pobierz pełną kopię zapasową wszystkich danych (użytkownicy, rachunki, historia, passkeys)
              </p>
              <button @click="exportBackup" :disabled="exportingBackup" class="btn btn-primary flex items-center gap-2">
                <Download class="w-4 h-4" />
                {{ exportingBackup ? 'Eksportowanie...' : 'Pobierz kopię zapasową' }}
              </button>
            </div>
          </div>

          <!-- Import Section -->
          <div class="border border-red-600/30 rounded-xl overflow-hidden">
            <button
              @click="showImportBackup = !showImportBackup"
              class="w-full p-4 bg-gray-700/30 hover:bg-gray-700/50 transition-colors flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-red-600/20 flex items-center justify-center flex-shrink-0">
                  <Upload class="w-5 h-5 text-red-400" />
                </div>
                <div class="text-left">
                  <h3 class="font-medium text-red-400">Importuj dane (NIEBEZPIECZNE)</h3>
                  <p class="text-xs text-gray-400">Zastąp wszystkie dane kopią zapasową</p>
                </div>
              </div>
              <svg class="w-5 h-5 transition-transform" :class="{ 'rotate-180': showImportBackup }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            <div v-if="showImportBackup" class="p-4 border-t border-red-600/30 bg-red-600/5">
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

          <!-- Status Messages -->
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
              <option v-for="role in roles" :key="role.id" :value="role.name">
                {{ role.displayName }}
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
              <option v-for="role in roles" :key="role.id" :value="role.name">
                {{ role.displayName }}
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

    <!-- Manage Group Users Modal -->
    <div v-if="showManageGroupUsersModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showManageGroupUsersModal = false">
      <div class="card max-w-lg w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold">Użytkownicy w grupie: {{ selectedGroup?.name }}</h2>
          <button @click="showManageGroupUsersModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Dodaj użytkownika do grupy</label>
            <select v-model="userToAdd" class="input mb-2">
              <option value="">Wybierz użytkownika</option>
              <option v-for="user in getAvailableUsers()" :key="user.id" :value="user.id">
                {{ user.name }} ({{ user.email }})
              </option>
            </select>
            <button @click="addUserToGroup" :disabled="!userToAdd || addingUserToGroup" class="btn btn-primary btn-sm">
              {{ addingUserToGroup ? 'Dodawanie...' : 'Dodaj do grupy' }}
            </button>
          </div>

          <div v-if="getUsersInGroup(selectedGroup?.id).length > 0">
            <label class="block text-sm font-medium mb-2">Obecni członkowie</label>
            <div class="space-y-2">
              <div v-for="user in getUsersInGroup(selectedGroup?.id)" :key="user.id"
                   class="flex justify-between items-center p-3 bg-gray-700/30 rounded-lg">
                <div>
                  <p class="font-medium">{{ user.name }}</p>
                  <p class="text-sm text-gray-400">{{ user.email }}</p>
                </div>
                <button @click="removeUserFromGroup(user.id)" class="btn btn-sm btn-secondary">
                  <Trash class="w-3 h-3" />
                </button>
              </div>
            </div>
          </div>
          <div v-else class="text-center py-4 text-gray-400">
            Brak użytkowników w tej grupie
          </div>

          <div v-if="groupUserError" class="text-red-500 text-sm">{{ groupUserError }}</div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Role Modal -->
    <div v-if="showCreateRoleModal || showEditRoleModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showCreateRoleModal = false; showEditRoleModal = false">
      <div class="bg-gray-800 rounded-2xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-xl font-semibold">{{ showEditRoleModal ? 'Edytuj rolę' : 'Nowa rola' }}</h3>
          <button @click="showCreateRoleModal = false; showEditRoleModal = false" class="text-gray-400 hover:text-white">
            <X class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="showEditRoleModal ? updateRole() : createRole()" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Nazwa systemowa</label>
            <input v-model="roleForm.name" type="text" class="input" :disabled="showEditRoleModal" required />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Nazwa wyświetlana</label>
            <input v-model="roleForm.displayName" type="text" class="input" required />
          </div>
          <div>
            <label class="block text-sm font-medium mb-3">Uprawnienia</label>
            <div class="space-y-4 max-h-[300px] overflow-y-auto">
              <div v-for="(perms, category) in permissions" :key="category" class="p-3 bg-gray-700/30 rounded-lg">
                <h4 class="text-sm font-semibold mb-2 capitalize">{{ translateCategory(category) }}</h4>
                <div class="space-y-2">
                  <label v-for="perm in perms" :key="perm.name" class="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" :value="perm.name" v-model="roleForm.permissions" class="checkbox" />
                    <span class="text-sm">{{ translatePermission(perm.description) }}</span>
                  </label>
                </div>
              </div>
            </div>
          </div>
          <div class="flex gap-3">
            <button type="submit" class="btn btn-primary flex-1">
              {{ showEditRoleModal ? 'Zapisz' : 'Utwórz' }}
            </button>
            <button type="button" @click="showCreateRoleModal = false; showEditRoleModal = false" class="btn btn-outline">
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
import { UserPlus, Users, Edit, Trash, X, Key, Shield, ShieldOff, Download, Upload, RefreshCw, CheckSquare, Gauge, ShoppingCart, Search } from 'lucide-vue-next'

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
const showExportBackup = ref(false)
const showImportBackup = ref(false)

// Modals
const showCreateUserModal = ref(false)
const showEditUserModal = ref(false)
const showCreateGroupModal = ref(false)
const showManageGroupUsersModal = ref(false)

// Forms
const newUser = ref({
  name: '',
  email: '',
  password: '',
  role: 'RESIDENT'
})

const editUserForm = ref({
  id: '',
  name: '',
  email: '',
  role: ''
})

const groupForm = ref({
  name: '',
  weight: 1.0
})

const editingGroup = ref(null)
const selectedGroup = ref(null)
const userToAdd = ref('')
const addingUserToGroup = ref(false)
const creatingUser = ref(false)
const updatingUser = ref(false)
const savingGroup = ref(false)
const userError = ref('')
const groupError = ref('')
const groupUserError = ref('')

// Passkey state
const passkeySupported = ref(false)
const passkeys = ref([])
const loadingPasskeys = ref(false)
const addingPasskey = ref(false)
const showAddPasskeyModal = ref(false)
const passkeyName = ref('')
const passkeyError = ref('')
const passkeySuccess = ref('')

// Session state
const sessions = ref([])
const loadingSessions = ref(false)
const showRenameSessionModal = ref(false)
const renamingSession = ref(false)
const renameSessionForm = ref({
  id: '',
  name: ''
})

// Audit logs state
const auditLogs = ref([])
const loadingAuditLogs = ref(false)
const expandedAuditLogs = ref(new Set())
const auditFilters = ref({
  userEmail: '',
  action: '',
  status: '',
  search: '',
  dateFrom: '',
  dateTo: '',
  resourceType: ''
})
const auditLimit = ref(50)

// Role management state
const roles = ref([])
const loadingRoles = ref(false)
const permissions = ref({})
const showCreateRoleModal = ref(false)
const showEditRoleModal = ref(false)
const roleForm = ref({
  id: '',
  name: '',
  displayName: '',
  permissions: []
})

// Approval state
const pendingApprovals = ref([])
const loadingApprovals = ref(false)

onMounted(async () => {
  // Initialize profile form with current user data
  profileForm.value = {
    name: authStore.user?.name || '',
    email: authStore.user?.email || ''
  }

  if (authStore.isAdmin) {
    await Promise.all([loadUsers(), loadGroups(), loadRoles(), loadPermissions(), loadApprovals(), loadAuditLogs()])
  } else {
    // Load groups for regular users too so they can see group options
    await loadGroups()
  }

  // Check passkey support
  const support = await checkSupport()
  passkeySupported.value = support.supported

  // Load passkeys if supported
  if (passkeySupported.value) {
    await loadPasskeyList()
  }

  // Load sessions
  await loadSessions()
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
      oldPassword: passwordForm.value.currentPassword,
      newPassword: passwordForm.value.newPassword
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
      tempPassword: newUser.value.password,
      role: newUser.value.role
    })

    showCreateUserModal.value = false
    newUser.value = { name: '', email: '', password: '', role: 'RESIDENT' }
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
    role: user.role
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
      role: editUserForm.value.role
    })

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

function manageGroupUsers(group) {
  selectedGroup.value = group
  userToAdd.value = ''
  groupUserError.value = ''
  showManageGroupUsersModal.value = true
}

function getUsersInGroup(groupId) {
  if (!groupId) return []
  return users.value.filter(user => user.groupId === groupId)
}

function getAvailableUsers() {
  if (!selectedGroup.value) return []
  return users.value.filter(user => user.groupId !== selectedGroup.value.id)
}

async function addUserToGroup() {
  if (!userToAdd.value || !selectedGroup.value) return

  addingUserToGroup.value = true
  groupUserError.value = ''

  try {
    await api.patch(`/users/${userToAdd.value}`, {
      groupId: selectedGroup.value.id
    })

    await loadUsers()
    userToAdd.value = ''
    emit(DATA_EVENTS.USER_UPDATED, { userId: userToAdd.value })
  } catch (err) {
    groupUserError.value = err.response?.data?.error || 'Nie udało się dodać użytkownika do grupy'
  } finally {
    addingUserToGroup.value = false
  }
}

async function removeUserFromGroup(userId) {
  if (!confirm('Czy na pewno chcesz usunąć tego użytkownika z grupy?')) return

  groupUserError.value = ''

  try {
    await api.patch(`/users/${userId}`, {
      groupId: '000000000000000000000000' // Zero ObjectID to remove group
    })

    await loadUsers()
    emit(DATA_EVENTS.USER_UPDATED, { userId })
  } catch (err) {
    groupUserError.value = err.response?.data?.error || 'Nie udało się usunąć użytkownika z grupy'
  }
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

// Session functions
async function loadSessions() {
  loadingSessions.value = true
  try {
    const response = await api.get('/sessions')
    sessions.value = response.data || []
  } catch (err) {
    console.error('Failed to load sessions:', err)
  } finally {
    loadingSessions.value = false
  }
}

function isCurrentSession(session) {
  // Check if this is the current session by comparing last used time
  // The most recently used session is likely the current one
  if (sessions.value.length === 0) return false
  const mostRecent = sessions.value.reduce((prev, current) =>
    new Date(current.lastUsedAt) > new Date(prev.lastUsedAt) ? current : prev
  )
  return session.id === mostRecent.id
}

function openRenameSessionModal(session) {
  renameSessionForm.value = {
    id: session.id,
    name: session.name
  }
  showRenameSessionModal.value = true
}

async function confirmRenameSession() {
  renamingSession.value = true
  try {
    await api.patch(`/sessions/${renameSessionForm.value.id}`, {
      name: renameSessionForm.value.name
    })

    showRenameSessionModal.value = false
    await loadSessions()
  } catch (err) {
    console.error('Failed to rename session:', err)
    alert('Nie udało się zmienić nazwy sesji')
  } finally {
    renamingSession.value = false
  }
}

async function deleteSession(sessionId) {
  if (!confirm('Czy na pewno chcesz usunąć tę sesję? Zostaniesz wylogowany z tego urządzenia.')) return

  try {
    await api.delete(`/sessions/${sessionId}`)
    await loadSessions()
  } catch (err) {
    console.error('Failed to delete session:', err)
    alert('Nie udało się usunąć sesji')
  }
}

function formatDate(dateString) {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleString('pl-PL', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function translateCategory(category) {
  const translations = {
    'users': 'Użytkownicy',
    'groups': 'Grupy',
    'bills': 'Rachunki',
    'chores': 'Obowiązki',
    'supplies': 'Zaopatrzenie',
    'roles': 'Role',
    'approvals': 'Zatwierdzenia',
    'audit': 'Audyt',
    'loans': 'Pożyczki',
    'backup': 'Kopia zapasowa'
  }
  return translations[category] || category
}

function translatePermission(description) {
  const translations = {
    // Users
    'Create new users': 'Twórz nowych użytkowników',
    'View user information': 'Przeglądaj informacje o użytkownikach',
    'Update user information': 'Aktualizuj informacje o użytkownikach',
    'Delete users': 'Usuń użytkowników',

    // Groups
    'Create new groups': 'Twórz nowe grupy',
    'View groups': 'Przeglądaj grupy',
    'Update groups': 'Aktualizuj grupy',
    'Delete groups': 'Usuń grupy',

    // Bills
    'Create new bills': 'Twórz nowe rachunki',
    'View bills': 'Przeglądaj rachunki',
    'Update bills': 'Aktualizuj rachunki',
    'Delete bills': 'Usuń rachunki',
    'Post bills (freeze allocations)': 'Opublikuj rachunki',
    'Close bills': 'Zamknij rachunki',

    // Chores
    'Create new chores': 'Twórz nowe obowiązki',
    'View chores': 'Przeglądaj obowiązki',
    'Update chores': 'Aktualizuj obowiązki',
    'Delete chores': 'Usuń obowiązki',
    'Assign chores to users': 'Przypisz obowiązki do użytkowników',

    // Supplies
    'Add supply items': 'Dodaj artykuły zaopatrzeniowe',
    'View supplies': 'Przeglądaj zaopatrzenie',
    'Update supply items': 'Aktualizuj artykuły zaopatrzeniowe',
    'Delete supply items': 'Usuń artykuły zaopatrzeniowe',

    // Roles
    'Create custom roles': 'Twórz niestandardowe role',
    'View roles': 'Przeglądaj role',
    'Update roles': 'Aktualizuj role',
    'Delete roles': 'Usuń role',

    // Approvals
    'Review and approve/reject pending actions': 'Przeglądaj i zatwierdź/odrzuć oczekujące akcje',

    // Audit
    'View audit logs': 'Przeglądaj logi audytu'
  }
  return translations[description] || description
}

function translatePermissionName(name) {
  const translations = {
    'users.create': 'Twórz użytkowników',
    'users.read': 'Czytaj użytkowników',
    'users.update': 'Edytuj użytkowników',
    'users.delete': 'Usuń użytkowników',
    'groups.create': 'Twórz grupy',
    'groups.read': 'Czytaj grupy',
    'groups.update': 'Edytuj grupy',
    'groups.delete': 'Usuń grupy',
    'bills.create': 'Twórz rachunki',
    'bills.read': 'Czytaj rachunki',
    'bills.update': 'Edytuj rachunki',
    'bills.delete': 'Usuń rachunki',
    'bills.post': 'Opublikuj rachunki',
    'bills.close': 'Zamknij rachunki',
    'chores.create': 'Twórz obowiązki',
    'chores.read': 'Czytaj obowiązki',
    'chores.update': 'Edytuj obowiązki',
    'chores.delete': 'Usuń obowiązki',
    'chores.assign': 'Przypisz obowiązki',
    'supplies.create': 'Twórz zaopatrzenie',
    'supplies.read': 'Czytaj zaopatrzenie',
    'supplies.update': 'Edytuj zaopatrzenie',
    'supplies.delete': 'Usuń zaopatrzenie',
    'roles.create': 'Twórz role',
    'roles.read': 'Czytaj role',
    'roles.update': 'Edytuj role',
    'roles.delete': 'Usuń role',
    'approvals.review': 'Sprawdzaj zatwierdzenia',
    'audit.read': 'Czytaj logi audytu',
    'loans.create': 'Twórz pożyczki',
    'loans.read': 'Czytaj pożyczki',
    'loans.update': 'Edytuj pożyczki',
    'loans.delete': 'Usuń pożyczki',
    'backup.export': 'Eksportuj kopię zapasową',
    'backup.import': 'Importuj kopię zapasową'
  }
  return translations[name] || name
}

function translateAction(action) {
  const translations = {
    // Role actions
    'role.create': 'Tworzenie roli',
    'role.update': 'Aktualizacja roli',
    'role.delete': 'Usunięcie roli',

    // User actions
    'user.create': 'Tworzenie użytkownika',
    'user.update': 'Aktualizacja użytkownika',
    'user.delete': 'Usunięcie użytkownika',

    // Bill actions
    'create_bill': 'Tworzenie rachunku',
    'bill.create': 'Tworzenie rachunku',
    'bill.update': 'Aktualizacja rachunku',
    'bill.delete': 'Usunięcie rachunku',
    'post_bill': 'Opublikowanie rachunku',
    'close_bill': 'Zamknięcie rachunku',

    // Reading/Consumption actions
    'create_reading': 'Dodanie odczytu',
    'reading.create': 'Dodanie odczytu',

    // Chore actions
    'chore.create': 'Tworzenie obowiązku',
    'chore.update': 'Aktualizacja obowiązku',
    'chore.delete': 'Usunięcie obowiązku',

    // Supply actions
    'supply.create': 'Tworzenie zaopatrzenia',
    'supply.update': 'Aktualizacja zaopatrzenia',
    'supply.delete': 'Usunięcie zaopatrzenia',

    // Group actions
    'group.create': 'Tworzenie grupy',
    'group.update': 'Aktualizacja grupy',
    'group.delete': 'Usunięcie grupy',

    // Loan actions
    'loan.create': 'Tworzenie pożyczki',
    'loan.update': 'Aktualizacja pożyczki',
    'loan.delete': 'Usunięcie pożyczki'
  }
  return translations[action] || action
}

// Audit logs functions
let auditDebounceTimer = null

function debouncedLoadAuditLogs() {
  if (auditDebounceTimer) clearTimeout(auditDebounceTimer)
  auditDebounceTimer = setTimeout(() => {
    loadAuditLogs()
  }, 400)
}

async function loadAuditLogs() {
  loadingAuditLogs.value = true
  try {
    const params = new URLSearchParams({ limit: auditLimit.value.toString() })
    if (auditFilters.value.userEmail) params.append('userEmail', auditFilters.value.userEmail)
    if (auditFilters.value.action) params.append('action', auditFilters.value.action)
    if (auditFilters.value.status) params.append('status', auditFilters.value.status)
    if (auditFilters.value.search) params.append('search', auditFilters.value.search)
    if (auditFilters.value.dateFrom) params.append('dateFrom', auditFilters.value.dateFrom)
    if (auditFilters.value.dateTo) params.append('dateTo', auditFilters.value.dateTo)
    if (auditFilters.value.resourceType) params.append('resourceType', auditFilters.value.resourceType)

    const response = await api.get(`/audit/logs?${params.toString()}`)
    auditLogs.value = response.data.logs || []
  } catch (err) {
    console.error('Failed to load audit logs:', err)
  } finally {
    loadingAuditLogs.value = false
  }
}

function clearAuditFilters() {
  auditFilters.value = {
    userEmail: '',
    action: '',
    status: '',
    search: '',
    dateFrom: '',
    dateTo: '',
    resourceType: ''
  }
  loadAuditLogs()
}

function toggleAuditLogDetails(logId) {
  if (expandedAuditLogs.value.has(logId)) {
    expandedAuditLogs.value.delete(logId)
  } else {
    expandedAuditLogs.value.add(logId)
  }
}

function formatAuditDetails(log) {
  if (!log.details) return {}

  const details = log.details
  const formatted = {}

  // Handle permission changes specially
  if (details.oldPermissions && details.newPermissions) {
    const oldPerms = new Set(details.oldPermissions)
    const newPerms = new Set(details.newPermissions)

    const added = [...newPerms].filter(p => !oldPerms.has(p))
    const removed = [...oldPerms].filter(p => !newPerms.has(p))
    const unchanged = [...oldPerms].filter(p => newPerms.has(p))

    if (added.length > 0) {
      formatted['Dodane uprawnienia'] = added.map(p => '• ' + translatePermissionName(p)).join('\n')
    }
    if (removed.length > 0) {
      formatted['Usunięte uprawnienia'] = removed.map(p => '• ' + translatePermissionName(p)).join('\n')
    }
    if (unchanged.length > 0) {
      formatted['Niezmienione uprawnienia'] = `${unchanged.length} uprawnień`
    }

    // Skip showing raw oldPermissions/newPermissions
    const keysToSkip = ['oldPermissions', 'newPermissions']
    for (const [key, value] of Object.entries(details)) {
      if (!keysToSkip.includes(key)) {
        formatted[translateDetailKey(key)] = value
      }
    }
  }
  // Handle 'changes' object (flatten it)
  else if (details.changes && typeof details.changes === 'object') {
    formatted['Zmienione pola'] = Object.entries(details.changes)
      .map(([k, v]) => `${translateDetailKey(k)}: ${v}`)
      .join('\n')

    // Show other fields
    for (const [key, value] of Object.entries(details)) {
      if (key !== 'changes') {
        if (Array.isArray(value)) {
          formatted[translateDetailKey(key)] = value.join(', ')
        } else if (typeof value === 'object' && value !== null) {
          formatted[translateDetailKey(key)] = JSON.stringify(value, null, 2)
        } else {
          formatted[translateDetailKey(key)] = value
        }
      }
    }
  }
  // Default formatting for all other cases
  else {
    for (const [key, value] of Object.entries(details)) {
      const translatedKey = translateDetailKey(key)

      if (Array.isArray(value)) {
        // Check if it's a permission array
        if (key.includes('ermission') || key.includes('Permissions')) {
          formatted[translatedKey] = value.map(p => '• ' + translatePermissionName(p)).join('\n')
        } else {
          formatted[translatedKey] = value.join(', ')
        }
      } else if (typeof value === 'object' && value !== null) {
        formatted[translatedKey] = JSON.stringify(value, null, 2)
      } else {
        formatted[translatedKey] = value
      }
    }
  }

  return formatted
}

function getResourceRoute(log) {
  if (!log.resourceId || log.status !== 'success') return null

  const routes = {
    'bill': '/bills',
    'chore': '/chores',
    'user': '/settings',
    'role': '/settings',
    'group': '/settings'
  }

  return routes[log.resourceType] || null
}

function getAuditLogTitle(log) {
  const action = translateAction(log.action)

  // Try to extract entity name from details
  let entityName = ''

  if (log.details) {
    // For user updates - prefer targetUser over userName
    if (log.details.targetUser) {
      entityName = log.details.targetUser
    } else if (log.details.userName) {
      entityName = log.details.userName
    } else if (log.details.userEmail) {
      entityName = log.details.userEmail
    }
    // For role updates
    else if (log.details.roleName) {
      entityName = log.details.roleName
    } else if (log.details.newDisplayName) {
      entityName = log.details.newDisplayName
    } else if (log.details.displayName) {
      entityName = log.details.displayName
    }
    // For bills - handle both billType and type, and bill_type
    else if (log.details.bill_type) {
      entityName = translateBillType(log.details.bill_type)
    } else if (log.details.billType) {
      entityName = translateBillType(log.details.billType)
    } else if (log.details.type) {
      entityName = translateBillType(log.details.type)
    }
    // For chores
    else if (log.details.choreName) {
      entityName = log.details.choreName
    } else if (log.details.name && log.resourceType === 'chore') {
      entityName = log.details.name
    }
    // For groups
    else if (log.details.groupName) {
      entityName = log.details.groupName
    } else if (log.details.name && log.resourceType === 'group') {
      entityName = log.details.name
    }
  }

  return entityName ? `${action}: ${entityName}` : action
}

function translateBillType(type) {
  const translations = {
    'electricity': 'Prąd',
    'gas': 'Gaz',
    'internet': 'Internet',
    'water': 'Woda',
    'inne': 'Inne'
  }
  return translations[type] || type
}

function getAuditLogSummary(log) {
  if (!log.details) return ''

  const details = log.details
  const summaries = []

  // Role permission changes - show exact permissions added/removed
  if (details.oldPermissions && details.newPermissions) {
    const oldPerms = new Set(details.oldPermissions)
    const newPerms = new Set(details.newPermissions)

    const added = [...newPerms].filter(p => !oldPerms.has(p))
    const removed = [...oldPerms].filter(p => !newPerms.has(p))

    if (added.length > 0) {
      const addedNames = added.slice(0, 2).map(p => translatePermissionName(p))
      const addedSummary = addedNames.join(', ')
      summaries.push(`Dodano: ${addedSummary}${added.length > 2 ? ` i ${added.length - 2} więcej` : ''}`)
    }
    if (removed.length > 0) {
      const removedNames = removed.slice(0, 2).map(p => translatePermissionName(p))
      const removedSummary = removedNames.join(', ')
      summaries.push(`Usunięto: ${removedSummary}${removed.length > 2 ? ` i ${removed.length - 2} więcej` : ''}`)
    }
  }

  // Display name changes
  if (details.oldDisplayName && details.newDisplayName && details.oldDisplayName !== details.newDisplayName) {
    summaries.push(`Nazwa: ${details.oldDisplayName} → ${details.newDisplayName}`)
  }

  // User role changes
  if (details.oldRole && details.newRole) {
    summaries.push(`Rola: ${details.oldRole} → ${details.newRole}`)
  } else if (details.newRole) {
    summaries.push(`Nowa rola: ${details.newRole}`)
  }

  // Target user for user updates
  if (details.targetUser && !details.newRole && !details.oldRole) {
    summaries.push(`Użytkownik: ${details.targetUser}`)
  }

  // Handle 'changes' object from group/supply updates
  if (details.changes && typeof details.changes === 'object') {
    for (const [key, value] of Object.entries(details.changes)) {
      if (key === 'name') {
        summaries.push(`Nazwa: ${value}`)
      } else if (key === 'weight') {
        summaries.push(`Waga: ${value}`)
      } else if (key === 'category') {
        summaries.push(`Kategoria: ${value}`)
      } else if (key === 'minQuantity') {
        summaries.push(`Min. ilość: ${value}`)
      }
    }
  }

  // Bill amount
  if (details.amount && !details.old_status) {
    summaries.push(`Kwota: ${details.amount} PLN`)
  }

  // Bill type
  if (details.type && !details.amount) {
    summaries.push(`Typ: ${translateBillType(details.type)}`)
  } else if (details.bill_type && !details.old_status) {
    summaries.push(`Typ: ${translateBillType(details.bill_type)}`)
  }

  // Status changes (use old_status/new_status as per backend)
  if (details.old_status && details.new_status) {
    summaries.push(`Status: ${details.old_status} → ${details.new_status}`)
  }

  // Meter readings
  if (details.meter_value) {
    summaries.push(`Licznik: ${details.meter_value}`)
  }

  // Supply quantity changes
  if (details.quantity && log.action.includes('supply')) {
    summaries.push(`Ilość: ${details.quantity}`)
  }

  // Period for bills
  if (details.period_start && details.period_end) {
    summaries.push(`Okres: ${details.period_start} - ${details.period_end}`)
  }

  // Limit to first 3 items for summary
  if (summaries.length > 3) {
    return summaries.slice(0, 3).join(' • ') + ' i więcej...'
  }

  return summaries.join(' • ')
}

function translateResourceType(resourceType) {
  const translations = {
    'bill': 'rachunek',
    'chore': 'obowiązek',
    'user': 'użytkownika',
    'role': 'rolę',
    'group': 'grupę',
    'supply': 'zaopatrzenie',
    'loan': 'pożyczkę'
  }
  return translations[resourceType] || resourceType
}

function translateDetailKey(key) {
  const translations = {
    // User fields
    'userName': 'Nazwa użytkownika',
    'userEmail': 'Email użytkownika',
    'targetUser': 'Zmieniony użytkownik',
    'oldRole': 'Poprzednia rola',
    'newRole': 'Nowa rola',
    'userId': 'ID użytkownika',

    // Role fields
    'roleName': 'Nazwa roli',
    'displayName': 'Nazwa wyświetlana',
    'oldDisplayName': 'Poprzednia nazwa',
    'newDisplayName': 'Nowa nazwa',
    'oldPermissions': 'Poprzednie uprawnienia',
    'newPermissions': 'Nowe uprawnienia',
    'permissions': 'Uprawnienia',

    // Bill fields
    'billType': 'Typ rachunku',
    'bill_type': 'Typ rachunku',
    'type': 'Typ',
    'amount': 'Kwota (PLN)',
    'period': 'Okres',
    'period_start': 'Początek okresu',
    'period_end': 'Koniec okresu',
    'oldStatus': 'Poprzedni status',
    'newStatus': 'Nowy status',
    'old_status': 'Poprzedni status',
    'new_status': 'Nowy status',
    'status': 'Status',

    // Consumption/Reading fields
    'bill_id': 'ID rachunku',
    'meter_value': 'Wartość licznika',
    'source': 'Źródło',

    // Chore fields
    'choreName': 'Nazwa obowiązku',
    'name': 'Nazwa',
    'description': 'Opis',
    'assignee': 'Przypisany do',
    'dueDate': 'Termin',

    // Group fields
    'groupName': 'Nazwa grupy',
    'weight': 'Waga',

    // Supply fields
    'itemName': 'Nazwa pozycji',
    'quantity': 'Ilość',
    'unit': 'Jednostka',

    // Common fields
    'createdAt': 'Utworzono',
    'updatedAt': 'Zaktualizowano',
    'deletedAt': 'Usunięto',
    'id': 'ID',
    'error': 'Błąd'
  }

  return translations[key] || key
}

// Role management functions
async function loadRoles() {
  loadingRoles.value = true
  try {
    const response = await api.get('/roles')
    roles.value = response.data || []
  } catch (err) {
    console.error('Failed to load roles:', err)
  } finally {
    loadingRoles.value = false
  }
}

async function loadPermissions() {
  try {
    const response = await api.get('/permissions')
    permissions.value = response.data || {}
  } catch (err) {
    console.error('Failed to load permissions:', err)
  }
}

function openEditRoleModal(role) {
  roleForm.value = {
    id: role.id,
    name: role.name,
    displayName: role.displayName,
    permissions: [...role.permissions]
  }
  showEditRoleModal.value = true
}

async function createRole() {
  try {
    await api.post('/roles', {
      name: roleForm.value.name,
      displayName: roleForm.value.displayName,
      permissions: roleForm.value.permissions
    })
    showCreateRoleModal.value = false
    roleForm.value = { id: '', name: '', displayName: '', permissions: [] }
    await loadRoles()
    alert('Rola utworzona pomyślnie')
  } catch (err) {
    alert('Nie udało się utworzyć roli: ' + (err.response?.data?.error || err.message))
  }
}

async function updateRole() {
  try {
    await api.patch(`/roles/${roleForm.value.id}`, {
      displayName: roleForm.value.displayName,
      permissions: roleForm.value.permissions
    })
    showEditRoleModal.value = false
    roleForm.value = { id: '', name: '', displayName: '', permissions: [] }
    await loadRoles()
    alert('Rola zaktualizowana pomyślnie')
  } catch (err) {
    alert('Nie udało się zaktualizować roli: ' + (err.response?.data?.error || err.message))
  }
}

async function deleteRole(roleId) {
  if (!confirm('Czy na pewno chcesz usunąć tę rolę?')) return
  try {
    await api.delete(`/roles/${roleId}`)
    await loadRoles()
    alert('Rola usunięta pomyślnie')
  } catch (err) {
    alert('Nie udało się usunąć roli: ' + (err.response?.data?.error || err.message))
  }
}

// Approval functions
async function loadApprovals() {
  loadingApprovals.value = true
  try {
    const response = await api.get('/approvals/pending')
    pendingApprovals.value = response.data || []
  } catch (err) {
    console.error('Failed to load approvals:', err)
  } finally {
    loadingApprovals.value = false
  }
}

async function approveRequest(requestId) {
  try {
    await api.post(`/approvals/${requestId}/approve`)
    await loadApprovals()
    alert('Wniosek zatwierdzony')
  } catch (err) {
    alert('Nie udało się zatwierdzić wniosku: ' + (err.response?.data?.error || err.message))
  }
}

async function rejectRequest(requestId) {
  try {
    await api.post(`/approvals/${requestId}/reject`)
    await loadApprovals()
    alert('Wniosek odrzucony')
  } catch (err) {
    alert('Nie udało się odrzucić wniosku: ' + (err.response?.data?.error || err.message))
  }
}

// Helper function to get role display name
function getUserRoleDisplayName(roleName) {
  if (!roleName) return ''
  const role = roles.value.find(r => r.name === roleName)
  return role ? role.displayName : roleName
}
</script>