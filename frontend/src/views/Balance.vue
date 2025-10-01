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
            <span>{{ bal.toUserName }}</span>
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
            <span>{{ bal.fromUserName }}</span>
            <span class="font-bold text-green-400">{{ formatMoney(bal.netAmount) }} PLN</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Loan/Payment Forms (Admin Only) -->
    <div v-if="authStore.isAdmin" class="grid grid-cols-1 md:grid-cols-2 gap-6 mt-6">
      <div class="card">
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

      <div class="card">
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
      <h2 class="text-xl font-semibold mb-4">Historia pożyczek</h2>
      <div v-if="loading" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else-if="loans.length === 0" class="text-center py-8 text-gray-400">Brak historii</div>
      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead class="border-b border-gray-700">
            <tr class="text-left">
              <th class="pb-3">Od</th>
              <th class="pb-3">Do</th>
              <th class="pb-3">Kwota</th>
              <th class="pb-3">Opis</th>
              <th class="pb-3">Status</th>
              <th class="pb-3">Data</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="loan in loans" :key="loan.id" class="border-b border-gray-700">
              <td class="py-3">{{ loan.fromUserName }}</td>
              <td class="py-3">{{ loan.toUserName }}</td>
              <td class="py-3">{{ formatMoney(loan.amountPLN) }} PLN</td>
              <td class="py-3 text-gray-400">{{ loan.note || '-' }}</td>
              <td class="py-3">{{ loan.status }}</td>
              <td class="py-3">{{ formatDate(loan.createdAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
// @version 2.0.0 - Fixed array filter checks
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'

const authStore = useAuthStore()
const balances = ref([])
const loans = ref([])
const users = ref([])
const loading = ref(false)
const creatingLoan = ref(false)
const creatingPayment = ref(false)

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
  Array.isArray(loans.value) ? loans.value.filter(l => l.status === 'open') : []
)

onMounted(async () => {
  loading.value = true
  try {
    const requests = [
      api.get('/loans/balances/me'),
      api.get('/loans')
    ]

    if (authStore.isAdmin) {
      requests.push(api.get('/users'))
    }

    const responses = await Promise.all(requests)
    balances.value = responses[0].data || []
    loans.value = responses[1].data || []
    if (authStore.isAdmin) {
      users.value = responses[2].data || []
    }
  } catch (err) {
    console.error('Failed to load balance data:', err)
    balances.value = []
    loans.value = []
  } finally {
    loading.value = false
  }
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
    const [balancesRes, loansRes] = await Promise.all([
      api.get('/loans/balances/me'),
      api.get('/loans')
    ])
    balances.value = balancesRes.data || []
    loans.value = loansRes.data || []
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
    const [balancesRes, loansRes] = await Promise.all([
      api.get('/loans/balances/me'),
      api.get('/loans')
    ])
    balances.value = balancesRes.data || []
    loans.value = loansRes.data || []
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
  const amount = formatMoney(loan.amountPLN)
  return `${lender?.name || '?'} → ${borrower?.name || '?'} (${amount} PLN)`
}

function formatMoney(decimal128) {
  if (!decimal128) return '0.00'
  if (typeof decimal128 === 'number') return decimal128.toFixed(2)
  return parseFloat(decimal128.$numberDecimal || decimal128 || 0).toFixed(2)
}

function formatDate(date) {
  return new Date(date).toLocaleDateString('pl-PL')
}
</script>