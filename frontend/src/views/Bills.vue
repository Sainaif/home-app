<template>
  <div>
    <div class="flex justify-between items-center mb-8">
      <div>
        <h1 class="text-4xl font-bold gradient-text mb-2">{{ $t('bills.title') }}</h1>
        <p class="text-gray-400">Historia rachunków i alokacji kosztów</p>
      </div>
      <button v-if="authStore.isAdmin" @click="showCreateModal = true" class="btn btn-primary flex items-center gap-2">
        <Plus class="w-5 h-5" />
        {{ $t('bills.createNew') }}
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

          <div>
            <label class="block text-sm font-medium mb-2">Kwota (PLN)</label>
            <input v-model.number="newBill.totalAmountPLN" type="number" step="0.01" required class="input" placeholder="150.00" />
          </div>

          <div v-if="newBill.type === 'electricity' || newBill.type === 'gas'">
            <label class="block text-sm font-medium mb-2">Jednostki</label>
            <input v-model.number="newBill.totalUnits" type="number" step="0.001" class="input" placeholder="100.000" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-2">Okres od</label>
              <input v-model="newBill.periodStart" type="date" required class="input" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-2">Okres do</label>
              <input v-model="newBill.periodEnd" type="date" required class="input" />
            </div>
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
      <div v-else class="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>{{ $t('bills.type') }}</th>
              <th>{{ $t('bills.period') }}</th>
              <th>{{ $t('bills.amount') }}</th>
              <th>{{ $t('bills.totalUnits') }}</th>
              <th>{{ $t('bills.status') }}</th>
              <th v-if="authStore.isAdmin">{{ $t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="bill in filteredBills" :key="bill.id">
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
                <span class="font-bold text-purple-400">{{ formatMoney(bill.totalAmountPLN) }} PLN</span>
              </td>
              <td>
                <span class="text-gray-300">{{ bill.totalUnits ? formatUnits(bill.totalUnits) + ' ' + getUnit(bill.type) : '-' }}</span>
              </td>
              <td>
                <span :class="`badge badge-${bill.status}`">
                  {{ $t(`bills.${bill.status}`) }}
                </span>
              </td>
              <td v-if="authStore.isAdmin">
                <div class="flex items-center gap-2">
                  <button v-if="bill.status === 'draft'" @click="allocateBill(bill.id)"
                          class="btn btn-sm btn-outline flex items-center gap-1">
                    <PieChart class="w-3 h-3" />
                    {{ $t('bills.allocate') }}
                  </button>
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
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'
import {
  Plus, FileX, Zap, Flame, Wifi, Users, Calendar,
  PieChart, Send, Check, X, AlertCircle, Trash2
} from 'lucide-vue-next'

const authStore = useAuthStore()
const bills = ref([])
const loading = ref(false)
const showCreateModal = ref(false)
const creating = ref(false)
const createError = ref('')

const filters = ref({
  type: '',
  dateFrom: '',
  dateTo: '',
  sortBy: 'date-desc'
})

const filteredBills = computed(() => {
  let result = [...bills.value]

  // Filter by type
  if (filters.value.type) {
    result = result.filter(b => b.type === filters.value.type)
  }

  // Filter by date range
  if (filters.value.dateFrom) {
    const fromDate = new Date(filters.value.dateFrom)
    result = result.filter(b => new Date(b.periodEnd) >= fromDate)
  }
  if (filters.value.dateTo) {
    const toDate = new Date(filters.value.dateTo)
    result = result.filter(b => new Date(b.periodStart) <= toDate)
  }

  // Sort
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

const newBill = ref({
  type: 'electricity',
  customType: '',
  totalAmountPLN: '',
  totalUnits: '',
  periodStart: '',
  periodEnd: '',
  notes: ''
})

onMounted(loadBills)

async function loadBills() {
  loading.value = true
  try {
    const response = await api.get('/bills')
    console.log('Bills response:', response.data)
    bills.value = response.data || []
  } catch (err) {
    console.error('Failed to load bills:', err)
    console.error('Error response:', err.response)
    bills.value = []
  } finally {
    loading.value = false
  }
}

async function createBill() {
  creating.value = true
  createError.value = ''

  try {
    const payload = {
      type: newBill.value.type,
      totalAmountPLN: newBill.value.totalAmountPLN,
      totalUnits: newBill.value.totalUnits || undefined,
      periodStart: new Date(newBill.value.periodStart).toISOString(),
      periodEnd: new Date(newBill.value.periodEnd).toISOString(),
      notes: newBill.value.notes || undefined
    }

    if (newBill.value.type === 'inne' && newBill.value.customType) {
      payload.customType = newBill.value.customType
    }

    console.log('Creating bill with payload:', payload)
    const response = await api.post('/bills', payload)
    console.log('Bill created:', response.data)

    showCreateModal.value = false
    await loadBills()
    newBill.value = {
      type: 'electricity',
      customType: '',
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

async function allocateBill(billId) {
  try {
    await api.post(`/bills/${billId}/allocate`)
    await loadBills()
    alert('Rachunek zaalokowany pomyślnie')
  } catch (err) {
    console.error('Failed to allocate bill:', err)
    alert('Błąd podczas alokacji: ' + (err.response?.data?.error || err.message))
  }
}

async function deleteBill(billId) {
  if (!confirm('Czy na pewno chcesz usunąć ten rachunek? To usunie również wszystkie powiązane odczyty.')) {
    return
  }

  try {
    await api.delete(`/bills/${billId}`)
    await loadBills()
    alert('Rachunek usunięty')
  } catch (err) {
    console.error('Failed to delete bill:', err)
    alert('Błąd podczas usuwania: ' + (err.response?.data?.error || err.message))
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

function formatDate(date) {
  return new Date(date).toLocaleDateString('pl-PL', { day: 'numeric', month: 'short', year: 'numeric' })
}

function getUnit(type) {
  if (type === 'electricity') return 'kWh'
  if (type === 'gas') return 'm³'
  return ''
}
</script>