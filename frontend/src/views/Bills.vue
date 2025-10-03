<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="text-4xl font-bold gradient-text mb-2">{{ $t('bills.title') }}</h1>
        <p class="text-gray-400">Historia rachunków i odczyty liczników</p>
      </div>
      <button v-if="authStore.hasPermission('bills.create') && activeTab === 'bills'" @click="showCreateModal = true" class="btn btn-primary flex items-center gap-2">
        <Plus class="w-5 h-5" />
        {{ $t('bills.createNew') }}
      </button>
    </div>

    <!-- Tabs -->
    <div class="flex gap-2 mb-6">
      <button
        @click="activeTab = 'bills'"
        :class="['btn', activeTab === 'bills' ? 'btn-primary' : 'btn-outline']"
        class="flex items-center gap-2">
        <Receipt class="w-4 h-4" />
        Rachunki
      </button>
      <button
        v-if="hasMeteredBills"
        @click="activeTab = 'readings'"
        :class="['btn', activeTab === 'readings' ? 'btn-primary' : 'btn-outline']"
        class="flex items-center gap-2">
        <Gauge class="w-4 h-4" />
        Odczyty
      </button>
    </div>

    <!-- Create Bill Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showCreateModal = false">
      <div class="card max-w-lg w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold gradient-text">Nowy Rachunek</h2>
          <button @click="showCreateModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="createBill" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">Typ</label>
            <select v-model="newBill.type" required class="input">
              <option value="electricity">Prąd</option>
              <option value="gas">Gaz</option>
              <option value="internet">Internet</option>
              <option value="inne">Inne</option>
            </select>
          </div>

          <div v-if="newBill.type === 'inne'">
            <label class="block text-sm font-medium mb-2">Nazwa typu</label>
            <input v-model="newBill.customType" type="text" required class="input" placeholder="np. Czynsz, Woda..." />
          </div>

          <div v-if="newBill.type === 'inne'">
            <label class="block text-sm font-medium mb-2">Sposób rozliczenia</label>
            <select v-model="newBill.allocationType" required class="input">
              <option value="simple">Równy podział (jak Gaz/Internet)</option>
              <option value="metered">Według odczytów (jak Prąd)</option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">Kwota (PLN)</label>
            <input v-model.number="newBill.totalAmountPLN" type="number" step="0.01" required class="input" placeholder="150.00" />
          </div>

          <div v-if="newBill.type === 'electricity' || newBill.type === 'gas' || (newBill.type === 'inne' && newBill.allocationType === 'metered')">
            <label class="block text-sm font-medium mb-2">Jednostki</label>
            <input v-model.number="newBill.totalUnits" type="number" step="0.001" class="input" placeholder="100.000" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-2">Okres od</label>
              <input v-model="newBill.periodStart" type="date" required class="input" min="2000-01-01" max="2099-12-31" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-2">Okres do</label>
              <input v-model="newBill.periodEnd" type="date" required class="input" min="2000-01-01" max="2099-12-31" />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">Termin płatności (opcjonalny)</label>
            <input v-model="newBill.paymentDeadline" type="date" class="input" min="2000-01-01" max="2099-12-31" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">Notatki</label>
            <textarea v-model="newBill.notes" class="input" rows="3" placeholder="Opcjonalne uwagi..."></textarea>
          </div>

          <div v-if="createError" class="flex items-center gap-2 p-3 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
            <AlertCircle class="w-4 h-4" />
            {{ createError }}
          </div>

          <div class="flex gap-3">
            <button type="submit" :disabled="creating" class="btn btn-primary flex-1 flex items-center justify-center gap-2">
              <div v-if="creating" class="loading-spinner"></div>
              <Plus v-else class="w-5 h-5" />
              {{ creating ? 'Tworzenie...' : 'Utwórz rachunek' }}
            </button>
            <button type="button" @click="showCreateModal = false" class="btn btn-outline">
              Anuluj
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Bills Tab -->
    <div v-show="activeTab === 'bills'">
      <!-- Filters -->
      <div class="card mb-6">
        <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">Typ</label>
            <select v-model="filters.type" class="input">
              <option value="">Wszystkie</option>
              <option value="electricity">Prąd</option>
              <option value="gas">Gaz</option>
              <option value="internet">Internet</option>
              <option value="inne">Inne</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Data od</label>
            <input v-model="filters.dateFrom" type="date" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Data do</label>
            <input v-model="filters.dateTo" type="date" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Sortuj</label>
            <select v-model="filters.sortBy" class="input">
              <option value="date-desc">Data (najnowsze)</option>
              <option value="date-asc">Data (najstarsze)</option>
              <option value="amount-desc">Kwota (malejąco)</option>
              <option value="amount-asc">Kwota (rosnąco)</option>
            </select>
          </div>
        </div>
      </div>

      <div class="card">
        <div v-if="loading" class="flex justify-center py-12">
          <div class="loading-spinner"></div>
        </div>
        <div v-else-if="filteredBills.length === 0" class="text-center py-12 text-gray-500">
          <FileX class="w-16 h-16 mx-auto mb-4 opacity-50" />
          <p class="text-lg">Brak rachunków</p>
        </div>
        <template v-else>
          <!-- Mobile card layout -->
          <div class="md:hidden space-y-3">
            <div v-for="bill in filteredBills" :key="bill.id"
               @click="$router.push(`/bills/${bill.id}`)"
               class="p-4 bg-gray-700/50 rounded-lg border border-gray-600/50 hover:bg-gray-700 transition-colors cursor-pointer">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0"
                     :class="{
                       'bg-yellow-600/20': bill.type === 'electricity',
                       'bg-orange-600/20': bill.type === 'gas',
                       'bg-blue-600/20': bill.type === 'internet',
                       'bg-purple-600/20': bill.type === 'inne'
                     }">
                  <Zap v-if="bill.type === 'electricity'" class="w-5 h-5 text-yellow-400" />
                  <Flame v-else-if="bill.type === 'gas'" class="w-5 h-5 text-orange-400" />
                  <Wifi v-else-if="bill.type === 'internet'" class="w-5 h-5 text-blue-400" />
                  <FileX v-else class="w-5 h-5 text-purple-400" />
                </div>
                <div>
                  <div class="font-medium text-sm">{{ bill.type === 'inne' && bill.customType ? bill.customType : $t(`bills.${bill.type}`) }}</div>
                  <div class="text-xs text-gray-400">{{ formatDate(bill.periodStart) }} - {{ formatDate(bill.periodEnd) }}</div>
                </div>
              </div>
              <span :class="`badge badge-${bill.status} text-xs`">
                {{ $t(`bills.${bill.status}`) }}
              </span>
            </div>

            <div class="flex items-center justify-between">
              <span class="text-lg font-bold text-purple-400">{{ formatMoney(bill.totalAmountPLN) }} PLN</span>
              <span v-if="bill.totalUnits" class="text-sm text-gray-300">{{ formatUnits(bill.totalUnits) }} {{ getUnit(bill.type) }}</span>
            </div>

            <div v-if="authStore.isAdmin" class="flex items-center gap-2 mt-3 pt-3 border-t border-gray-600/50" @click.stop>
              <button v-if="bill.status === 'draft'" @click="postBill(bill.id)"
                      class="btn btn-sm btn-primary flex items-center gap-1 flex-1">
                <Send class="w-3 h-3" />
                Opublikuj
              </button>
              <button v-if="bill.status === 'posted'" @click="closeBill(bill.id)"
                      class="btn btn-sm btn-secondary flex items-center gap-1 flex-1">
                <Check class="w-3 h-3" />
                Zamknij
              </button>
              <button @click="deleteBill(bill.id)"
                      class="btn btn-sm bg-red-600/20 hover:bg-red-600/30 text-red-400 flex items-center gap-1">
                <Trash2 class="w-3 h-3" />
              </button>
            </div>
            </div>
          </div>

          <!-- Desktop table layout -->
          <div class="table-wrapper hidden md:block">
            <table>
            <thead>
              <tr>
                <th>{{ $t('bills.type') }}</th>
                <th>{{ $t('bills.period') }}</th>
                <th>{{ $t('bills.amount') }}</th>
                <th>{{ $t('bills.totalUnits') }}</th>
                <th>Opis</th>
                <th>{{ $t('bills.status') }}</th>
                <th v-if="authStore.isAdmin">{{ $t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="bill in filteredBills" :key="bill.id" @click="$router.push(`/bills/${bill.id}`)" class="cursor-pointer">
                <td>
                  <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg flex items-center justify-center"
                         :class="{
                           'bg-yellow-600/20': bill.type === 'electricity',
                           'bg-orange-600/20': bill.type === 'gas',
                           'bg-blue-600/20': bill.type === 'internet',
                           'bg-purple-600/20': bill.type === 'inne'
                         }">
                      <Zap v-if="bill.type === 'electricity'" class="w-5 h-5 text-yellow-400" />
                      <Flame v-else-if="bill.type === 'gas'" class="w-5 h-5 text-orange-400" />
                      <Wifi v-else-if="bill.type === 'internet'" class="w-5 h-5 text-blue-400" />
                      <FileX v-else class="w-5 h-5 text-purple-400" />
                    </div>
                    <div>
                      <span class="font-medium">{{ bill.type === 'inne' && bill.customType ? bill.customType : $t(`bills.${bill.type}`) }}</span>
                    </div>
                  </div>
                </td>
                <td>
                  <div class="flex items-center gap-2">
                    <Calendar class="w-4 h-4 text-gray-500" />
                    <span>{{ formatDate(bill.periodStart) }} - {{ formatDate(bill.periodEnd) }}</span>
                  </div>
                </td>
                <td>
                  <div class="flex items-center gap-2">
                    <span class="font-bold text-purple-400">{{ formatMoney(bill.totalAmountPLN) }} PLN</span>
                    <button @click.stop="toggleBillExpansion(bill.id)" class="text-gray-400 hover:text-purple-400 transition-colors">
                      <ChevronDown v-if="!expandedBills[bill.id]" class="w-4 h-4" />
                      <ChevronUp v-else class="w-4 h-4" />
                    </button>
                  </div>
                </td>
                <td>
                  <span class="text-gray-300">{{ bill.totalUnits ? formatUnits(bill.totalUnits) + ' ' + getUnit(bill.type) : '-' }}</span>
                </td>
                <td>
                  <span class="text-gray-400 text-sm">{{ bill.notes || '-' }}</span>
                </td>
                <td>
                  <span :class="`badge badge-${bill.status}`">
                    {{ $t(`bills.${bill.status}`) }}
                  </span>
                </td>
                <td v-if="authStore.isAdmin" @click.stop>
                  <div class="flex items-center gap-2">
                    <button v-if="bill.status === 'draft'" @click="postBill(bill.id)"
                            class="btn btn-sm btn-primary flex items-center gap-1">
                      <Send class="w-3 h-3" />
                      {{ $t('bills.post') }}
                    </button>
                    <button v-if="bill.status === 'posted'" @click="closeBill(bill.id)"
                            class="btn btn-sm btn-secondary flex items-center gap-1">
                      <Check class="w-3 h-3" />
                      {{ $t('bills.close') }}
                    </button>
                    <button @click="deleteBill(bill.id)"
                            class="btn btn-sm bg-red-600/20 hover:bg-red-600/30 text-red-400 flex items-center gap-1">
                      <Trash2 class="w-3 h-3" />
                      Usuń
                    </button>
                  </div>
                </td>
              </tr>
              <!-- Allocation breakdown row -->
              <template v-for="bill in filteredBills" :key="bill.id + '-allocation'">
                <tr v-if="expandedBills[bill.id]" class="bg-gray-800/30">
                  <td colspan="7" class="p-4">
                    <div v-if="loadingAllocations[bill.id]" class="text-center text-gray-400">
                      Ładowanie rozliczenia...
                    </div>
                    <div v-else-if="billAllocations[bill.id] && billAllocations[bill.id].length > 0" class="space-y-2">
                      <h3 class="text-sm font-semibold text-purple-400 mb-3">Rozliczenie między użytkownikami:</h3>
                      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                        <div v-for="allocation in billAllocations[bill.id]" :key="allocation.subjectId"
                             class="bg-gray-800/50 rounded-lg p-3 border border-gray-700/50">
                          <div class="flex justify-between items-start">
                            <div>
                              <p class="font-medium text-white">{{ allocation.subjectName }}</p>
                              <p class="text-xs text-gray-400">Waga: {{ allocation.weight.toFixed(2) }}</p>
                            </div>
                            <div class="text-right">
                              <p class="font-bold text-purple-400">{{ formatMoney(allocation.amount) }} PLN</p>
                              <div v-if="allocation.units !== undefined" class="text-xs text-gray-400">
                                {{ formatUnits(allocation.units) }} {{ getUnit(bill.type) }}
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
                    <div v-else class="text-center text-gray-400">
                      Brak danych o rozliczeniu
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
            </table>
          </div>
        </template>
      </div>
    </div>

    <!-- Readings Tab -->
    <div v-show="activeTab === 'readings'">
      <!-- Add Reading Form -->
      <div class="card mb-6">
        <h2 class="text-xl font-semibold mb-4">{{ $t('readings.addReading') }}</h2>
        <form @submit.prevent="submitReading" class="space-y-4">
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('readings.bill') }}</label>
              <select v-model="form.billId" required class="input">
                <option value="">Wybierz rachunek</option>
                <option v-for="bill in postedBills" :key="bill.id" :value="bill.id">
                  {{ $t(`bills.${bill.type}`) }} - {{ formatDate(bill.periodStart) }}
                </option>
              </select>
            </div>

            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('readings.meterReading') }}</label>
              <input v-model.number="form.meterReading" type="number" step="0.001" required class="input" />
            </div>

            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('readings.date') }}</label>
              <input v-model="form.readingDate" type="datetime-local" required class="input" />
            </div>
          </div>

          <button type="submit" :disabled="loadingReading" class="btn btn-primary">
            {{ $t('readings.submit') }}
          </button>
        </form>
      </div>

      <!-- Readings Filters -->
      <div class="card mb-6">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">Rachunek</label>
            <select v-model="readingFilters.billId" class="input">
              <option value="">Wszystkie rachunki</option>
              <option v-for="bill in allBills" :key="bill.id" :value="bill.id">
                {{ $t(`bills.${bill.type}`) }} - {{ formatDate(bill.periodStart) }}
              </option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Użytkownik</label>
            <select v-model="readingFilters.userId" class="input">
              <option value="">Wszyscy użytkownicy</option>
              <option v-for="user in users" :key="user.id" :value="user.id">
                {{ user.name }}
              </option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">Sortuj</label>
            <select v-model="readingFilters.sortBy" class="input">
              <option value="date-desc">Data (najnowsze)</option>
              <option value="date-asc">Data (najstarsze)</option>
              <option value="value-desc">Wartość (malejąco)</option>
              <option value="value-asc">Wartość (rosnąco)</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Readings List -->
      <div class="card">
        <h2 class="text-xl font-semibold mb-4">Ostatnie odczyty</h2>
        <div v-if="loadingReadings" class="text-center py-8">{{ $t('common.loading') }}</div>
        <div v-else-if="filteredReadings.length === 0" class="text-center py-8 text-gray-400">Brak odczytów</div>
        <div v-else class="space-y-3">
          <div v-for="reading in filteredReadings" :key="reading.id" class="flex justify-between items-center p-3 bg-gray-700 rounded hover:bg-gray-600 cursor-pointer transition-colors" @click="viewBill(reading.billId)">
            <div>
              <span class="font-medium">{{ formatMeterValue(reading.meterValue) }} {{ getUnitForBill(reading.billId) }}</span>
              <span class="text-gray-400 text-sm ml-4">{{ formatDateTime(reading.recordedAt) }}</span>
              <span v-if="getBillInfo(reading.billId)" class="text-blue-400 text-sm ml-4">
                {{ getBillInfo(reading.billId) }} →
              </span>
            </div>
            <span class="text-sm text-gray-400">{{ getUserName(reading.userId) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'
import {
  Plus, FileX, Zap, Flame, Wifi, Calendar, Receipt, Gauge,
  Send, Check, X, AlertCircle, Trash2, ChevronDown, ChevronUp
} from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const activeTab = ref('bills')

// Bills state
const bills = ref([])
const loading = ref(false)
const showCreateModal = ref(false)
const creating = ref(false)
const createError = ref('')
const billAllocations = ref({}) // Store allocations by bill ID
const loadingAllocations = ref({})
const expandedBills = ref({})

const filters = ref({
  type: '',
  dateFrom: '',
  dateTo: '',
  sortBy: 'date-desc'
})

// Readings state
const allBills = ref([])
const postedBills = ref([])
const readings = ref([])
const users = ref([])
const loadingReadings = ref(false)
const loadingReading = ref(false)

const readingFilters = ref({
  billId: '',
  userId: '',
  sortBy: 'date-desc'
})

const form = ref({
  billId: '',
  meterReading: '',
  readingDate: new Date().toISOString().slice(0, 16)
})

const filteredBills = computed(() => {
  let result = [...bills.value].filter(b => b && b.id) // Filter out undefined/null bills

  if (filters.value.type) {
    result = result.filter(b => b.type === filters.value.type)
  }

  if (filters.value.dateFrom) {
    const fromDate = new Date(filters.value.dateFrom)
    result = result.filter(b => new Date(b.periodEnd) >= fromDate)
  }
  if (filters.value.dateTo) {
    const toDate = new Date(filters.value.dateTo)
    result = result.filter(b => new Date(b.periodStart) <= toDate)
  }

  result.sort((a, b) => {
    switch (filters.value.sortBy) {
      case 'date-desc':
        return new Date(b.periodEnd) - new Date(a.periodEnd)
      case 'date-asc':
        return new Date(a.periodEnd) - new Date(b.periodEnd)
      case 'amount-desc':
        return b.totalAmountPLN - a.totalAmountPLN
      case 'amount-asc':
        return a.totalAmountPLN - b.totalAmountPLN
      default:
        return 0
    }
  })

  return result
})

// Check if we have any metered bills (electricity or metered 'inne')
const hasMeteredBills = computed(() => {
  return bills.value.some(b =>
    b.type === 'electricity' ||
    (b.type === 'inne' && b.allocationType === 'metered')
  )
})

const filteredReadings = computed(() => {
  let result = [...readings.value]

  if (readingFilters.value.billId) {
    result = result.filter(r => r.billId === readingFilters.value.billId)
  }

  if (readingFilters.value.userId) {
    result = result.filter(r => r.userId === readingFilters.value.userId)
  }

  result.sort((a, b) => {
    switch (readingFilters.value.sortBy) {
      case 'date-desc':
        return new Date(b.recordedAt) - new Date(a.recordedAt)
      case 'date-asc':
        return new Date(a.recordedAt) - new Date(b.recordedAt)
      case 'value-desc':
        return b.meterValue - a.meterValue
      case 'value-asc':
        return a.meterValue - b.meterValue
      default:
        return 0
    }
  })

  return result
})

const newBill = ref({
  type: 'electricity',
  customType: '',
  allocationType: 'simple',
  totalAmountPLN: '',
  totalUnits: '',
  periodStart: '',
  periodEnd: '',
  paymentDeadline: '',
  notes: ''
})

onMounted(async () => {
  await loadBills()
  await loadReadingsData()
})

async function loadBills() {
  loading.value = true
  try {
    const response = await api.get('/bills')
    bills.value = response.data || []
  } catch (err) {
    console.error('Failed to load bills:', err)
    bills.value = []
  } finally {
    loading.value = false
  }
}

async function loadReadingsData() {
  loadingReadings.value = true
  try {
    const billsRes = await api.get('/bills?status=posted')
    postedBills.value = (billsRes.data || []).filter(b =>
      b.type === 'electricity' ||
      (b.type === 'inne' && b.allocationType === 'metered')
    )

    const allBillsRes = await api.get('/bills')
    allBills.value = allBillsRes.data || []

    const readingsRes = await api.get('/consumptions')
    readings.value = readingsRes.data || []

    const usersRes = await api.get('/users')
    users.value = usersRes.data || []
  } catch (err) {
    console.error('Failed to load readings data:', err)
    postedBills.value = []
    allBills.value = []
    readings.value = []
    users.value = []
  } finally {
    loadingReadings.value = false
  }
}

async function createBill() {
  creating.value = true
  createError.value = ''

  try {
    // Validate dates
    const startDate = new Date(newBill.value.periodStart)
    const endDate = new Date(newBill.value.periodEnd)

    if (isNaN(startDate.getTime()) || isNaN(endDate.getTime())) {
      createError.value = 'Nieprawidłowe daty'
      creating.value = false
      return
    }

    if (startDate.getFullYear() < 2000 || startDate.getFullYear() > 2100) {
      createError.value = 'Data rozpoczęcia musi być między 2000 a 2100'
      creating.value = false
      return
    }

    if (endDate.getFullYear() < 2000 || endDate.getFullYear() > 2100) {
      createError.value = 'Data zakończenia musi być między 2000 a 2100'
      creating.value = false
      return
    }

    if (endDate <= startDate) {
      createError.value = 'Data zakończenia musi być późniejsza niż data rozpoczęcia'
      creating.value = false
      return
    }

    const payload = {
      type: newBill.value.type,
      totalAmountPLN: newBill.value.totalAmountPLN,
      totalUnits: newBill.value.totalUnits || undefined,
      periodStart: startDate.toISOString(),
      periodEnd: endDate.toISOString(),
      notes: newBill.value.notes || undefined
    }

    if (newBill.value.type === 'inne' && newBill.value.customType) {
      payload.customType = newBill.value.customType
      payload.allocationType = newBill.value.allocationType
    }

    await api.post('/bills', payload)
    showCreateModal.value = false
    await loadBills()
    await loadReadingsData()
    newBill.value = {
      type: 'electricity',
      customType: '',
      allocationType: 'simple',
      totalAmountPLN: '',
      totalUnits: '',
      periodStart: '',
      periodEnd: '',
      notes: ''
    }
  } catch (err) {
    createError.value = err.response?.data?.error || 'Nie udało się utworzyć rachunku'
  } finally {
    creating.value = false
  }
}

async function postBill(billId) {
  try {
    await api.post(`/bills/${billId}/post`)
    await loadBills()
    await loadReadingsData()
  } catch (err) {
    console.error('Failed to post bill:', err)
  }
}

async function closeBill(billId) {
  try {
    await api.post(`/bills/${billId}/close`)
    await loadBills()
  } catch (err) {
    console.error('Failed to close bill:', err)
  }
}

async function deleteBill(billId) {
  if (!confirm('Czy na pewno chcesz usunąć ten rachunek? To usunie również wszystkie powiązane odczyty.')) {
    return
  }

  try {
    await api.delete(`/bills/${billId}`)
    await loadBills()
    await loadReadingsData()
  } catch (err) {
    console.error('Failed to delete bill:', err)
    alert('Błąd podczas usuwania: ' + (err.response?.data?.error || err.message))
  }
}

async function submitReading() {
  loadingReading.value = true
  try {
    await api.post('/consumptions', {
      billId: form.value.billId,
      meterValue: parseFloat(form.value.meterReading),
      units: 0,
      recordedAt: new Date(form.value.readingDate).toISOString()
    })

    form.value.meterReading = ''
    await loadReadingsData()
  } catch (err) {
    console.error('Failed to submit reading:', err)
  } finally {
    loadingReading.value = false
  }
}

function formatMoney(decimal128) {
  if (!decimal128) return '0.00'
  if (typeof decimal128 === 'number') return decimal128.toFixed(2)
  return parseFloat(decimal128.$numberDecimal || decimal128 || 0).toFixed(2)
}

function formatUnits(decimal128) {
  if (!decimal128) return '0.000'
  if (typeof decimal128 === 'number') return decimal128.toFixed(3)
  return parseFloat(decimal128.$numberDecimal || decimal128 || 0).toFixed(3)
}

function formatMeterValue(value) {
  if (!value) return '0.000'
  const numValue = parseFloat(value.$numberDecimal || value || 0)
  return numValue.toFixed(3)
}

function formatDate(date) {
  if (!date) return '-'
  return new Date(date).toLocaleDateString('pl-PL', { day: 'numeric', month: 'short', year: 'numeric' })
}

async function loadBillAllocation(billId) {
  if (billAllocations.value[billId]) {
    // Already loaded, just return
    return
  }

  loadingAllocations.value[billId] = true
  try {
    const response = await api.get(`/bills/${billId}/allocation`)
    billAllocations.value[billId] = response.data
  } catch (err) {
    console.error('Failed to load allocation:', err)
    billAllocations.value[billId] = []
  } finally {
    loadingAllocations.value[billId] = false
  }
}

async function toggleBillExpansion(billId) {
  if (expandedBills.value[billId]) {
    expandedBills.value[billId] = false
  } else {
    expandedBills.value[billId] = true
    await loadBillAllocation(billId)
  }
}

function formatDateTime(date) {
  if (!date) return '-'
  return new Date(date).toLocaleString('pl-PL')
}

function getUnit(type) {
  if (type === 'electricity') return 'kWh'
  if (type === 'gas') return 'm³'
  return ''
}

function getUnitForBill(billId) {
  const bill = allBills.value.find(b => b.id === billId)
  if (!bill) return 'jednostek'
  return getUnit(bill.type) || 'jednostek'
}

function getUserName(userId) {
  const user = users.value.find(u => u.id === userId)
  return user ? user.name : 'Nieznany'
}

function getBillInfo(billId) {
  const bill = allBills.value.find(b => b.id === billId)
  if (!bill) return ''

  const typeLabel = bill.type === 'electricity' ? 'Prąd' :
                    bill.type === 'gas' ? 'Gaz' : bill.type
  const dateRange = `${formatDate(bill.periodStart)} - ${formatDate(bill.periodEnd)}`
  return `${typeLabel}: ${dateRange}`
}

function viewBill(billId) {
  router.push(`/bills/${billId}`)
}
</script>
