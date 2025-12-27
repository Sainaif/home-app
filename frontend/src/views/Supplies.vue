<template>
  <div>
    <div class="flex items-center justify-between mb-8">
      <h1 class="text-3xl font-bold">{{ $t('supplies.title') }}</h1>
      <div class="flex gap-2">
        <button
          v-if="authStore.hasPermission('reminders.send')"
          @click="sendLowSuppliesReminder"
          :disabled="sendingLowSuppliesReminder"
          class="btn btn-secondary flex items-center gap-2">
          <svg v-if="!sendingLowSuppliesReminder" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/>
            <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>
          </svg>
          <svg v-else class="w-4 h-4 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          {{ $t('supplies.remindLowStock') }}
        </button>
        <button v-if="authStore.isAdmin" @click="showSettingsModal = true" class="btn btn-outline flex items-center gap-2">
          <Settings class="w-4 h-4" />
          {{ $t('supplies.settings') }}
        </button>
      </div>
    </div>

    <!-- Budget Overview Card -->
    <div class="card mb-6">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm text-gray-400 mb-1">{{ $t('supplies.currentBudget') }}</p>
          <p class="text-3xl font-bold gradient-text">{{ formatMoney(settings?.currentBudgetPLN) }} PLN</p>
        </div>
        <div class="text-right">
          <p class="text-sm text-gray-400 mb-1">{{ $t('supplies.weeklyContribution') }}</p>
          <p class="text-lg font-semibold">{{ formatMoney(settings?.weeklyContributionPLN) }} PLN / {{ $t('supplies.perPerson') }}</p>
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <div class="flex gap-2 mb-6 overflow-x-auto">
      <button
        @click="activeTab = 'inventory'"
        :class="['btn', activeTab === 'inventory' ? 'btn-primary' : 'btn-outline']">
        <Package class="w-4 h-4 mr-2" />
        {{ $t('supplies.inventory') }}
      </button>
      <button
        @click="activeTab = 'refunds'"
        :class="['btn', activeTab === 'refunds' ? 'btn-primary' : 'btn-outline']">
        <DollarSign class="w-4 h-4 mr-2" />
        {{ $t('supplies.refunds') }}
        <span v-if="refundCount > 0" class="ml-1 bg-red-600 text-white text-xs px-2 py-0.5 rounded-full">{{ refundCount }}</span>
      </button>
      <button
        @click="activeTab = 'stats'"
        :class="['btn', activeTab === 'stats' ? 'btn-primary' : 'btn-outline']">
        <TrendingUp class="w-4 h-4 mr-2" />
        {{ $t('supplies.stats') }}
      </button>
    </div>

    <!-- Inventory Tab -->
    <div v-if="activeTab === 'inventory'">
      <!-- Filters and Sorting -->
      <div class="card mb-6">
        <div class="flex flex-wrap gap-3 items-center">
          <select v-model="selectedFilter" class="input flex-1 min-w-[150px]">
            <option value="">{{ $t('supplies.filters.all') }}</option>
            <option value="low_stock">{{ $t('supplies.filters.lowStock') }}</option>
            <option value="needs_refund">{{ $t('supplies.filters.needsRefund') }}</option>
          </select>

          <select v-model="selectedSort" class="input flex-1 min-w-[150px]">
            <option value="">{{ $t('supplies.sorting.default') }}</option>
            <option value="name">{{ $t('supplies.sorting.name') }}</option>
            <option value="quantity_asc">{{ $t('supplies.sorting.quantityLow') }}</option>
            <option value="recently_restocked">{{ $t('supplies.sorting.recentlyRestocked') }}</option>
          </select>

          <input
            v-model="searchQuery"
            type="text"
            :placeholder="$t('supplies.search')"
            class="input flex-1 min-w-[200px]" />
        </div>
      </div>

      <!-- Add Item -->
      <div class="card mb-6">
        <h3 class="text-lg font-semibold mb-4">{{ $t('supplies.addItem') }}</h3>
        <form @submit.prevent="addItem" class="space-y-3">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <input
              v-model="newItem.name"
              type="text"
              :placeholder="$t('supplies.itemName')"
              required
              class="input" />

            <select v-model="newItem.category" class="input">
              <option value="groceries">{{ $t('supplies.categories.groceries') }}</option>
              <option value="cleaning">{{ $t('supplies.categories.cleaning') }}</option>
              <option value="toiletries">{{ $t('supplies.categories.toiletries') }}</option>
              <option value="other">{{ $t('supplies.categories.other') }}</option>
            </select>
          </div>

          <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <input
              v-model.number="newItem.currentQuantity"
              type="number"
              min="0"
              :placeholder="$t('supplies.currentQty')"
              required
              class="input" />

            <input
              v-model.number="newItem.minQuantity"
              type="number"
              min="0"
              :placeholder="$t('supplies.minQty')"
              required
              class="input" />

            <select v-model="newItem.unit" class="input">
              <option value="pcs">{{ $t('supplies.units.pcs') }}</option>
              <option value="kg">{{ $t('supplies.units.kg') }}</option>
              <option value="L">{{ $t('supplies.units.L') }}</option>
              <option value="bottles">{{ $t('supplies.units.bottles') }}</option>
              <option value="boxes">{{ $t('supplies.units.boxes') }}</option>
              <option value="rolls">{{ $t('supplies.units.rolls') }}</option>
              <option value="bags">{{ $t('supplies.units.bags') }}</option>
              <option value="jars">{{ $t('supplies.units.jars') }}</option>
              <option value="cans">{{ $t('supplies.units.cans') }}</option>
            </select>

            <button type="submit" :disabled="loadingItems" class="btn btn-primary flex items-center justify-center gap-2 font-semibold">
              <Plus class="w-5 h-5" />
              {{ $t('supplies.add') }}
            </button>
          </div>
        </form>
      </div>

      <!-- Items List -->
      <div v-if="loadingItems && items.length === 0" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else-if="filteredItems.length === 0" class="text-center py-8 text-gray-400">{{ $t('supplies.noItems') }}</div>
      <div v-else class="space-y-3">
        <div
          v-for="item in filteredItems"
          :key="item.id"
          class="card">
          <div class="flex items-start justify-between gap-4">
            <!-- Item Info -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <h3 class="font-semibold text-lg truncate">{{ item.name }}</h3>
                <span :class="['text-xs px-2 py-0.5 rounded-full', getCategoryColor(item.category)]">
                  {{ $t(`supplies.categories.${item.category}`) }}
                </span>
                <AlertCircle v-if="isLowStock(item)" class="w-5 h-5 text-red-500" :title="$t('supplies.lowStockWarning')" />
                <DollarSign v-if="item.needsRefund" class="w-5 h-5 text-yellow-500" :title="$t('supplies.needsRefund')" />
              </div>

              <!-- Quantity Display -->
              <div class="flex items-center gap-4 mb-2">
                <div class="flex items-center gap-2">
                  <span class="text-3xl font-bold">{{ item.currentQuantity }}</span>
                  <span class="text-gray-400">{{ $t(`supplies.units.${item.unit}`) }}</span>
                </div>
                <span class="text-sm text-gray-400">
                  {{ $t('supplies.min') }}: {{ item.minQuantity }} {{ $t(`supplies.units.${item.unit}`) }}
                </span>
              </div>

              <!-- Item History -->
              <div class="text-sm text-gray-400 space-y-1">
                <div>{{ $t('supplies.addedBy') }}: {{ getUserName(item.addedByUserId) }} • {{ formatDate(item.addedAt) }}</div>
                <div v-if="item.lastRestockedAt">
                  {{ $t('supplies.lastRestock') }}: {{ getUserName(item.lastRestockedByUserId) }} •
                  {{ formatDate(item.lastRestockedAt) }}
                  <span v-if="item.lastRestockAmountPLN">• {{ formatMoney(item.lastRestockAmountPLN) }} PLN</span>
                </div>
              </div>
            </div>

            <!-- Action Buttons -->
            <div class="flex flex-col gap-2 flex-shrink-0">
              <button @click="openRestockModal(item)" class="btn btn-sm btn-primary flex items-center gap-1">
                <Plus class="w-4 h-4" />
                {{ $t('supplies.restock') }}
              </button>

              <button
                @click="consumeItem(item)"
                :disabled="item.currentQuantity === 0"
                class="btn btn-sm btn-outline flex items-center gap-1">
                <Minus class="w-4 h-4" />
                {{ $t('supplies.consume') }}
              </button>

              <button @click="openEditModal(item)" class="btn btn-sm btn-outline flex items-center gap-1">
                <Edit class="w-4 h-4" />
                {{ $t('common.edit') }}
              </button>

              <button @click="deleteItem(item.id)" class="btn btn-sm btn-secondary flex items-center gap-1">
                <Trash class="w-4 h-4" />
                {{ $t('common.delete') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Refunds Tab -->
    <div v-if="activeTab === 'refunds'">
      <div v-if="loadingItems" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else-if="itemsNeedingRefund.length === 0" class="text-center py-8 text-gray-400">
        {{ $t('supplies.noRefunds') }}
      </div>
      <div v-else class="space-y-3">
        <div
          v-for="item in itemsNeedingRefund"
          :key="item.id"
          class="card flex items-center justify-between">
          <div class="flex-1">
            <div class="flex items-center gap-2 mb-2">
              <h3 class="font-semibold text-lg">{{ item.name }}</h3>
              <span :class="['text-xs px-2 py-0.5 rounded-full', getCategoryColor(item.category)]">
                {{ $t(`supplies.categories.${item.category}`) }}
              </span>
            </div>
            <div class="text-sm text-gray-400">
              {{ $t('supplies.restockedBy') }}: {{ getUserName(item.lastRestockedByUserId) }} •
              {{ formatDate(item.lastRestockedAt) }}
            </div>
            <div class="text-lg font-semibold text-yellow-400 mt-1">
              {{ formatMoney(item.lastRestockAmountPLN) }} PLN
            </div>
          </div>

          <button
            v-if="authStore.isAdmin"
            @click="markAsRefunded(item.id)"
            class="btn btn-primary ml-4">
            {{ $t('supplies.markRefunded') }}
          </button>
        </div>

        <!-- Total Pending -->
        <div class="card bg-gray-800/50">
          <div class="flex items-center justify-between">
            <span class="text-lg font-semibold">{{ $t('supplies.totalPending') }}:</span>
            <span class="text-2xl font-bold text-yellow-400">{{ totalPendingRefunds }} PLN</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Stats Tab -->
    <div v-if="activeTab === 'stats'">
      <div v-if="loadingStats" class="text-center py-8">{{ $t('common.loading') }}</div>
      <div v-else class="space-y-6">
        <!-- Spending by Category -->
        <div class="card">
          <h3 class="text-lg font-semibold mb-4">{{ $t('supplies.spendingByCategory') }}</h3>
          <div v-if="stats?.byCategory && stats.byCategory.length > 0" class="space-y-3">
            <div v-for="cat in stats.byCategory" :key="cat._id" class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span :class="['w-3 h-3 rounded-full', getCategoryBgColor(cat._id)]"></span>
                <span class="capitalize">{{ $t(`supplies.categories.${cat._id}`) }}</span>
              </div>
              <span class="font-semibold">{{ formatMoney(cat.totalSpent) }} PLN</span>
            </div>
          </div>
          <p v-else class="text-gray-400">{{ $t('supplies.noData') }}</p>
        </div>

        <!-- Spending by User -->
        <div class="card">
          <h3 class="text-lg font-semibold mb-4">{{ $t('supplies.spendingByUser') }}</h3>
          <div v-if="stats?.byUser && stats.byUser.length > 0" class="space-y-3">
            <div v-for="user in stats.byUser" :key="user._id" class="flex items-center justify-between">
              <span>{{ getUserName(user._id) }}</span>
              <span class="font-semibold">{{ formatMoney(user.totalSpent) }} PLN</span>
            </div>
          </div>
          <p v-else class="text-gray-400">{{ $t('supplies.noData') }}</p>
        </div>

        <!-- Recent Contributions -->
        <div class="card">
          <h3 class="text-lg font-semibold mb-4">{{ $t('supplies.recentContributions') }}</h3>
          <div v-if="stats?.recentContributions && stats.recentContributions.length > 0" class="space-y-3">
            <div v-for="contrib in stats.recentContributions" :key="contrib.id" class="flex items-center justify-between">
              <div>
                <p>{{ getUserName(contrib.userId) }}</p>
                <p class="text-sm text-gray-400">{{ formatDate(contrib.createdAt) }}</p>
              </div>
              <span class="font-semibold text-green-400">+{{ formatMoney(contrib.amountPLN) }} PLN</span>
            </div>
          </div>
          <p v-else class="text-gray-400">{{ $t('supplies.noData') }}</p>
        </div>
      </div>
    </div>

    <!-- Settings Modal (Admin Only) -->
    <div v-if="showSettingsModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showSettingsModal = false">
      <div class="card max-w-lg w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold">{{ $t('supplies.settingsTitle') }}</h2>
          <button @click="showSettingsModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="saveSettings" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('supplies.weeklyContributionPerPerson') }}</label>
            <input
              v-model.number="settingsForm.weeklyContributionPLN"
              type="number"
              step="0.01"
              required
              class="input" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('supplies.contributionDay') }}</label>
            <select v-model="settingsForm.contributionDay" class="input">
              <option value="monday">{{ $t('supplies.days.monday') }}</option>
              <option value="tuesday">{{ $t('supplies.days.tuesday') }}</option>
              <option value="wednesday">{{ $t('supplies.days.wednesday') }}</option>
              <option value="thursday">{{ $t('supplies.days.thursday') }}</option>
              <option value="friday">{{ $t('supplies.days.friday') }}</option>
              <option value="saturday">{{ $t('supplies.days.saturday') }}</option>
              <option value="sunday">{{ $t('supplies.days.sunday') }}</option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('supplies.manualAdjustment') }}</label>
            <div class="flex gap-2">
              <input
                v-model.number="budgetAdjustment"
                type="number"
                step="0.01"
                placeholder="0.00"
                class="input flex-1" />
              <button type="button" @click="adjustBudget" class="btn btn-outline">
                {{ $t('supplies.adjust') }}
              </button>
            </div>
            <p class="text-xs text-gray-400 mt-1">{{ $t('supplies.adjustmentHint') }}</p>
          </div>

          <div v-if="settingsError" class="text-red-500 text-sm">{{ settingsError }}</div>

          <div class="flex gap-3">
            <button type="submit" :disabled="savingSettings" class="btn btn-primary flex-1">
              {{ savingSettings ? $t('common.saving') : $t('common.save') }}
            </button>
            <button type="button" @click="showSettingsModal = false" class="btn btn-outline">
              {{ $t('common.cancel') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Restock Modal -->
    <div v-if="showRestockModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showRestockModal = false">
      <div class="card max-w-md w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-xl font-bold">{{ $t('supplies.restockItem') }}: {{ selectedItem?.name }}</h2>
          <button @click="showRestockModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="confirmRestock" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('supplies.quantityToAdd') }}</label>
            <input
              v-model.number="restockForm.quantityToAdd"
              type="number"
              min="1"
              required
              class="input"
              placeholder="1" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('supplies.amountSpent') }} (PLN) - {{ $t('common.optional') }}</label>
            <input
              v-model.number="restockForm.amountPLN"
              type="number"
              step="0.01"
              min="0"
              class="input"
              placeholder="0.00" />
          </div>

          <div class="flex items-center gap-2">
            <input
              v-model="restockForm.needsRefund"
              type="checkbox"
              id="needsRefund"
              class="w-5 h-5 rounded border-gray-600 text-purple-600 focus:ring-purple-600 focus:ring-offset-gray-800" />
            <label for="needsRefund" class="text-sm">{{ $t('supplies.needsRefundCheck') }}</label>
          </div>

          <div class="flex gap-3">
            <button type="submit" :disabled="restocking" class="btn btn-primary flex-1">
              {{ restocking ? $t('common.saving') : $t('common.confirm') }}
            </button>
            <button type="button" @click="showRestockModal = false" class="btn btn-outline">
              {{ $t('common.cancel') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit Item Modal -->
    <div v-if="showEditModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showEditModal = false">
      <div class="card max-w-md w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-xl font-bold">{{ $t('supplies.editItem') }}</h2>
          <button @click="showEditModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="confirmEdit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('supplies.itemName') }}</label>
            <input
              v-model="editForm.name"
              type="text"
              required
              class="input" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('supplies.category') }}</label>
            <select v-model="editForm.category" class="input">
              <option value="groceries">{{ $t('supplies.categories.groceries') }}</option>
              <option value="cleaning">{{ $t('supplies.categories.cleaning') }}</option>
              <option value="toiletries">{{ $t('supplies.categories.toiletries') }}</option>
              <option value="other">{{ $t('supplies.categories.other') }}</option>
            </select>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('supplies.minQty') }}</label>
              <input
                v-model.number="editForm.minQuantity"
                type="number"
                min="0"
                required
                class="input" />
            </div>

            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('supplies.unit') }}</label>
              <select v-model="editForm.unit" class="input">
                <option value="pcs">{{ $t('supplies.units.pcs') }}</option>
                <option value="kg">{{ $t('supplies.units.kg') }}</option>
                <option value="L">{{ $t('supplies.units.L') }}</option>
                <option value="bottles">{{ $t('supplies.units.bottles') }}</option>
                <option value="boxes">{{ $t('supplies.units.boxes') }}</option>
                <option value="rolls">{{ $t('supplies.units.rolls') }}</option>
                <option value="bags">{{ $t('supplies.units.bags') }}</option>
                <option value="jars">{{ $t('supplies.units.jars') }}</option>
                <option value="cans">{{ $t('supplies.units.cans') }}</option>
              </select>
            </div>
          </div>

          <div class="flex gap-3">
            <button type="submit" :disabled="editing" class="btn btn-primary flex-1">
              {{ editing ? $t('common.saving') : $t('common.save') }}
            </button>
            <button type="button" @click="showEditModal = false" class="btn btn-outline">
              {{ $t('common.cancel') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useEventStream } from '../composables/useEventStream'
import { useDataEvents, DATA_EVENTS } from '../composables/useDataEvents'
import api from '../api/client'
import { Package, DollarSign, TrendingUp, Plus, Minus, Trash, Settings, X, Edit, AlertCircle } from 'lucide-vue-next'

const { t } = useI18n()
const authStore = useAuthStore()
const { connect, on: onEvent } = useEventStream()
const { on, emit } = useDataEvents()

const activeTab = ref('inventory')
const items = ref([])
const stats = ref(null)
const settings = ref(null)
const users = ref([])

const loadingItems = ref(false)
const loadingStats = ref(false)
const loadingSettings = ref(false)

const selectedFilter = ref('')
const selectedSort = ref('')
const searchQuery = ref('')

const showSettingsModal = ref(false)
const showRestockModal = ref(false)
const showEditModal = ref(false)
const selectedItem = ref(null)

const newItem = ref({
  name: '',
  category: 'groceries',
  currentQuantity: 0,
  minQuantity: 1,
  unit: 'pcs'
})

const settingsForm = ref({
  weeklyContributionPLN: 10,
  contributionDay: 'monday'
})

const restockForm = ref({
  quantityToAdd: 1,
  amountPLN: null,
  needsRefund: false
})

const editForm = ref({
  name: '',
  category: '',
  minQuantity: 0,
  unit: ''
})

const budgetAdjustment = ref(0)
const savingSettings = ref(false)
const restocking = ref(false)
const editing = ref(false)
const settingsError = ref('')
const sendingLowSuppliesReminder = ref(false)

const filteredItems = computed(() => {
  let result = items.value

  // Apply search
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(item => item.name.toLowerCase().includes(query))
  }

  return result
})

const itemsNeedingRefund = computed(() => {
  return items.value.filter(item => item.needsRefund)
})

const refundCount = computed(() => itemsNeedingRefund.value.length)

const totalPendingRefunds = computed(() => {
  return itemsNeedingRefund.value.reduce((sum, item) => {
    const amount = item.lastRestockAmountPLN ? parseFloat(item.lastRestockAmountPLN.$numberDecimal || item.lastRestockAmountPLN || 0) : 0
    return sum + amount
  }, 0).toFixed(2)
})

onMounted(async () => {
  await Promise.all([
    loadSettings(),
    loadItems(),
    loadUsers()
  ])

  // Connect to WebSocket for real-time updates
  connect()

  // Listen for supply-related WebSocket events
  onEvent('supply.item.added', () => {
    console.log('[Supplies] Item added event received, refreshing...')
    loadItems()
  })

  onEvent('supply.item.bought', () => {
    console.log('[Supplies] Item bought event received, refreshing...')
    loadItems()
    loadStats()
  })

  onEvent('supply.budget.contributed', () => {
    console.log('[Supplies] Budget contributed event received, refreshing...')
    loadStats()
  })

  onEvent('supply.budget.low', () => {
    console.log('[Supplies] Budget low event received, refreshing...')
    loadStats()
  })

  // Listen for local data events
  on(DATA_EVENTS.SUPPLY_ITEM_CREATED, loadItems)
  on(DATA_EVENTS.SUPPLY_ITEM_UPDATED, loadItems)
  on(DATA_EVENTS.SUPPLY_ITEM_DELETED, loadItems)
  on(DATA_EVENTS.SUPPLY_CONTRIBUTION_CREATED, loadStats)
  on(DATA_EVENTS.USER_UPDATED, loadUsers)
})

async function loadSettings() {
  loadingSettings.value = true
  try {
    const response = await api.get('/supplies/settings')
    settings.value = response.data
    settingsForm.value = {
      weeklyContributionPLN: parseFloat(response.data.weeklyContributionPLN.$numberDecimal || response.data.weeklyContributionPLN || 10),
      contributionDay: response.data.contributionDay || 'monday'
    }
  } catch (err) {
    console.error('Failed to load supply settings:', err)
  } finally {
    loadingSettings.value = false
  }
}

async function loadItems() {
  loadingItems.value = true
  try {
    const params = {}
    if (selectedFilter.value) params.filter = selectedFilter.value
    if (selectedSort.value) params.sort = selectedSort.value

    const response = await api.get('/supplies/items', { params })
    items.value = response.data || []
  } catch (err) {
    console.error('Failed to load supply items:', err)
    items.value = []
  } finally {
    loadingItems.value = false
  }
}

async function loadStats() {
  loadingStats.value = true
  try {
    const response = await api.get('/supplies/stats')
    stats.value = response.data
  } catch (err) {
    console.error('Failed to load supply stats:', err)
  } finally {
    loadingStats.value = false
  }
}

async function loadUsers() {
  try {
    const response = await api.get('/users')
    users.value = response.data || []
  } catch (err) {
    console.error('Failed to load users:', err)
  }
}

async function addItem() {
  try {
    await api.post('/supplies/items', {
      name: newItem.value.name,
      category: newItem.value.category,
      currentQuantity: newItem.value.currentQuantity,
      minQuantity: newItem.value.minQuantity,
      unit: newItem.value.unit
    })

    newItem.value = {
      name: '',
      category: 'groceries',
      currentQuantity: 0,
      minQuantity: 1,
      unit: 'pcs'
    }

    await loadItems()
    emit(DATA_EVENTS.SUPPLY_ITEM_CREATED)
  } catch (err) {
    console.error('Failed to add item:', err)
    alert('Nie udało się dodać przedmiotu: ' + (err.response?.data?.error || err.message))
  }
}

function openRestockModal(item) {
  selectedItem.value = item
  restockForm.value = {
    quantityToAdd: 1,
    amountPLN: null,
    needsRefund: false
  }
  showRestockModal.value = true
}

async function confirmRestock() {
  if (!selectedItem.value) return

  restocking.value = true
  try {
    await api.post(`/supplies/items/${selectedItem.value.id}/restock`, {
      quantityToAdd: restockForm.value.quantityToAdd,
      amountPLN: restockForm.value.amountPLN || undefined,
      needsRefund: restockForm.value.needsRefund
    })

    showRestockModal.value = false
    selectedItem.value = null
    await Promise.all([loadItems(), loadSettings()])
    emit(DATA_EVENTS.SUPPLY_ITEM_UPDATED)
  } catch (err) {
    console.error('Failed to restock:', err)
    alert('Nie udało się uzupełnić zapasów: ' + (err.response?.data?.error || err.message))
  } finally {
    restocking.value = false
  }
}

async function consumeItem(item) {
  const quantity = prompt(`Ile ${item.unit} zużyto?`, '1')
  if (!quantity || quantity <= 0) return

  try {
    await api.post(`/supplies/items/${item.id}/consume`, {
      quantityToSubtract: parseInt(quantity)
    })

    await loadItems()
    emit(DATA_EVENTS.SUPPLY_ITEM_UPDATED)
  } catch (err) {
    console.error('Failed to consume:', err)
    alert('Nie udało się zmniejszyć ilości: ' + (err.response?.data?.error || err.message))
  }
}

function openEditModal(item) {
  selectedItem.value = item
  editForm.value = {
    name: item.name,
    category: item.category,
    minQuantity: item.minQuantity,
    unit: item.unit
  }
  showEditModal.value = true
}

async function confirmEdit() {
  if (!selectedItem.value) return

  editing.value = true
  try {
    await api.patch(`/supplies/items/${selectedItem.value.id}`, {
      name: editForm.value.name,
      category: editForm.value.category,
      minQuantity: editForm.value.minQuantity,
      unit: editForm.value.unit
    })

    showEditModal.value = false
    selectedItem.value = null
    await loadItems()
    emit(DATA_EVENTS.SUPPLY_ITEM_UPDATED)
  } catch (err) {
    console.error('Failed to edit item:', err)
    alert('Nie udało się edytować przedmiotu: ' + (err.response?.data?.error || err.message))
  } finally {
    editing.value = false
  }
}

async function deleteItem(itemId) {
  if (!confirm('Czy na pewno chcesz usunąć ten przedmiot?')) return

  try {
    await api.delete(`/supplies/items/${itemId}`)
    await loadItems()
    emit(DATA_EVENTS.SUPPLY_ITEM_DELETED)
  } catch (err) {
    console.error('Failed to delete item:', err)
    alert('Nie udało się usunąć przedmiotu: ' + (err.response?.data?.error || err.message))
  }
}

async function markAsRefunded(itemId) {
  if (!confirm('Czy na pewno chcesz oznaczyć jako zwrócone?')) return

  try {
    await api.post(`/supplies/items/${itemId}/refund`)
    await Promise.all([loadItems(), loadSettings()])
    emit(DATA_EVENTS.SUPPLY_ITEM_UPDATED)
  } catch (err) {
    console.error('Failed to mark as refunded:', err)
    alert('Nie udało się oznaczyć jako zwrócone: ' + (err.response?.data?.error || err.message))
  }
}

async function saveSettings() {
  savingSettings.value = true
  settingsError.value = ''

  try {
    await api.patch('/supplies/settings', {
      weeklyContributionPLN: settingsForm.value.weeklyContributionPLN,
      contributionDay: settingsForm.value.contributionDay
    })

    showSettingsModal.value = false
    await loadSettings()
  } catch (err) {
    console.error('Failed to save settings:', err)
    settingsError.value = err.response?.data?.error || 'Nie udało się zapisać ustawień'
  } finally {
    savingSettings.value = false
  }
}

async function adjustBudget() {
  if (!budgetAdjustment.value || budgetAdjustment.value === 0) return

  try {
    await api.post('/supplies/settings/adjust', {
      adjustment: budgetAdjustment.value
    })

    budgetAdjustment.value = 0
    await loadSettings()
  } catch (err) {
    console.error('Failed to adjust budget:', err)
    alert('Nie udało się dostosować budżetu: ' + (err.response?.data?.error || err.message))
  }
}

function getUserName(userId) {
  const user = users.value.find(u => u.id === userId)
  return user ? user.name : 'Nieznany'
}

function formatMoney(value) {
  if (!value) return '0.00'
  const num = typeof value === 'object' ? parseFloat(value.$numberDecimal || value) : parseFloat(value)
  return num.toFixed(2)
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('pl-PL', { day: 'numeric', month: 'short', year: 'numeric' })
}

function isLowStock(item) {
  return item.currentQuantity <= item.minQuantity
}

function getCategoryColor(category) {
  const colors = {
    groceries: 'bg-green-600/20 text-green-400',
    cleaning: 'bg-blue-600/20 text-blue-400',
    toiletries: 'bg-purple-600/20 text-purple-400',
    other: 'bg-gray-600/20 text-gray-400'
  }
  return colors[category] || colors.other
}

function getCategoryBgColor(category) {
  const colors = {
    groceries: 'bg-green-500',
    cleaning: 'bg-blue-500',
    toiletries: 'bg-purple-500',
    other: 'bg-gray-500'
  }
  return colors[category] || colors.other
}

// Watch filters and sorting to reload items
watch([selectedFilter, selectedSort], () => {
  loadItems()
})

// Watch activeTab and load stats when switched to stats tab
watch(activeTab, (newTab) => {
  if (newTab === 'stats' && !stats.value) {
    loadStats()
  }
})

async function sendLowSuppliesReminder() {
  sendingLowSuppliesReminder.value = true
  try {
    const response = await api.post('/reminders/supplies')
    const notifiedCount = response.data.notifiedCount || 0
    alert(t('supplies.reminderSent', { count: notifiedCount }))
  } catch (err) {
    console.error('Failed to send low supplies reminder:', err)
    alert(t('errors.sendReminderFailed') + ' ' + (err.response?.data?.error || err.message))
  } finally {
    sendingLowSuppliesReminder.value = false
  }
}
</script>
