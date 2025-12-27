<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="text-4xl font-bold gradient-text mb-2">{{ $t('bills.title') }}</h1>
        <p class="text-gray-400">{{ $t('bills.description') }}</p>
      </div>
      <button v-if="authStore.hasPermission('bills.create') && activeTab === 'bills'" @click="showCreateModal = true" class="btn btn-primary flex items-center gap-2">
        <Plus class="w-5 h-5" />
        {{ $t('bills.createNew') }}
      </button>
      <button v-if="authStore.hasPermission('bills.create') && activeTab === 'recurring'" @click="showRecurringModal = true" class="btn btn-primary flex items-center gap-2">
        <Plus class="w-5 h-5" />
        {{ $t('bills.createRecurring') }}
      </button>
    </div>

    <!-- Tabs -->
    <div class="flex gap-2 mb-6">
      <button
        @click="activeTab = 'bills'"
        :class="['btn', activeTab === 'bills' ? 'btn-primary' : 'btn-outline']"
        class="flex items-center gap-2">
        <Receipt class="w-4 h-4" />
        {{ $t('bills.tabBills') }}
      </button>
      <button
        v-if="hasMeteredBills"
        @click="activeTab = 'readings'"
        :class="['btn', activeTab === 'readings' ? 'btn-primary' : 'btn-outline']"
        class="flex items-center gap-2">
        <Gauge class="w-4 h-4" />
        {{ $t('bills.tabReadings') }}
      </button>
      <button
        v-if="authStore.hasPermission('bills.create')"
        @click="activeTab = 'recurring'"
        :class="['btn', activeTab === 'recurring' ? 'btn-primary' : 'btn-outline']"
        class="flex items-center gap-2">
        <Calendar class="w-4 h-4" />
        {{ $t('bills.tabRecurring') }}
      </button>
    </div>

    <!-- Create Bill Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showCreateModal = false">
      <div class="card max-w-lg w-full mx-4">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold gradient-text">{{ $t('bills.createNew') }}</h2>
          <button @click="showCreateModal = false" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="createBill" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.type') }}</label>
            <select v-model="newBill.type" required class="input">
              <option value="electricity">{{ $t('bills.electricity') }}</option>
              <option value="gas">{{ $t('bills.gas') }}</option>
              <option value="internet">{{ $t('bills.internet') }}</option>
              <option value="inne">{{ $t('bills.inne') }}</option>
            </select>
          </div>

          <div v-if="newBill.type === 'inne'">
            <label class="block text-sm font-medium mb-2">{{ $t('bills.customTypeName') }}</label>
            <input v-model="newBill.customType" type="text" required class="input" :placeholder="$t('bills.customTypeExample')" />
          </div>

          <div v-if="newBill.type === 'inne'">
            <label class="block text-sm font-medium mb-2">{{ $t('bills.allocationMethod') }}</label>
            <select v-model="newBill.allocationType" required class="input">
              <option value="simple">{{ $t('bills.allocationSimple') }}</option>
              <option value="metered">{{ $t('bills.allocationMetered') }}</option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.amount') }}</label>
            <input v-model.number="newBill.totalAmountPLN" type="number" step="0.01" required class="input" placeholder="150.00" />
          </div>

          <div v-if="newBill.type === 'electricity' || newBill.type === 'gas' || (newBill.type === 'inne' && newBill.allocationType === 'metered')">
            <label class="block text-sm font-medium mb-2">{{ $t('bills.units') }}</label>
            <input v-model.number="newBill.totalUnits" type="number" step="0.001" class="input" placeholder="100.000" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('bills.periodStart') }}</label>
              <input v-model="newBill.periodStart" type="date" required class="input" min="2000-01-01" max="2099-12-31" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('bills.periodEnd') }}</label>
              <input v-model="newBill.periodEnd" type="date" required class="input" min="2000-01-01" max="2099-12-31" />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.paymentDeadline') }}</label>
            <input v-model="newBill.paymentDeadline" type="date" class="input" min="2000-01-01" max="2099-12-31" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.notes') }}</label>
            <textarea v-model="newBill.notes" class="input" rows="3" :placeholder="$t('bills.notesPlaceholder')"></textarea>
          </div>

          <div v-if="createError" class="flex items-center gap-2 p-3 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
            <AlertCircle class="w-4 h-4" />
            {{ createError }}
          </div>

          <div class="flex gap-3">
            <button type="submit" :disabled="creating" class="btn btn-primary flex-1 flex items-center justify-center gap-2">
              <div v-if="creating" class="loading-spinner"></div>
              <Plus v-else class="w-5 h-5" />
              {{ creating ? $t('common.creating') : $t('bills.createButton') }}
            </button>
            <button type="button" @click="showCreateModal = false" class="btn btn-outline">
              {{ $t('common.cancel') }}
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
            <label class="block text-sm font-medium mb-2">{{ $t('bills.type') }}</label>
            <select v-model="filters.type" class="input">
              <option value="">{{ $t('common.all') }}</option>
              <option value="electricity">{{ $t('bills.electricity') }}</option>
              <option value="gas">{{ $t('bills.gas') }}</option>
              <option value="internet">{{ $t('bills.internet') }}</option>
              <option value="inne">{{ $t('bills.inne') }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.dateFrom') }}</label>
            <input v-model="filters.dateFrom" type="date" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.dateTo') }}</label>
            <input v-model="filters.dateTo" type="date" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.sortBy') }}</label>
            <select v-model="filters.sortBy" class="input">
              <option value="date-desc">{{ $t('bills.sortDateNewest') }}</option>
              <option value="date-asc">{{ $t('bills.sortDateOldest') }}</option>
              <option value="amount-desc">{{ $t('bills.sortAmountDesc') }}</option>
              <option value="amount-asc">{{ $t('bills.sortAmountAsc') }}</option>
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
          <p class="text-lg">{{ $t('bills.noBills') }}</p>
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
                {{ $t('bills.postButton') }}
              </button>
              <button v-if="bill.status === 'posted'" @click="closeBill(bill.id)"
                      class="btn btn-sm btn-secondary flex items-center gap-1 flex-1">
                <Check class="w-3 h-3" />
                {{ $t('bills.closeButton') }}
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
                <th>{{ $t('common.description') }}</th>
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
                      {{ $t('bills.deleteButton') }}
                    </button>
                  </div>
                </td>
              </tr>
              <!-- Allocation breakdown row -->
              <template v-for="bill in filteredBills" :key="bill.id + '-allocation'">
                <tr v-if="expandedBills[bill.id]" class="bg-gray-800/30">
                  <td colspan="7" class="p-4">
                    <div v-if="loadingAllocations[bill.id]" class="text-center text-gray-400">
                      {{ $t('bills.loadingAllocation') }}
                    </div>
                    <div v-else-if="billAllocations[bill.id] && billAllocations[bill.id].length > 0" class="space-y-2">
                      <h3 class="text-sm font-semibold text-purple-400 mb-3">{{ $t('bills.allocationBreakdown') }}</h3>
                      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                        <div v-for="allocation in billAllocations[bill.id]" :key="allocation.subjectId"
                             class="bg-gray-800/50 rounded-lg p-3 border border-gray-700/50">
                          <div class="flex justify-between items-start">
                            <div>
                              <p class="font-medium text-white">{{ allocation.subjectName }}</p>
                              <p class="text-xs text-gray-400">{{ $t('bills.weight') }} {{ allocation.weight.toFixed(2) }}</p>
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
                              <span>{{ $t('bills.personal') }}</span>
                              <span>{{ formatMoney(allocation.personalAmount) }} PLN</span>
                            </div>
                            <div class="flex justify-between">
                              <span>{{ $t('bills.shared') }}</span>
                              <span>{{ formatMoney(allocation.sharedAmount) }} PLN</span>
                            </div>
                          </div>
                          <!-- Payment Status -->
                          <div v-if="hasUserPaid(bill.id, allocation)" class="mt-2 pt-2 border-t border-gray-700/50 text-center">
                            <span class="text-xs text-green-400">✓ {{ $t('bills.paid') }}</span>
                          </div>
                          <div v-else class="mt-2 pt-2 border-t border-gray-700/50 text-center">
                            <span class="text-xs text-yellow-400">⏳ {{ $t('bills.pendingPayment') }}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                    <div v-else class="text-center text-gray-400">
                      {{ $t('bills.noAllocationData') }}
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
                <option value="">{{ $t('common.select') }}</option>
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
            <label class="block text-sm font-medium mb-2">{{ $t('readings.bill') }}</label>
            <select v-model="readingFilters.billId" class="input">
              <option value="">{{ $t('common.all') }}</option>
              <option v-for="bill in allBills" :key="bill.id" :value="bill.id">
                {{ $t(`bills.${bill.type}`) }} - {{ formatDate(bill.periodStart) }}
              </option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('readings.personOrGroup') }}</label>
            <select v-model="readingFilters.subjectId" class="input">
              <option value="">{{ $t('common.everyone') }}</option>
              <optgroup :label="$t('readings.groups')">
                <option v-for="group in groups" :key="'group-' + group.id" :value="group.id">
                  {{ group.name }}
                </option>
              </optgroup>
              <optgroup :label="$t('readings.users')">
                <option v-for="user in users" :key="'user-' + user.id" :value="user.id">
                  {{ user.name }}
                </option>
              </optgroup>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.sortBy') }}</label>
            <select v-model="readingFilters.sortBy" class="input">
              <option value="date-desc">{{ $t('bills.sortDateNewest') }}</option>
              <option value="date-asc">{{ $t('bills.sortDateOldest') }}</option>
              <option value="value-desc">{{ $t('bills.sortAmountDesc') }}</option>
              <option value="value-asc">{{ $t('bills.sortAmountAsc') }}</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Readings List -->
      <div class="card">
        <h2 class="text-xl font-semibold mb-4">{{ $t('readings.recentReadings') }}</h2>
        <div v-if="loadingReadings" class="text-center py-8">{{ $t('common.loading') }}</div>
        <div v-else-if="filteredReadings.length === 0" class="text-center py-8 text-gray-400">{{ $t('readings.noReadings') }}</div>
        <div v-else class="space-y-3">
          <div v-for="reading in filteredReadings" :key="reading.id" class="flex justify-between items-center p-3 bg-gray-700 rounded hover:bg-gray-600 cursor-pointer transition-colors" @click="viewBill(reading.billId)">
            <div>
              <span class="font-medium">{{ formatMeterValue(reading.meterValue) }} {{ getUnitForBill(reading.billId) }}</span>
              <span class="text-gray-400 text-sm ml-4">{{ formatDateTime(reading.recordedAt) }}</span>
              <span v-if="getBillInfo(reading.billId)" class="text-blue-400 text-sm ml-4">
                {{ getBillInfo(reading.billId) }} →
              </span>
            </div>
            <span class="text-sm text-gray-400">{{ getSubjectName(reading.subjectId, reading.subjectType) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Recurring Bills Tab -->
    <div v-show="activeTab === 'recurring'">
      <div class="card">
        <h2 class="text-xl font-semibold mb-4">{{ $t('bills.recurringBills') }}</h2>
        <div v-if="loadingRecurring" class="text-center py-8">{{ $t('common.loading') }}</div>
        <div v-else-if="recurringTemplates.length === 0" class="text-center py-8 text-gray-400">
          <FileX class="w-12 h-12 mx-auto mb-3 opacity-50" />
          <p>{{ $t('bills.noRecurring') }}</p>
          <p class="text-sm mt-2">{{ $t('bills.createFirstRecurring') }}</p>
        </div>
        <div v-else class="space-y-3">
          <div v-for="template in recurringTemplates" :key="template.id" class="p-4 bg-gray-700 rounded-lg hover:bg-gray-600 transition-colors">
            <div class="flex justify-between items-start">
              <div class="flex-1">
                <div class="flex items-center gap-3 mb-2">
                  <h3 class="text-lg font-semibold">{{ template.customType }}</h3>
                  <span :class="['px-2 py-1 rounded text-xs', template.isActive ? 'bg-green-500/20 text-green-400' : 'bg-gray-500/20 text-gray-400']">
                    {{ template.isActive ? $t('bills.active') : $t('bills.inactive') }}
                  </span>
                  <span class="px-2 py-1 rounded text-xs bg-purple-500/20 text-purple-400">
                    {{ formatFrequency(template.frequency) }}
                  </span>
                </div>
                <div class="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span class="text-gray-400">{{ $t('bills.recurringAmount') }}</span>
                    <span class="ml-2 font-medium">{{ formatAmount(template.amount) }} PLN</span>
                  </div>
                  <div>
                    <span class="text-gray-400">{{ $t('bills.dayOfMonth') }}</span>
                    <span class="ml-2 font-medium">{{ template.dayOfMonth }}</span>
                  </div>
                  <div>
                    <span class="text-gray-400">{{ $t('bills.nextDueDate') }}</span>
                    <span class="ml-2 font-medium">{{ formatDate(template.nextDueDate) }}</span>
                  </div>
                  <div v-if="template.lastGeneratedAt">
                    <span class="text-gray-400">{{ $t('bills.lastGenerated') }}</span>
                    <span class="ml-2 font-medium">{{ formatDate(template.lastGeneratedAt) }}</span>
                  </div>
                </div>
                <div v-if="template.allocations && template.allocations.length > 0" class="mt-3">
                  <span class="text-gray-400 text-sm">{{ $t('bills.allocation') }}</span>
                  <div class="flex flex-wrap gap-2 mt-1">
                    <span v-for="(alloc, idx) in template.allocations" :key="idx" class="px-2 py-1 bg-gray-600 rounded text-xs">
                      {{ getAllocationLabel(alloc) }}
                    </span>
                  </div>
                </div>
                <div v-if="template.notes" class="mt-2 text-sm text-gray-400">
                  {{ template.notes }}
                </div>
              </div>
              <div class="flex gap-2 ml-4">
                <button @click="editRecurringTemplate(template)" class="btn btn-sm btn-outline">
                  {{ $t('common.edit') }}
                </button>
                <button v-if="authStore.hasPermission('bills.delete')" @click="deleteRecurringTemplate(template.id)" class="btn btn-sm btn-outline text-red-400 hover:text-red-300">
                  <Trash2 class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Recurring Bill Modal -->
    <div v-if="showRecurringModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" @click.self="closeRecurringModal">
      <div class="card max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-2xl font-bold gradient-text">{{ editingRecurring ? $t('common.edit') : $t('common.create') }} {{ $t('bills.recurringModalTitle') }}</h2>
          <button @click="closeRecurringModal" class="text-gray-400 hover:text-white">
            <X class="w-6 h-6" />
          </button>
        </div>

        <form @submit.prevent="saveRecurringTemplate" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.recurringName') }}</label>
            <input v-model="newRecurring.customType" type="text" required class="input" :placeholder="$t('bills.recurringNameExample')" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('bills.amount') }}</label>
              <input v-model.number="newRecurring.amount" type="number" step="0.01" required class="input" placeholder="150.00" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('bills.frequency') }}</label>
              <select v-model="newRecurring.frequency" required class="input">
                <option value="monthly">{{ $t('bills.monthly') }}</option>
                <option value="quarterly">{{ $t('bills.quarterly') }}</option>
                <option value="yearly">{{ $t('bills.yearly') }}</option>
              </select>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('bills.dayOfMonthLabel') }}</label>
              <input v-model.number="newRecurring.dayOfMonth" type="number" min="1" max="31" required class="input" placeholder="15" />
              <p class="text-xs text-gray-400 mt-1">{{ $t('bills.dayOfMonthHint') }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-2">{{ $t('bills.startDate') }}</label>
              <input v-model="newRecurring.startDate" type="date" class="input" min="2000-01-01" max="2099-12-31" required />
              <p class="text-xs text-gray-400 mt-1">{{ $t('bills.startDateHint') }}</p>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.costAllocation') }}</label>
            <div class="space-y-3">
              <div v-for="(alloc, idx) in newRecurring.allocations" :key="idx" class="p-3 bg-gray-700 rounded-lg space-y-2">
                <div class="flex gap-2">
                  <select v-model="alloc.subjectType" class="input flex-1" @change="alloc.subjectId = ''">
                    <option value="user">{{ $t('common.user') }}</option>
                    <option value="group">{{ $t('common.group') }}</option>
                  </select>
                  <select v-model="alloc.subjectId" required class="input flex-1">
                    <option value="">{{ $t('common.select') }}</option>
                    <option v-if="alloc.subjectType === 'user'" v-for="user in users" :key="user.id" :value="user.id">
                      {{ user.name || user.email }}
                    </option>
                    <option v-if="alloc.subjectType === 'group'" v-for="group in groups" :key="group.id" :value="group.id">
                      {{ group.name }}
                    </option>
                  </select>
                  <button type="button" @click="removeAllocation(idx)" class="btn btn-sm btn-outline text-red-400">
                    <X class="w-4 h-4" />
                  </button>
                </div>
                <div class="flex gap-2 items-center">
                  <select v-model="alloc.allocationType" class="input w-32">
                    <option value="percentage">{{ $t('bills.percentage') }}</option>
                    <option value="fraction">{{ $t('bills.fraction') }}</option>
                    <option value="fixed">{{ $t('bills.fixedAmount') }}</option>
                  </select>

                  <!-- Percentage input -->
                  <div v-if="alloc.allocationType === 'percentage'" class="flex items-center gap-2 flex-1">
                    <input v-model.number="alloc.percentage" type="number" step="0.01" min="0" max="100" required class="input flex-1" placeholder="50" />
                    <span class="text-gray-400">%</span>
                  </div>

                  <!-- Fraction input -->
                  <div v-if="alloc.allocationType === 'fraction'" class="flex items-center gap-2 flex-1">
                    <input v-model.number="alloc.fractionNum" type="number" min="1" required class="input w-20" placeholder="1" />
                    <span class="text-gray-400">/</span>
                    <input v-model.number="alloc.fractionDenom" type="number" min="1" required class="input w-20" placeholder="3" />
                  </div>

                  <!-- Fixed amount input -->
                  <div v-if="alloc.allocationType === 'fixed'" class="flex items-center gap-2 flex-1">
                    <input v-model.number="alloc.fixedAmount" type="number" step="0.01" min="0" required class="input flex-1" placeholder="100.00" />
                    <span class="text-gray-400">PLN</span>
                  </div>
                </div>
              </div>
              <button type="button" @click="addAllocation" class="btn btn-sm btn-outline w-full">
                <Plus class="w-4 h-4 mr-2" />
                {{ $t('bills.addAllocation') }}
              </button>
              <p v-if="!isAllocationValid" class="text-xs text-yellow-400">
                {{ allocationValidationMessage }}
              </p>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">{{ $t('bills.recurringNotes') }}</label>
            <textarea v-model="newRecurring.notes" class="input" rows="2" :placeholder="$t('bills.notesPlaceholder')"></textarea>
          </div>

          <div v-if="recurringError" class="flex items-center gap-2 p-3 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
            <AlertCircle class="w-4 h-4" />
            {{ recurringError }}
          </div>

          <div class="flex gap-3 justify-end pt-4">
            <button type="button" @click="closeRecurringModal" class="btn btn-outline">
              {{ $t('common.cancel') }}
            </button>
            <button type="submit" :disabled="savingRecurring || !isAllocationValid" class="btn btn-primary">
              {{ savingRecurring ? $t('common.saving') : (editingRecurring ? $t('bills.saveChanges') : $t('common.create')) }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useEventStream } from '../composables/useEventStream'
import { useDataEvents, DATA_EVENTS } from '../composables/useDataEvents'
import api from '../api/client'
import {
  Plus, FileX, Zap, Flame, Wifi, Calendar, Receipt, Gauge,
  Send, Check, X, AlertCircle, Trash2, ChevronDown, ChevronUp
} from 'lucide-vue-next'

const router = useRouter()
const { t, locale } = useI18n()
const authStore = useAuthStore()
const { connect, on } = useEventStream()
const { on: onDataEvent } = useDataEvents()
const activeTab = ref('bills')

// Bills state
const bills = ref([])
const allPayments = ref([]) // Store all payments to show who paid
const loading = ref(false)
const showCreateModal = ref(false)
const creating = ref(false)
const createError = ref('')
const billAllocations = ref({}) // Store allocations by bill ID
const billPayments = ref({}) // Store payments by bill ID
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
  subjectId: '',
  sortBy: 'date-desc'
})

const form = ref({
  billId: '',
  meterReading: '',
  readingDate: new Date().toISOString().slice(0, 16)
})

// Recurring bills state
const recurringTemplates = ref([])
const loadingRecurring = ref(false)
const showRecurringModal = ref(false)
const savingRecurring = ref(false)
const editingRecurring = ref(null)
const recurringError = ref('')
const groups = ref([])

const newRecurring = ref({
  customType: '',
  frequency: 'monthly',
  amount: '',
  dayOfMonth: 1,
  startDate: '',
  allocations: [],
  notes: ''
})

const isAllocationValid = computed(() => {
  if (newRecurring.value.allocations.length === 0) return false

  // Check that all allocations have a subject selected
  if (newRecurring.value.allocations.some(alloc => !alloc.subjectId)) return false

  const { total, hasNonFixed } = newRecurring.value.allocations.reduce((acc, alloc) => {
    if (alloc.allocationType === 'percentage') {
      acc.total += (parseFloat(alloc.percentage) || 0) / 100
      acc.hasNonFixed = true
    } else if (alloc.allocationType === 'fraction') {
      const num = parseInt(alloc.fractionNum) || 0
      const denom = parseInt(alloc.fractionDenom) || 1
      acc.total += num / denom
      acc.hasNonFixed = true
    }
    return acc
  }, { total: 0, hasNonFixed: false })

  if (hasNonFixed) {
    return total >= 0.999 && total <= 1.001
  }

  return true
})

const allocationValidationMessage = computed(() => {
  if (newRecurring.value.allocations.length === 0) return t('bills.addAllocationRequired')

  // Check for missing subject selection
  if (newRecurring.value.allocations.some(alloc => !alloc.subjectId)) {
    return t('bills.selectSubjectRequired')
  }

  const { total, hasNonFixed } = newRecurring.value.allocations.reduce((acc, alloc) => {
    if (alloc.allocationType === 'percentage') {
      acc.total += (parseFloat(alloc.percentage) || 0) / 100
      acc.hasNonFixed = true
    } else if (alloc.allocationType === 'fraction') {
      const num = parseInt(alloc.fractionNum) || 0
      const denom = parseInt(alloc.fractionDenom) || 1
      acc.total += num / denom
      acc.hasNonFixed = true
    }
    return acc
  }, { total: 0, hasNonFixed: false })

  if (hasNonFixed) {
    return t('bills.allocationTotal', { total: (total * 100).toFixed(2) })
  }

  return ''
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

  if (readingFilters.value.subjectId) {
    result = result.filter(r => r.subjectId === readingFilters.value.subjectId)
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
  await loadRecurringTemplates()
  await loadGroups()

  // Connect to WebSocket for real-time updates
  connect()

  // Listen for bill-related events from WebSocket
  on('bill.created', () => {
    console.log('[Bills] Bill created event received, refreshing...')
    loadBills()
    loadReadingsData()
    loadRecurringTemplates()
  })

  on('bill.posted', () => {
    console.log('[Bills] Bill posted event received, refreshing...')
    loadBills()
    loadReadingsData()
  })

  on('consumption.created', () => {
    console.log('[Bills] Consumption created event received, refreshing...')
    loadReadingsData()
  })

  on('payment.created', () => {
    console.log('[Bills] Payment created event received, refreshing...')
    loadBills()
  })

  // Listen for local data events (from same tab actions in other components)
  onDataEvent(DATA_EVENTS.BILL_CREATED, () => loadBills())
  onDataEvent(DATA_EVENTS.BILL_UPDATED, () => loadBills())
  onDataEvent(DATA_EVENTS.BILL_DELETED, () => loadBills())
  onDataEvent(DATA_EVENTS.CONSUMPTION_CREATED, () => loadReadingsData())
  onDataEvent(DATA_EVENTS.CONSUMPTION_DELETED, () => loadReadingsData())
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
    const currentYear = new Date().getFullYear()
    const maxYear = currentYear + 2 // Allow bills up to 2 years in the future

    if (isNaN(startDate.getTime()) || isNaN(endDate.getTime())) {
      createError.value = t('errors.invalidDates')
      creating.value = false
      return
    }

    if (startDate.getFullYear() < 2000 || startDate.getFullYear() > maxYear) {
      createError.value = t('errors.startDateRange', { maxYear })
      creating.value = false
      return
    }

    if (endDate.getFullYear() < 2000 || endDate.getFullYear() > maxYear) {
      createError.value = t('errors.endDateRange', { maxYear })
      creating.value = false
      return
    }

    if (endDate <= startDate) {
      createError.value = t('errors.endBeforeStart')
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
    createError.value = err.response?.data?.error || t('errors.createBillFailed')
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
  if (!confirm(t('bills.confirmDelete'))) {
    return
  }

  try {
    await api.delete(`/bills/${billId}`)
    await loadBills()
    await loadReadingsData()
  } catch (err) {
    console.error('Failed to delete bill:', err)
    alert(t('errors.deleteFailed') + ' ' + (err.response?.data?.error || err.message))
  }
}

async function submitReading() {
  loadingReading.value = true
  try {
    const units = parseFloat(form.value.meterReading)
    if (isNaN(units) || units <= 0) {
      throw new Error(t('errors.invalidConsumption'))
    }

    await api.post('/consumptions', {
      billId: form.value.billId,
      units,
      meterValue: units,
      recordedAt: new Date(form.value.readingDate).toISOString()
    })

    form.value.meterReading = ''
    await loadReadingsData()
  } catch (err) {
    console.error('Failed to submit reading:', err)
    alert(t('errors.saveReadingFailed') + ' ' + (err.response?.data?.error || err.message))
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
  const localeMap = { 'pl': 'pl-PL', 'en': 'en-US' }
  const dateLocale = localeMap[locale.value] || 'en-US'
  return new Date(date).toLocaleDateString(dateLocale, { day: 'numeric', month: 'short', year: 'numeric' })
}

async function loadBillAllocation(billId) {
  if (billAllocations.value[billId]) {
    // Already loaded, just return
    return
  }

  loadingAllocations.value[billId] = true
  try {
    const [allocRes, paymentsRes] = await Promise.all([
      api.get(`/bills/${billId}/allocation`),
      api.get(`/payments/bill/${billId}`)
    ])
    billAllocations.value[billId] = allocRes.data
    billPayments.value[billId] = paymentsRes.data || []
  } catch (err) {
    console.error('Failed to load allocation:', err)
    billAllocations.value[billId] = []
    billPayments.value[billId] = []
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
  const localeMap = { 'pl': 'pl-PL', 'en': 'en-US' }
  const dateLocale = localeMap[locale.value] || 'en-US'
  return new Date(date).toLocaleString(dateLocale)
}

function getUnit(type) {
  if (type === 'electricity') return 'kWh'
  if (type === 'gas') return 'm³'
  return ''
}

function getUnitForBill(billId) {
  const bill = allBills.value.find(b => b.id === billId)
  if (!bill) return t('common.units')
  return getUnit(bill.type) || t('common.units')
}

function getSubjectName(subjectId, subjectType) {
  if (subjectType === 'group') {
    const group = groups.value.find(g => g.id === subjectId)
    return group ? group.name : t('errors.unknownGroup')
  } else {
    const user = users.value.find(u => u.id === subjectId)
    return user ? user.name : t('errors.unknown')
  }
}

function getBillInfo(billId) {
  const bill = allBills.value.find(b => b.id === billId)
  if (!bill) return ''

  const typeLabel = t(`bills.${bill.type}`, bill.type)
  const dateRange = `${formatDate(bill.periodStart)} - ${formatDate(bill.periodEnd)}`
  return `${typeLabel}: ${dateRange}`
}

function viewBill(billId) {
  router.push(`/bills/${billId}`)
}

// Recurring Bills Functions
async function loadRecurringTemplates() {
  loadingRecurring.value = true
  try {
    const response = await api.get('/recurring-bills')
    recurringTemplates.value = response.data || []
  } catch (err) {
    console.error('Failed to load recurring templates:', err)
    recurringTemplates.value = []
  } finally {
    loadingRecurring.value = false
  }
}

async function loadGroups() {
  try {
    const response = await api.get('/groups')
    groups.value = response.data || []
  } catch (err) {
    console.error('Failed to load groups:', err)
    groups.value = []
  }
}

function addAllocation() {
  newRecurring.value.allocations.push({
    subjectType: 'user',
    subjectId: '',
    allocationType: 'fraction',
    percentage: null,
    fractionNum: 1,
    fractionDenom: 1,
    fixedAmount: null
  })
}

function removeAllocation(index) {
  newRecurring.value.allocations.splice(index, 1)
}

function closeRecurringModal() {
  showRecurringModal.value = false
  editingRecurring.value = null
  recurringError.value = ''
  newRecurring.value = {
    customType: '',
    frequency: 'monthly',
    amount: '',
    dayOfMonth: 1,
    startDate: '',
    allocations: [],
    notes: ''
  }
}

async function saveRecurringTemplate() {
  savingRecurring.value = true
  recurringError.value = ''

  try {
    if (!isAllocationValid.value) {
      recurringError.value = allocationValidationMessage.value || 'Nieprawidłowe alokacje'
      savingRecurring.value = false
      return
    }

    const payload = {
      customType: newRecurring.value.customType,
      frequency: newRecurring.value.frequency,
      amount: parseFloat(newRecurring.value.amount).toFixed(2),
      dayOfMonth: parseInt(newRecurring.value.dayOfMonth),
      startDate: newRecurring.value.startDate ? new Date(newRecurring.value.startDate).toISOString() : undefined,
      allocations: newRecurring.value.allocations.map(a => {
        const alloc = {
          subjectType: a.subjectType,
          subjectId: a.subjectId,
          allocationType: a.allocationType
        }

        if (a.allocationType === 'percentage') {
          alloc.percentage = parseFloat(a.percentage)
        } else if (a.allocationType === 'fraction') {
          alloc.fractionNumerator = parseInt(a.fractionNum)
          alloc.fractionDenominator = parseInt(a.fractionDenom)
        } else if (a.allocationType === 'fixed') {
          alloc.fixedAmount = parseFloat(a.fixedAmount).toFixed(2)
        }

        return alloc
      }),
      notes: newRecurring.value.notes || undefined
    }

    if (editingRecurring.value) {
      await api.patch(`/recurring-bills/${editingRecurring.value}`, payload)
    } else {
      await api.post('/recurring-bills', payload)
    }

    await loadRecurringTemplates()
    closeRecurringModal()
  } catch (err) {
    recurringError.value = err.response?.data?.error || t('errors.saveTemplateFailed')
  } finally {
    savingRecurring.value = false
  }
}

function editRecurringTemplate(template) {
  editingRecurring.value = template.id
  newRecurring.value = {
    customType: template.customType,
    frequency: template.frequency,
    amount: formatAmount(template.amount),
    dayOfMonth: template.dayOfMonth,
    startDate: template.startDate ? new Date(template.startDate).toISOString().split('T')[0] : '',
    allocations: template.allocations.map(a => ({
      subjectType: a.subjectType,
      subjectId: a.subjectId,
      allocationType: a.allocationType,
      percentage: a.percentage || null,
      fractionNum: a.fractionNumerator || 1,
      fractionDenom: a.fractionDenominator || 1,
      fixedAmount: a.fixedAmount ? formatAmount(a.fixedAmount) : null
    })),
    notes: template.notes || ''
  }
  showRecurringModal.value = true
}

async function deleteRecurringTemplate(templateId) {
  if (!confirm(t('bills.confirmDeleteRecurring'))) {
    return
  }

  try {
    await api.delete(`/recurring-bills/${templateId}`)
    await loadRecurringTemplates()
  } catch (err) {
    console.error('Failed to delete recurring template:', err)
    alert(t('errors.deleteTemplateFailed'))
  }
}

function formatFrequency(frequency) {
  return t(`bills.${frequency}`, frequency)
}

function formatAmount(amountString) {
  if (!amountString) return '0.00'
  // Parse Decimal128 JSON format or plain number
  const cleaned = String(amountString).replace(/["{}$numberDecimal:]/g, '')
  const amount = parseFloat(cleaned)
  return amount.toFixed(2)
}

function getAllocationLabel(alloc) {
  const subject = alloc.subjectType === 'user'
    ? users.value.find(u => u.id === alloc.subjectId)
    : groups.value.find(g => g.id === alloc.subjectId)

  const name = subject ? (subject.name || subject.email) : t('errors.unknown')

  let value = ''
  if (alloc.allocationType === 'percentage') {
    value = `${alloc.percentage}%`
  } else if (alloc.allocationType === 'fraction') {
    value = `${alloc.fractionNumerator}/${alloc.fractionDenominator}`
  } else if (alloc.allocationType === 'fixed') {
    value = `${formatAmount(alloc.fixedAmount)} PLN`
  }

  return `${name}: ${value}`
}

const PAYMENT_MATCH_EPSILON = 0.01

function hasUserPaid(billId, allocation) {
  const payments = billPayments.value[billId]
  if (!payments || !payments.length) return false

  const targetAmount = Number(allocation.amount || 0)
  if (!targetAmount) return false

  const subjectIds = allocation.subjectType === 'group'
    ? getGroupMemberIds(allocation.subjectId)
    : [allocation.subjectId]

  if (!subjectIds.length) return false

  const paidAmount = sumPaymentsForUsers(payments, subjectIds)
  return paidAmount + PAYMENT_MATCH_EPSILON >= targetAmount
}

function getGroupMemberIds(groupId) {
  if (!groupId) return []
  return users.value
    .filter(user => user.groupId === groupId)
    .map(user => user.id)
}

function sumPaymentsForUsers(payments, userIds) {
  return payments.reduce((total, payment) => {
    if (userIds.includes(payment.payerUserId)) {
      total += decimalToNumber(payment.amountPLN)
    }
    return total
  }, 0)
}

function decimalToNumber(value) {
  if (value == null) return 0
  if (typeof value === 'number') return value
  if (typeof value === 'string') return parseFloat(value)
  if (typeof value === 'object' && value.$numberDecimal) {
    return parseFloat(value.$numberDecimal)
  }
  return 0
}
</script>
