<template>
  <div>
    <h1 class="text-3xl font-bold mb-8">{{ $t('readings.title') }}</h1>

    <div class="card mb-6">
      <h2 class="text-xl font-semibold mb-4">{{ $t('readings.addReading') }}</h2>
      <form @submit.prevent="submitReading" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('readings.bill') }}</label>
            <select v-model="form.billId" required class="input">
              <option value="">Wybierz rachunek</option>
              <option v-for="bill in draftBills" :key="bill.id" :value="bill.id">
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

        <button type="submit" :disabled="loading" class="btn btn-primary">
          {{ $t('readings.submit') }}
        </button>
      </form>
    </div>

    <div class="card">
      <h2 class="text-xl font-semibold mb-4">Ostatnie odczyty</h2>
      <div v-if="loadingReadings" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else-if="readings.length === 0" class="text-center py-8 text-gray-400">Brak odczytów</div>
      <div v-else class="space-y-3">
        <div v-for="reading in readings" :key="reading.id" class="flex justify-between items-center p-3 bg-gray-700 rounded hover:bg-gray-600 cursor-pointer transition-colors" @click="viewBill(reading.billId)">
          <div>
            <span class="font-medium">{{ formatMeterValue(reading.meterValue) }} {{ getUnit(reading.billId) }}</span>
            <span class="text-gray-400 text-sm ml-4">{{ formatDateTime(reading.recordedAt) }}</span>
            <span v-if="getBillInfo(reading.billId)" class="text-blue-400 text-sm ml-4">
              {{ getBillInfo(reading.billId) }} →
            </span>
          </div>
          <span class="text-sm text-gray-400">{{ reading.source || 'użytkownik' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api/client'

const router = useRouter()

const draftBills = ref([])
const allBills = ref([])
const readings = ref([])
const loading = ref(false)
const loadingReadings = ref(false)

const form = ref({
  billId: '',
  meterReading: '',
  readingDate: new Date().toISOString().slice(0, 16)
})

onMounted(async () => {
  loadingReadings.value = true
  try {
    // Load only posted bills for readings (zamieszczone)
    const billsRes = await api.get('/bills?status=posted')
    draftBills.value = (billsRes.data || []).filter(b => b.type === 'electricity' || b.type === 'gas')

    const allBillsRes = await api.get('/bills')
    allBills.value = allBillsRes.data || []

    const readingsRes = await api.get('/consumptions')
    readings.value = readingsRes.data || []
  } catch (err) {
    console.error('Failed to load data:', err)
    draftBills.value = []
    allBills.value = []
    readings.value = []
  } finally {
    loadingReadings.value = false
  }
})

async function submitReading() {
  loading.value = true
  try {
    await api.post('/consumptions', {
      billId: form.value.billId,
      meterValue: parseFloat(form.value.meterReading),
      units: 0, // Units will be calculated by backend
      recordedAt: new Date(form.value.readingDate).toISOString()
    })

    form.value.meterReading = ''

    const readingsRes = await api.get('/consumptions')
    readings.value = readingsRes.data || []
  } catch (err) {
    console.error('Failed to submit reading:', err)
  } finally {
    loading.value = false
  }
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

function getUnit(billId) {
  const bill = allBills.value.find(b => b.id === billId)
  if (!bill) return 'jednostek'

  if (bill.type === 'electricity') return 'kWh'
  if (bill.type === 'gas') return 'm³'
  return 'jednostek'
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