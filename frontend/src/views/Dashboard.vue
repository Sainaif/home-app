<template>
  <div>
    <div class="mb-8">
      <div v-if="route.params.userId" class="mb-4">
        <button @click="router.push('/')" class="btn btn-secondary">← Powrót do mojego dashboardu</button>
      </div>
      <h1 class="text-4xl font-bold gradient-text mb-2">
        {{ route.params.userId ? `Dashboard użytkownika: ${viewingUser?.name || 'Ładowanie...'}` : $t('dashboard.welcome', { name: authStore.user?.name }) }}
      </h1>
      <p class="text-gray-400">Przegląd {{ route.params.userId ? 'użytkownika' : 'Twojego' }} gospodarstwa domowego</p>
    </div>

    <!-- Stats Overview -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
      <div class="stat-card cursor-pointer hover:scale-105 transition-transform" @click="router.push('/bills')">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-gray-400 text-sm mb-1">Rachunki tego miesiąca</p>
            <p class="text-3xl font-bold">{{ bills.length }}</p>
          </div>
          <div class="w-12 h-12 rounded-xl bg-purple-600/20 flex items-center justify-center">
            <Receipt class="w-6 h-6 text-purple-400" />
          </div>
        </div>
      </div>

      <div class="stat-card cursor-pointer hover:scale-105 transition-transform" @click="router.push('/chores')">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-gray-400 text-sm mb-1">Oczekujące obowiązki</p>
            <p class="text-3xl font-bold">{{ chores.length }}</p>
          </div>
          <div class="w-12 h-12 rounded-xl bg-pink-600/20 flex items-center justify-center">
            <CheckSquare class="w-6 h-6 text-pink-400" />
          </div>
        </div>
      </div>

      <div class="stat-card cursor-pointer hover:scale-105 transition-transform" @click="router.push('/balance')">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-gray-400 text-sm mb-1">Twój bilans</p>
            <p class="text-3xl font-bold" :class="totalBalanceNumber < 0 ? 'text-red-400' : 'text-green-400'">{{ totalBalance }} PLN</p>
          </div>
          <div class="w-12 h-12 rounded-xl" :class="totalBalanceNumber < 0 ? 'bg-red-600/20' : 'bg-green-600/20'">
            <Wallet class="w-6 h-6" :class="totalBalanceNumber < 0 ? 'text-red-400' : 'text-green-400'" />
          </div>
        </div>
      </div>
    </div>

    <!-- Pending Bills to Pay -->
    <div v-if="pendingAllocations.length > 0" class="card mb-6">
      <div class="card-header">
        <h2 class="card-title">Rachunki do zapłaty</h2>
        <div class="text-yellow-400 font-bold">{{ totalPendingAmount.toFixed(2) }} PLN</div>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="border-b border-gray-700">
            <tr class="text-left">
              <th class="pb-3">Rachunek</th>
              <th class="pb-3">Okres</th>
              <th class="pb-3">Twoje zużycie</th>
              <th class="pb-3">Do zapłaty</th>
              <th class="pb-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in pendingAllocations" :key="item.allocation.id" class="border-b border-gray-700">
              <td class="py-3">{{ getBillType(item.bill) }}</td>
              <td class="py-3">{{ formatDateRange(item.bill.periodStart, item.bill.periodEnd) }}</td>
              <td class="py-3">{{ formatUnits(item.allocation.units) }} {{ getUnit(item.bill.type) }}</td>
              <td class="py-3 font-bold text-yellow-400">{{ formatMoney(item.allocation.amountPLN) }} PLN</td>
              <td class="py-3">
                <button @click="viewBill(item.bill.id)" class="text-blue-400 hover:text-blue-300 text-sm">
                  Szczegóły →
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Details Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Recent Bills -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">{{ $t('dashboard.recentBills') }}</h2>
          <Receipt class="w-5 h-5 text-purple-400" />
        </div>

        <div v-if="loading" class="flex justify-center py-8">
          <div class="loading-spinner"></div>
        </div>
        <div v-else-if="bills.length === 0" class="text-center py-8 text-gray-500">
          <FileX class="w-12 h-12 mx-auto mb-2 opacity-50" />
          <p>Brak rachunków</p>
        </div>
        <div v-else class="space-y-3">
          <div v-for="bill in bills.slice(0, 5)" :key="bill.id"
               @click="viewBill(bill.id)"
               class="flex items-center justify-between p-3 rounded-xl bg-gray-700/30 hover:bg-gray-700/50 transition-colors cursor-pointer">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-purple-600/20 flex items-center justify-center">
                <Zap v-if="bill.type === 'electricity'" class="w-5 h-5 text-yellow-400" />
                <Flame v-else-if="bill.type === 'gas'" class="w-5 h-5 text-orange-400" />
                <Wifi v-else-if="bill.type === 'internet'" class="w-5 h-5 text-blue-400" />
                <Users v-else class="w-5 h-5 text-gray-400" />
              </div>
              <div>
                <p class="font-medium">{{ $t(`bills.${bill.type}`) }}</p>
                <p class="text-xs text-gray-400">{{ formatDateRange(bill.periodStart, bill.periodEnd) }}</p>
              </div>
            </div>
            <span class="font-bold text-purple-400">{{ formatMoney(bill.totalAmountPLN) }} PLN</span>
          </div>
        </div>

        <router-link to="/bills" class="btn btn-outline w-full mt-4 flex items-center justify-center gap-2">
          Zobacz wszystkie
          <ArrowRight class="w-4 h-4" />
        </router-link>
      </div>

      <!-- Upcoming Chores -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">{{ $t('dashboard.upcomingChores') }}</h2>
          <CheckSquare class="w-5 h-5 text-pink-400" />
        </div>

        <div v-if="loading" class="flex justify-center py-8">
          <div class="loading-spinner"></div>
        </div>
        <div v-else-if="chores.length === 0" class="text-center py-8 text-gray-500">
          <CheckCircle class="w-12 h-12 mx-auto mb-2 opacity-50" />
          <p>Brak obowiązków</p>
        </div>
        <div v-else class="space-y-3">
          <div v-for="chore in chores.slice(0, 5)" :key="chore.id"
               @click="router.push('/chores')"
               class="flex items-center justify-between p-3 rounded-xl bg-gray-700/30 hover:bg-gray-700/50 transition-colors cursor-pointer">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-pink-600/20 flex items-center justify-center">
                <ClipboardList class="w-5 h-5 text-pink-400" />
              </div>
              <div>
                <p class="font-medium">{{ chore.choreName }}</p>
                <p class="text-xs text-gray-400">{{ chore.userName }}</p>
              </div>
            </div>
            <span class="text-sm text-gray-400">{{ formatDate(chore.dueDate) }}</span>
          </div>
        </div>

        <router-link to="/chores" class="btn btn-outline w-full mt-4 flex items-center justify-center gap-2">
          Zobacz wszystkie
          <ArrowRight class="w-4 h-4" />
        </router-link>
      </div>

      <!-- Balance Overview -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">{{ $t('dashboard.currentBalance') }}</h2>
          <Wallet class="w-5 h-5 text-green-400" />
        </div>

        <div v-if="loading" class="flex justify-center py-8">
          <div class="loading-spinner"></div>
        </div>
        <div v-else-if="balances.length === 0" class="text-center py-8 text-gray-500">
          <BadgeCheck class="w-12 h-12 mx-auto mb-2 opacity-50" />
          <p>Brak zobowiązań</p>
        </div>
        <div v-else class="space-y-3">
          <div v-for="bal in balances.slice(0, 5)" :key="`${bal.fromUserId}-${bal.toUserId}`"
               @click="router.push('/balance')"
               class="flex items-center justify-between p-3 rounded-xl bg-gray-700/30 hover:bg-gray-700/50 transition-colors cursor-pointer">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-red-600/20 flex items-center justify-center">
                <TrendingDown class="w-5 h-5 text-red-400" />
              </div>
              <div>
                <p class="font-medium">{{ bal.fromUserName }}</p>
                <p class="text-xs text-gray-400">dla {{ bal.toUserName }}</p>
              </div>
            </div>
            <span class="font-bold text-red-400">{{ formatMoney(bal.netAmount) }} PLN</span>
          </div>
        </div>

        <router-link to="/balance" class="btn btn-outline w-full mt-4 flex items-center justify-center gap-2">
          Zobacz szczegóły
          <ArrowRight class="w-4 h-4" />
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useEventStream } from '../composables/useEventStream'
import { useDataEvents, DATA_EVENTS } from '../composables/useDataEvents'
import api from '../api/client'
import {
  Receipt, CheckSquare, Wallet, Zap, Flame, Wifi, Users,
  ArrowRight, FileX, CheckCircle, ClipboardList, BadgeCheck, TrendingDown
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const bills = ref([])
const chores = ref([])
const balances = ref([])
const pendingAllocations = ref([])
const loading = ref(false)
const viewingUser = ref(null)

// Determine which user's dashboard to show
const targetUserId = computed(() => route.params.userId || authStore.user?.id)

// Setup SSE for real-time updates
const { connect, on } = useEventStream()

// Setup event bus for cross-component updates
const { on: onDataEvent } = useDataEvents()

const totalBalanceNumber = computed(() => {
  const userId = targetUserId.value
  return balances.value.reduce((sum, bal) => {
    const amount = parseFloat(bal.netAmount.$numberDecimal || bal.netAmount || 0)

    // If you're the fromUser, you owe (negative)
    if (bal.fromUserId === userId) {
      return sum - amount
    }
    // If you're the toUser, someone owes you (positive)
    if (bal.toUserId === userId) {
      return sum + amount
    }
    return sum
  }, 0)
})

const totalBalance = computed(() => {
  const total = totalBalanceNumber.value
  const formatted = total.toFixed(2)
  return total >= 0 ? `+${formatted}` : formatted
})

const totalPendingAmount = computed(() => {
  return pendingAllocations.value.reduce((sum, item) => {
    const amount = parseFloat(item.allocation.amountPLN || 0)
    return sum + amount
  }, 0)
})

onMounted(async () => {
  // Load user info if viewing another user's dashboard
  if (route.params.userId) {
    try {
      const userRes = await api.get(`/users/${route.params.userId}`)
      viewingUser.value = userRes.data
    } catch (err) {
      console.error('Failed to load user:', err)
    }
  }

  // Load initial data
  await loadDashboardData()

  // Connect to SSE stream (only for own dashboard)
  if (!route.params.userId) {
    connect()

    // Listen for relevant events
    on('bill.created', () => {
      console.log('[Dashboard] Bill created, refreshing...')
      loadBills()
    })

    on('chore.updated', () => {
      console.log('[Dashboard] Chore updated, refreshing...')
      loadChores()
    })

    on('payment.created', () => {
      console.log('[Dashboard] Payment created, refreshing...')
      loadBalances()
    })
  }

  // Listen to cross-component events
  onDataEvent(DATA_EVENTS.USER_UPDATED, () => loadDashboardData())
  onDataEvent(DATA_EVENTS.GROUP_UPDATED, () => loadDashboardData())
  onDataEvent(DATA_EVENTS.CHORE_CREATED, () => loadChores())
  onDataEvent(DATA_EVENTS.CHORE_ASSIGNMENT_UPDATED, () => loadChores())
})

// Watch for userId changes
watch(targetUserId, () => {
  loadDashboardData()
})

async function loadDashboardData() {
  loading.value = true
  try {
    const userId = targetUserId.value
    const isViewingOther = route.params.userId

    // First, load chore definitions and users for enrichment
    const [allChoresRes, usersRes] = await Promise.all([
      api.get('/chores'),
      api.get('/users')
    ])

    const allChoresData = allChoresRes.data || []
    const usersData = usersRes.data || []

    // Then load dashboard data
    const [billsRes, choresRes, balanceRes] = await Promise.all([
      api.get('/bills'),
      isViewingOther
        ? api.get(`/chore-assignments?userId=${userId}&status=pending`)
        : api.get('/chore-assignments/me?status=pending'),
      isViewingOther
        ? api.get(`/loans/balances/user/${userId}`)
        : api.get('/loans/balances/me')
    ])

    bills.value = billsRes.data || []
    const choreAssignments = choresRes.data || []

    // Enrich chore assignments with chore and user details
    for (let assignment of choreAssignments) {
      const chore = allChoresData.find(c => c.id === assignment.choreId)
      if (chore) {
        assignment.choreName = chore.name
      }
      const user = usersData.find(u => u.id === assignment.assigneeUserId)
      if (user) {
        assignment.userName = user.name
      }
    }

    chores.value = choreAssignments
    // Balance API may return object with balances array or just array
    balances.value = Array.isArray(balanceRes.data) ? balanceRes.data : (balanceRes.data?.balances || [])

    // Load pending allocations (posted bills user hasn't paid yet)
    await loadPendingAllocations()
  } catch (err) {
    console.error('Failed to load dashboard data:', err)
    bills.value = []
    chores.value = []
    balances.value = []
  } finally {
    loading.value = false
  }
}

async function loadPendingAllocations() {
  try {
    const userId = targetUserId.value

    // Get user's groupId if viewing another user
    let userGroupId = authStore.user?.groupId
    if (route.params.userId && viewingUser.value) {
      userGroupId = viewingUser.value.groupId
    }

    // Get all posted bills
    const billsRes = await api.get('/bills?status=posted')
    const postedBills = billsRes.data || []

    // For each posted bill, get user's allocations
    const allocationsPromises = postedBills.map(async (bill) => {
      try {
        const allocRes = await api.get(`/bills/${bill.id}/allocation`)
        const allocations = allocRes.data || []

        // Find user's allocation (either direct user or through group)
        const userAllocation = allocations.find(a => {
          if (a.subjectType === 'user' && a.subjectId === userId) {
            return true
          }
          if (a.subjectType === 'group' && userGroupId && a.subjectId === userGroupId) {
            return true
          }
          return false
        })

        if (userAllocation) {
          // Map new allocation format to expected Dashboard format
          const mapped = {
            bill,
            allocation: {
              id: userAllocation.subjectId,
              amountPLN: userAllocation.amount || 0, // New endpoint returns plain number as 'amount'
              units: userAllocation.units || 0
            }
          }
          console.log('Mapped allocation:', mapped)
          return mapped
        }
      } catch (err) {
        console.error(`Failed to load allocations for bill ${bill.id}:`, err)
      }
      return null
    })

    const results = await Promise.all(allocationsPromises)
    pendingAllocations.value = results.filter(r => r !== null)
  } catch (err) {
    console.error('Failed to load pending allocations:', err)
    pendingAllocations.value = []
  }
}

async function loadBills() {
  try {
    const res = await api.get('/bills')
    bills.value = res.data || []
  } catch (err) {
    console.error('Failed to load bills:', err)
  }
}

async function loadChores() {
  try {
    const res = await api.get('/chore-assignments/me?status=pending')
    chores.value = res.data || []
  } catch (err) {
    console.error('Failed to load chores:', err)
  }
}

async function loadBalances() {
  try {
    const res = await api.get('/loans/balances/me')
    // Balance API may return object with balances array or just array
    balances.value = Array.isArray(res.data) ? res.data : (res.data?.balances || [])
  } catch (err) {
    console.error('Failed to load balances:', err)
  }
}

function formatMoney(decimal128) {
  return parseFloat(decimal128.$numberDecimal || decimal128 || 0).toFixed(2)
}

function formatDate(date) {
  return new Date(date).toLocaleDateString('pl-PL', { day: 'numeric', month: 'short' })
}

function formatDateRange(start, end) {
  const startDate = new Date(start).toLocaleDateString('pl-PL', { day: 'numeric', month: 'short' })
  const endDate = new Date(end).toLocaleDateString('pl-PL', { day: 'numeric', month: 'short' })
  return `${startDate} - ${endDate}`
}

function formatUnits(decimal128) {
  return parseFloat(decimal128.$numberDecimal || decimal128 || 0).toFixed(3)
}

function getBillType(bill) {
  if (bill.type === 'electricity') return 'Prąd'
  if (bill.type === 'gas') return 'Gaz'
  if (bill.type === 'internet') return 'Internet'
  if (bill.type === 'inne' && bill.customType) return bill.customType
  return bill.type
}

function getUnit(type) {
  if (type === 'electricity') return 'kWh'
  if (type === 'gas') return 'm³'
  return 'jednostek'
}

function viewBill(billId) {
  router.push(`/bills/${billId}`)
}
</script>