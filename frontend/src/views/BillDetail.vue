<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center">
        <button @click="$router.back()" class="btn btn-secondary mr-4">← Powrót</button>
        <h1 class="text-3xl font-bold">Szczegóły rachunku</h1>
      </div>
      <!-- Reopen button for admins -->
      <button
        v-if="authStore.isAdmin && bill && (bill.status === 'posted' || bill.status === 'closed')"
        @click="showReopenModal = true"
        class="btn btn-outline flex items-center gap-2"
      >
        <RotateCcw class="w-4 h-4" />
        Ponownie otwórz
      </button>
    </div>

    <div v-if="loading" class="text-center py-8">Ładowanie...</div>
    <div v-else-if="!bill" class="text-center py-8 text-red-400">Nie znaleziono rachunku</div>
    <div v-else>
      <!-- Bill Info Card -->
      <div class="card mb-6">
        <h2 class="text-xl font-semibold mb-4">Informacje o rachunku</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <span class="text-gray-400">Typ:</span>
            <span class="ml-2 font-medium">{{ getBillType(bill) }}</span>
          </div>
          <div>
            <span class="text-gray-400">Status:</span>
            <span class="ml-2 font-medium">{{ bill.status }}</span>
          </div>
          <div>
            <span class="text-gray-400">Okres:</span>
            <span class="ml-2 font-medium">{{ formatDate(bill.periodStart) }} - {{ formatDate(bill.periodEnd) }}</span>
          </div>
          <div>
            <span class="text-gray-400">Całkowita kwota:</span>
            <span class="ml-2 font-medium">{{ formatMoney(bill.totalAmountPLN) }} PLN</span>
          </div>
          <div v-if="bill.totalUnits">
            <span class="text-gray-400">Całkowite zużycie:</span>
            <span class="ml-2 font-medium">{{ formatMeterValue(bill.totalUnits) }} {{ getUnit(bill.type) }}</span>
          </div>
          <div v-if="bill.reopenedAt">
            <span class="text-gray-400">Ponownie otwarty:</span>
            <span class="ml-2 font-medium">{{ formatDateTime(bill.reopenedAt) }}</span>
          </div>
          <div v-if="bill.reopenReason" class="md:col-span-2">
            <span class="text-gray-400">Powód ponownego otwarcia:</span>
            <span class="ml-2 font-medium">{{ bill.reopenReason }}</span>
          </div>
        </div>
      </div>

      <!-- Reopen Modal -->
      <div v-if="showReopenModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showReopenModal = false">
        <div class="card max-w-md w-full mx-4">
          <div class="flex justify-between items-center mb-6">
            <h2 class="text-2xl font-bold gradient-text">Ponownie otwórz rachunek</h2>
            <button @click="showReopenModal = false" class="text-gray-400 hover:text-white">
              <X class="w-6 h-6" />
            </button>
          </div>

          <form @submit.prevent="reopenBill" class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2">Status docelowy</label>
              <select v-model="reopenData.targetStatus" required class="input">
                <option value="draft">Szkic (draft)</option>
                <option value="posted" v-if="bill.status === 'closed'">Zamieszczony (posted)</option>
              </select>
            </div>

            <div>
              <label class="block text-sm font-medium mb-2">Powód ponownego otwarcia *</label>
              <textarea
                v-model="reopenData.reason"
                required
                class="input"
                rows="3"
                placeholder="Np. Poprawka odczytu, błąd w alokacji..."
              ></textarea>
            </div>

            <div v-if="reopenError" class="flex items-center gap-2 p-3 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
              <AlertCircle class="w-4 h-4" />
              {{ reopenError }}
            </div>

            <div class="flex gap-3">
              <button type="submit" :disabled="reopening" class="btn btn-primary flex-1 flex items-center justify-center gap-2">
                <div v-if="reopening" class="loading-spinner"></div>
                <RotateCcw v-else class="w-5 h-5" />
                {{ reopening ? 'Otwieranie...' : 'Ponownie otwórz' }}
              </button>
              <button type="button" @click="showReopenModal = false" class="btn btn-outline">
                Anuluj
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- Readings Card -->
      <div class="card mb-6">
        <h2 class="text-xl font-semibold mb-4">Odczyty liczników</h2>
        <div v-if="loadingReadings" class="text-center py-4">Ładowanie odczytów...</div>
        <div v-else-if="readings.length === 0" class="text-center py-4 text-gray-400">Brak odczytów</div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b border-gray-700">
              <tr class="text-left">
                <th class="pb-3">Użytkownik</th>
                <th class="pb-3">Odczyt</th>
                <th class="pb-3">Zużycie</th>
                <th class="pb-3">Data</th>
                <th class="pb-3">Źródło</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="reading in readings" :key="reading.id" class="border-b border-gray-700">
                <td class="py-3">{{ getUserName(reading.userId) }}</td>
                <td class="py-3">{{ formatMeterValue(reading.meterValue) }} {{ getUnit(bill.type) }}</td>
                <td class="py-3">{{ formatMeterValue(reading.units) }} {{ getUnit(bill.type) }}</td>
                <td class="py-3">{{ formatDateTime(reading.recordedAt) }}</td>
                <td class="py-3">
                  <span :class="reading.source === 'invalid' ? 'text-red-400' : 'text-gray-400'">
                    {{ reading.source || 'user' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Allocations Card -->
      <div class="card mb-6">
        <h2 class="text-xl font-semibold mb-4">Podział kosztów</h2>
        <div v-if="loadingAllocations" class="text-center py-4">Ładowanie podziału...</div>
        <div v-else-if="allocations.length === 0" class="text-center py-4 text-gray-400">
          Rachunek jeszcze nie został podzielony
        </div>
        <div v-else>
          <!-- Summary -->
          <div class="mb-4 p-4 bg-gray-700 rounded">
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <span class="text-gray-400">Suma odczytów:</span>
                <span class="ml-2 font-medium">{{ totalUserUnits.toFixed(3) }} {{ getUnit(bill.type) }}</span>
              </div>
              <div>
                <span class="text-gray-400">Część wspólna:</span>
                <span class="ml-2 font-medium">{{ sharedPortion.toFixed(3) }} {{ getUnit(bill.type) }}</span>
                <span class="text-gray-400 text-sm ml-2">({{ sharedPortionPercent.toFixed(1) }}%)</span>
              </div>
              <div>
                <span class="text-gray-400">Koszt części wspólnej:</span>
                <span class="ml-2 font-medium">{{ sharedCost.toFixed(2) }} PLN</span>
              </div>
            </div>
          </div>

          <!-- Per-user breakdown -->
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead class="border-b border-gray-700">
                <tr class="text-left">
                  <th class="pb-3">Użytkownik/Grupa</th>
                  <th class="pb-3">Zużycie</th>
                  <th class="pb-3">Kwota</th>
                  <th class="pb-3">Metoda</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="allocation in allocations" :key="allocation.id" class="border-b border-gray-700">
                  <td class="py-3">{{ getSubjectName(allocation) }}</td>
                  <td class="py-3">{{ formatMeterValue(allocation.units) }} {{ getUnit(bill.type) }}</td>
                  <td class="py-3 font-medium">{{ formatMoney(allocation.amountPLN) }} PLN</td>
                  <td class="py-3 text-gray-400">{{ allocation.method }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { RotateCcw, X, AlertCircle } from 'lucide-vue-next'
import api from '../api/client'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const billId = route.params.id

const bill = ref(null)
const readings = ref([])
const allocations = ref([])
const users = ref([])
const groups = ref([])
const loading = ref(false)
const loadingReadings = ref(false)
const loadingAllocations = ref(false)

// Reopen state
const showReopenModal = ref(false)
const reopening = ref(false)
const reopenError = ref(null)
const reopenData = ref({
  targetStatus: 'draft',
  reason: ''
})

const totalUserUnits = computed(() => {
  if (!bill.value?.totalUnits) return 0
  const total = parseFloat(bill.value.totalUnits.$numberDecimal || bill.value.totalUnits || 0)
  const userSum = readings.value
    .filter(r => r.source !== 'invalid')
    .reduce((sum, r) => sum + parseFloat(r.units.$numberDecimal || r.units || 0), 0)
  return userSum
})

const sharedPortion = computed(() => {
  if (!bill.value?.totalUnits) return 0
  const total = parseFloat(bill.value.totalUnits.$numberDecimal || bill.value.totalUnits || 0)
  return Math.max(0, total - totalUserUnits.value)
})

const sharedPortionPercent = computed(() => {
  if (!bill.value?.totalUnits) return 0
  const total = parseFloat(bill.value.totalUnits.$numberDecimal || bill.value.totalUnits || 0)
  if (total === 0) return 0
  return (sharedPortion.value / total) * 100
})

const sharedCost = computed(() => {
  if (!bill.value?.totalAmountPLN || !bill.value?.totalUnits) return 0
  const totalAmount = parseFloat(bill.value.totalAmountPLN.$numberDecimal || bill.value.totalAmountPLN || 0)
  const totalUnits = parseFloat(bill.value.totalUnits.$numberDecimal || bill.value.totalUnits || 0)
  if (totalUnits === 0) return 0
  const pricePerUnit = totalAmount / totalUnits
  return sharedPortion.value * pricePerUnit
})

onMounted(async () => {
  loading.value = true
  try {
    const [billRes, usersRes, groupsRes] = await Promise.all([
      api.get(`/bills/${billId}`),
      api.get('/users'),
      api.get('/groups')
    ])
    bill.value = billRes.data
    users.value = usersRes.data || []
    groups.value = groupsRes.data || []

    // Load readings
    loadingReadings.value = true
    const readingsRes = await api.get(`/consumptions?billId=${billId}`)
    readings.value = readingsRes.data || []
    loadingReadings.value = false

    // Load allocations
    loadingAllocations.value = true
    const allocationsRes = await api.get(`/allocations?billId=${billId}`)
    allocations.value = allocationsRes.data || []
    loadingAllocations.value = false
  } catch (err) {
    console.error('Failed to load bill details:', err)
    bill.value = null
  } finally {
    loading.value = false
  }
})

function getBillType(b) {
  if (b.type === 'electricity') return 'Prąd'
  if (b.type === 'gas') return 'Gaz'
  if (b.type === 'internet') return 'Internet'
  if (b.type === 'inne' && b.customType) return b.customType
  return b.type
}

function getUnit(type) {
  if (type === 'electricity') return 'kWh'
  if (type === 'gas') return 'm³'
  return 'jednostek'
}

function getUserName(userId) {
  const user = users.value.find(u => u.id === userId)
  return user?.name || 'Nieznany'
}

function getSubjectName(allocation) {
  if (allocation.subjectType === 'user') {
    const user = users.value.find(u => u.id === allocation.subjectId)
    return user?.name || 'Nieznany użytkownik'
  } else if (allocation.subjectType === 'group') {
    const group = groups.value.find(g => g.id === allocation.subjectId)
    return group?.name || 'Nieznana grupa'
  }
  return 'Nieznany'
}

function formatMoney(decimal128) {
  if (!decimal128) return '0.00'
  return parseFloat(decimal128.$numberDecimal || decimal128 || 0).toFixed(2)
}

function formatMeterValue(value) {
  if (!value) return '0.000'
  const numValue = parseFloat(value.$numberDecimal || value || 0)
  return numValue.toFixed(3)
}

function formatDate(date) {
  if (!date) return '-'
  return new Date(date).toLocaleDateString('pl-PL')
}

function formatDateTime(date) {
  if (!date) return '-'
  return new Date(date).toLocaleString('pl-PL')
}

async function reopenBill() {
  reopening.value = true
  reopenError.value = null

  try {
    await api.post(`/bills/${billId}/reopen`, reopenData.value)

    // Refresh bill data
    const billRes = await api.get(`/bills/${billId}`)
    bill.value = billRes.data

    // Reload allocations as they may have been cleared
    loadingAllocations.value = true
    const allocationsRes = await api.get(`/allocations?billId=${billId}`)
    allocations.value = allocationsRes.data || []
    loadingAllocations.value = false

    showReopenModal.value = false
    reopenData.value = { targetStatus: 'draft', reason: '' }
  } catch (err) {
    reopenError.value = err.response?.data?.error || 'Nie udało się ponownie otworzyć rachunku'
  } finally {
    reopening.value = false
  }
}
</script>
