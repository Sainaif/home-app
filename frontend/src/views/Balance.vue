<template>
  <div>
    <h1 class="text-3xl font-bold mb-8">{{ $t('balance.title') }}</h1>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div class="card">
        <h2 class="text-xl font-semibold mb-4 text-red-400">{{ $t('balance.youOwe') }}</h2>
        <div v-if="loading" class="text-center py-8">{{ $t('common.loading') }}</div>
        <div v-else-if="youOwe.length === 0" class="text-center py-8 text-gray-400">{{ $t('balance.settled') }}</div>
        <div v-else class="space-y-3">
          <div v-for="bal in youOwe" :key="`${bal.fromUserId}-${bal.toUserId}`"
               class="flex justify-between items-center p-3 bg-gray-700 rounded">
            <span>
              {{ bal.toUserName }}
              <span v-if="bal.toUserGroupName" class="text-xs text-purple-400 ml-1">({{ bal.toUserGroupName }})</span>
            </span>
            <span class="font-bold text-red-400">{{ formatMoney(bal.netAmount) }} PLN</span>
          </div>
        </div>
      </div>

      <div class="card">
        <h2 class="text-xl font-semibold mb-4 text-green-400">{{ $t('balance.owesYou') }}</h2>
        <div v-if="loading" class="text-center py-8">{{ $t('common.loading') }}</div>
        <div v-else-if="owesYou.length === 0" class="text-center py-8 text-gray-400">{{ $t('balance.settled') }}</div>
        <div v-else class="space-y-3">
          <div v-for="bal in owesYou" :key="`${bal.fromUserId}-${bal.toUserId}`"
               class="flex justify-between items-center p-3 bg-gray-700 rounded">
            <span>
              {{ bal.fromUserName }}
              <span v-if="bal.fromUserGroupName" class="text-xs text-purple-400 ml-1">({{ bal.fromUserGroupName }})</span>
            </span>
            <span class="font-bold text-green-400">{{ formatMoney(bal.netAmount) }} PLN</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Loan/Payment Forms (Permission-based) -->
    <div v-if="authStore.hasPermission('loans.create') || authStore.hasPermission('loan-payments.create')" class="grid grid-cols-1 md:grid-cols-2 gap-6 mt-6">
      <div v-if="authStore.hasPermission('loans.create')" class="card">
        <h2 class="text-xl font-semibold mb-4">Dodaj pożyczkę</h2>
        <form @submit.prevent="createLoan" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Pożyczkodawca</label>
            <select v-model="loanForm.lenderId" required class="input">
              <option value="">Wybierz użytkownika</option>
              <option v-for="user in users" :key="user.id" :value="user.id">{{ user.name }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Pożyczkobiorca</label>
            <select v-model="loanForm.borrowerId" required class="input">
              <option value="">Wybierz użytkownika</option>
              <option v-for="user in users" :key="user.id" :value="user.id">{{ user.name }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Kwota (PLN)</label>
            <input v-model.number="loanForm.amount" type="number" step="0.01" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Notatka (opcjonalnie)</label>
            <input v-model="loanForm.note" type="text" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Termin spłaty (opcjonalnie)</label>
            <input v-model="loanForm.dueDate" type="date" class="input" />
          </div>
          <button type="submit" :disabled="creatingLoan" class="btn btn-primary w-full">
            {{ creatingLoan ? 'Dodawanie...' : 'Dodaj pożyczkę' }}
          </button>
        </form>
      </div>

      <div v-if="authStore.hasPermission('loan-payments.create')" class="card">
        <h2 class="text-xl font-semibold mb-4">Dodaj spłatę</h2>
        <form @submit.prevent="createPayment" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Pożyczka</label>
            <select v-model="paymentForm.loanId" required class="input">
              <option value="">Wybierz pożyczkę</option>
              <option v-for="loan in openLoans" :key="loan.id" :value="loan.id">
                {{ getLoanDescription(loan) }}
              </option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Kwota (PLN)</label>
            <input v-model.number="paymentForm.amount" type="number" step="0.01" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Data spłaty</label>
            <input v-model="paymentForm.paidAt" type="datetime-local" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Notatka (opcjonalnie)</label>
            <input v-model="paymentForm.note" type="text" class="input" />
          </div>
          <button type="submit" :disabled="creatingPayment" class="btn btn-primary w-full">
            {{ creatingPayment ? 'Dodawanie...' : 'Dodaj spłatę' }}
          </button>
        </form>
      </div>
    </div>

    <div class="card mt-6">
      <div class="space-y-3 mb-4">
        <h2 class="text-xl font-semibold">Historia pożyczek</h2>

        <!-- Filter by user -->
        <div>
          <label class="block text-sm font-medium mb-2">Filtruj po użytkowniku</label>
          <select v-model="userFilter" class="input mb-3">
            <option value="">Wszyscy użytkownicy</option>
            <option v-for="user in users" :key="user.id" :value="user.id">
              {{ user.name }}
            </option>
          </select>
        </div>

        <!-- Status filter -->
        <div class="button-group grid-cols-4">
          <button
            @click="statusFilter = 'all'"
            :class="statusFilter === 'all' ? 'btn-primary' : 'btn-outline'"
            class="btn btn-sm">
            Wszystkie
          </button>
          <button
            @click="statusFilter = 'open'"
            :class="statusFilter === 'open' ? 'btn-primary' : 'btn-outline'"
            class="btn btn-sm">
            Niespłacone
          </button>
          <button
            @click="statusFilter = 'partial'"
            :class="statusFilter === 'partial' ? 'btn-primary' : 'btn-outline'"
            class="btn btn-sm">
            Częściowo
          </button>
          <button
            @click="statusFilter = 'settled'"
            :class="statusFilter === 'settled' ? 'btn-primary' : 'btn-outline'"
            class="btn btn-sm">
            Spłacone
          </button>
        </div>

        <!-- Sorting and pagination controls -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div>
            <label class="block text-sm font-medium mb-2">Sortuj według</label>
            <select v-model="sortBy" @change="loadLoans" class="input">
              <option value="createdAt">Data utworzenia</option>
              <option value="amountPLN">Kwota</option>
              <option value="status">Status</option>
              <option value="remainingPLN">Pozostała kwota</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Kolejność</label>
            <select v-model="sortOrder" @change="loadLoans" class="input">
              <option value="desc">Malejąco</option>
              <option value="asc">Rosnąco</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Wyświetl</label>
            <select v-model="pageLimit" @change="loadLoans" class="input">
              <option :value="10">10</option>
              <option :value="25">25</option>
              <option :value="50">50</option>
              <option :value="100">100</option>
              <option :value="0">Wszystkie</option>
            </select>
          </div>
        </div>
      </div>
      <div v-if="loading" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else-if="filteredLoans.length === 0" class="text-center py-8 text-gray-400">Brak historii</div>
      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead class="border-b border-gray-700">
            <tr class="text-left">
              <th class="pb-3">Od</th>
              <th class="pb-3">Do</th>
              <th class="pb-3">Kwota</th>
              <th class="pb-3">Pozostało</th>
              <th class="pb-3">Opis</th>
              <th class="pb-3">Status</th>
              <th class="pb-3">Data</th>
              <th v-if="authStore.hasPermission('loans.delete')" class="pb-3">Akcje</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="loan in filteredLoans" :key="loan.id" class="border-b border-gray-700">
              <td class="py-3">
                {{ loan.fromUserName }}
                <span v-if="loan.fromUserGroupName" class="text-xs text-purple-400 ml-1">({{ loan.fromUserGroupName }})</span>
              </td>
              <td class="py-3">
                {{ loan.toUserName }}
                <span v-if="loan.toUserGroupName" class="text-xs text-purple-400 ml-1">({{ loan.toUserGroupName }})</span>
              </td>
              <td class="py-3">{{ formatMoney(loan.amountPLN) }} PLN</td>
              <td class="py-3" :class="getRemainingColorClass(loan)">
                {{ formatMoney(loan.remainingPLN) }} PLN
              </td>
              <td class="py-3 text-gray-400">{{ loan.note || '-' }}</td>
              <td class="py-3">
                <span :class="getStatusColorClass(loan.status)">
                  {{ translateStatus(loan.status) }}
                </span>
              </td>
              <td class="py-3">{{ formatDate(loan.createdAt) }}</td>
              <td v-if="authStore.hasPermission('loans.delete')" class="py-3">
                <button
                  @click="confirmDeleteLoan(loan.id)"
                  class="btn btn-sm btn-secondary"
                  title="Usuń pożyczkę">
                  <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M3 6h18"/>
                    <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
                    <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                  </svg>
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
// @version 2.0.0 - Fixed array filter checks
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useEventStream } from '../composables/useEventStream'
import api from '../api/client'

const authStore = useAuthStore()
const balances = ref([])
const loans = ref([])
const users = ref([])
const loading = ref(false)
const creatingLoan = ref(false)
const creatingPayment = ref(false)
const statusFilter = ref('all')
const userFilter = ref('')
const sortBy = ref('createdAt')
const sortOrder = ref('desc')
const pageLimit = ref(50)

const loanForm = ref({
  lenderId: '',
  borrowerId: '',
  amount: '',
  note: '',
  dueDate: ''
})

const paymentForm = ref({
  loanId: '',
  amount: '',
  paidAt: new Date().toISOString().slice(0, 16),
  note: ''
})

const youOwe = computed(() =>
  Array.isArray(balances.value) ? balances.value.filter(b => b.fromUserId === authStore.user?.id) : []
)

const owesYou = computed(() =>
  Array.isArray(balances.value) ? balances.value.filter(b => b.toUserId === authStore.user?.id) : []
)

const openLoans = computed(() =>
  Array.isArray(loans.value) ? loans.value.filter(l => l.status === 'open' || l.status === 'partial') : []
)

const filteredLoans = computed(() => {
  if (!Array.isArray(loans.value)) return []

  let result = loans.value

  // Filter by status
  if (statusFilter.value !== 'all') {
    result = result.filter(l => l.status === statusFilter.value)
  }

  // Filter by user (either lender or borrower)
  if (userFilter.value) {
    result = result.filter(l =>
      l.lenderId === userFilter.value || l.borrowerId === userFilter.value
    )
  }

  return result
})

async function loadLoans() {
  try {
    const params = new URLSearchParams({
      sort: sortBy.value,
      order: sortOrder.value,
    })

    if (pageLimit.value > 0) {
      params.append('limit', pageLimit.value.toString())
      params.append('offset', '0')
    }

    const response = await api.get(`/loans?${params.toString()}`)
    loans.value = response.data || []
  } catch (err) {
    console.error('Failed to load loans:', err)
    loans.value = []
  }
}

async function loadData() {
  try {
    const requests = [
      api.get('/loans/balances/me'),
      loadLoans()
    ]

    if (authStore.hasPermission('loans.create')) {
      requests.push(api.get('/users'))
    }

    const responses = await Promise.all(requests)
    balances.value = responses[0].data || []
    if (authStore.hasPermission('loans.create')) {
      users.value = responses[2].data || []
    }
  } catch (err) {
    console.error('Failed to load balance data:', err)
    balances.value = []
    loans.value = []
  }
}

const eventStream = useEventStream()

// Reload data when tab becomes visible
const handleVisibilityChange = () => {
  if (document.visibilityState === 'visible') {
    console.log('[Balance] Tab visible, reloading data')
    loadData()
  }
}

onMounted(async () => {
  loading.value = true
  await loadData()
  loading.value = false

  // Connect to SSE
  eventStream.connect()

  // Listen for loan-related events
  eventStream.on('loan.created', () => {
    console.log('[Balance] Loan created event received, reloading data')
    loadData()
  })

  eventStream.on('loan.payment.created', () => {
    console.log('[Balance] Loan payment event received, reloading data')
    loadData()
  })

  eventStream.on('loan.deleted', () => {
    console.log('[Balance] Loan deleted event received, reloading data')
    loadData()
  })

  eventStream.on('balance.updated', () => {
    console.log('[Balance] Balance updated event received, reloading data')
    loadData()
  })

  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onUnmounted(() => {
  // Cleanup SSE listeners
  eventStream.off('loan.created', loadData)
  eventStream.off('loan.payment.created', loadData)
  eventStream.off('loan.deleted', loadData)
  eventStream.off('balance.updated', loadData)
  eventStream.disconnect()

  document.removeEventListener('visibilitychange', handleVisibilityChange)
})

async function createLoan() {
  creatingLoan.value = true
  try {
    await api.post('/loans', {
      lenderId: loanForm.value.lenderId,
      borrowerId: loanForm.value.borrowerId,
      amountPLN: loanForm.value.amount,
      note: loanForm.value.note || undefined,
      dueDate: loanForm.value.dueDate ? new Date(loanForm.value.dueDate).toISOString() : undefined
    })

    // Reset form
    loanForm.value = { lenderId: '', borrowerId: '', amount: '', note: '', dueDate: '' }

    // Reload data
    await loadData()
  } catch (err) {
    console.error('Failed to create loan:', err)
    alert('Błąd tworzenia pożyczki: ' + (err.response?.data?.error || err.message))
  } finally {
    creatingLoan.value = false
  }
}

async function createPayment() {
  creatingPayment.value = true
  try {
    await api.post('/loan-payments', {
      loanId: paymentForm.value.loanId,
      amountPLN: paymentForm.value.amount,
      paidAt: new Date(paymentForm.value.paidAt).toISOString(),
      note: paymentForm.value.note || undefined
    })

    // Reset form
    paymentForm.value = {
      loanId: '',
      amount: '',
      paidAt: new Date().toISOString().slice(0, 16),
      note: ''
    }

    // Reload data
    await loadData()
  } catch (err) {
    console.error('Failed to create payment:', err)
    alert('Błąd tworzenia spłaty: ' + (err.response?.data?.error || err.message))
  } finally {
    creatingPayment.value = false
  }
}

function getLoanDescription(loan) {
  const lender = users.value.find(u => u.id === loan.lenderId)
  const borrower = users.value.find(u => u.id === loan.borrowerId)
  const totalAmount = formatMoney(loan.amountPLN)
  const remaining = loan.remainingPLN ? formatMoney(loan.remainingPLN) : totalAmount

  if (loan.status === 'partial') {
    return `${lender?.name || '?'} → ${borrower?.name || '?'} (pozostało: ${remaining} PLN z ${totalAmount} PLN)`
  }
  return `${lender?.name || '?'} → ${borrower?.name || '?'} (${totalAmount} PLN)`
}

function formatMoney(decimal128) {
  if (!decimal128) return '0.00'
  if (typeof decimal128 === 'number') return decimal128.toFixed(2)
  return parseFloat(decimal128.$numberDecimal || decimal128 || 0).toFixed(2)
}

function formatDate(date) {
  return new Date(date).toLocaleDateString('pl-PL')
}

function translateStatus(status) {
  const translations = {
    'open': 'Niespłacona',
    'partial': 'Częściowo spłacona',
    'settled': 'Spłacona'
  }
  return translations[status] || status
}

function getStatusColorClass(status) {
  const colors = {
    'open': 'text-red-400',
    'partial': 'text-yellow-400',
    'settled': 'text-green-400'
  }
  return colors[status] || 'text-gray-400'
}

function getRemainingColorClass(loan) {
  if (loan.status === 'settled') return 'text-green-400'
  if (loan.status === 'partial') return 'text-yellow-400'
  return 'text-red-400'
}

async function confirmDeleteLoan(loanId) {
  if (!confirm('Czy na pewno chcesz usunąć tę pożyczkę?')) return

  try {
    await api.delete(`/loans/${loanId}`)

    // Reload data
    await loadData()
  } catch (err) {
    console.error('Failed to delete loan:', err)
    alert('Błąd usuwania pożyczki: ' + (err.response?.data?.error || err.message))
  }
}
</script>