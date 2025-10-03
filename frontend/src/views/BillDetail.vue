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
        Zmień status rachunku
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
          <div v-if="bill.paymentDeadline">
            <span class="text-gray-400">Termin płatności:</span>
            <span class="ml-2 font-medium">{{ formatDate(bill.paymentDeadline) }}</span>
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
            <h2 class="text-2xl font-bold gradient-text">Zmień status rachunku</h2>
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
                {{ reopening ? 'Zmiana...' : 'Zmień status' }}
              </button>
              <button type="button" @click="showReopenModal = false" class="btn btn-outline">
                Anuluj
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- Allocation Card -->
      <div v-if="bill.status === 'posted' || bill.status === 'closed'" class="card mb-6">
        <h2 class="text-xl font-semibold mb-4">Alokacja kosztów</h2>
        <div v-if="loadingAllocations" class="text-center py-4">Ładowanie alokacji...</div>
        <div v-else-if="allocations.length === 0" class="text-center py-4 text-gray-400">Brak danych o alokacji</div>
        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          <div v-for="allocation in allocations" :key="allocation.subjectId"
               class="bg-gray-800/50 rounded-lg p-3 border border-gray-700/50">
            <div class="flex justify-between items-start">
              <div>
                <p class="font-medium text-white">{{ allocation.subjectName }}</p>
                <p class="text-xs text-gray-400">Waga: {{ allocation.weight.toFixed(2) }}</p>
              </div>
              <div class="text-right">
                <p class="font-bold text-purple-400">{{ formatMoney(allocation.amount) }} PLN</p>
                <div v-if="allocation.units !== undefined" class="text-xs text-gray-400">
                  {{ formatMeterValue(allocation.units) }} {{ getUnit(bill.type) }}
                </div>
              </div>
            </div>
            <div v-if="allocation.personalAmount !== undefined && allocation.sharedAmount !== undefined"
                 class="mt-2 pt-2 border-t border-gray-700/50 text-xs text-gray-400 space-y-1">
              <div class="flex justify-between">
                <span>Osobiste:</span>
                <span>{{ formatMoney(allocation.personalAmount) }} PLN</span>
              </div>
              <div class="flex justify-between">
                <span>Wspólne:</span>
                <span>{{ formatMoney(allocation.sharedAmount) }} PLN</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Readings Card -->
      <div v-if="bill.type === 'electricity' || (bill.type === 'inne' && bill.allocationType === 'metered')" class="card mb-6">
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

    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useDataEvents, DATA_EVENTS } from '../composables/useDataEvents'
import { RotateCcw, X, AlertCircle } from 'lucide-vue-next'
import api from '../api/client'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { on } = useDataEvents()
const billId = route.params.id

const bill = ref(null)
const readings = ref([])
const allocations = ref([])
const users = ref([])
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

onMounted(async () => {
  loading.value = true
  try {
    const [billRes, usersRes] = await Promise.all([
      api.get(`/bills/${billId}`),
      api.get('/users')
    ])
    bill.value = billRes.data
    users.value = usersRes.data || []

    // Load readings
    loadingReadings.value = true
    const readingsRes = await api.get(`/consumptions?billId=${billId}`)
    readings.value = readingsRes.data || []
    loadingReadings.value = false

    // Load allocations if bill is posted or closed
    if (bill.value && (bill.value.status === 'posted' || bill.value.status === 'closed')) {
      loadingAllocations.value = true
      try {
        const allocRes = await api.get(`/bills/${billId}/allocation`)
        allocations.value = allocRes.data || []
      } catch (err) {
        console.error('Failed to load allocations:', err)
        allocations.value = []
      } finally {
        loadingAllocations.value = false
      }
    }
  } catch (err) {
    console.error('Failed to load bill details:', err)
    bill.value = null
  } finally {
    loading.value = false
  }

  // Listen for user updates
  on(DATA_EVENTS.USER_UPDATED, async () => {
    const usersRes = await api.get('/users')
    users.value = usersRes.data || []
  })
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

    showReopenModal.value = false
    reopenData.value = { targetStatus: 'draft', reason: '' }
  } catch (err) {
    reopenError.value = err.response?.data?.error || 'Nie udało się ponownie otworzyć rachunku'
  } finally {
    reopening.value = false
  }
}
</script>
