<template>
  <div v-if="migrationStore.migrationEnabled" class="bg-gray-800/50 rounded-xl p-6 border border-gray-700">
    <div class="flex items-center gap-3 mb-4">
      <div class="p-2 bg-purple-600/20 rounded-lg">
        <svg class="w-6 h-6 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2 1 3 3 3h10c2 0 3-1 3-3V7c0-2-1-3-3-3H7c-2 0-3 1-3 3zm0 5h16M8 3v4m8-4v4" />
        </svg>
      </div>
      <div>
        <h3 class="text-lg font-semibold text-white">{{ $t('migration.title') }}</h3>
        <p class="text-sm text-gray-400">{{ $t('migration.description') }}</p>
      </div>
    </div>

    <!-- Warning if database has existing data -->
    <div v-if="migrationStore.hasExistingData && !overwriteConfirmed" class="mb-4 p-4 bg-amber-900/30 border border-amber-600/50 rounded-lg">
      <div class="flex items-start gap-3">
        <svg class="w-5 h-5 text-amber-400 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div>
          <p class="text-amber-200 text-sm font-medium">{{ $t('migration.existingDataWarning') }}</p>
          <label class="flex items-center gap-2 mt-2 cursor-pointer">
            <input
              type="checkbox"
              v-model="overwriteConfirmed"
              class="w-4 h-4 rounded border-gray-600 bg-gray-700 text-purple-500 focus:ring-purple-500"
            />
            <span class="text-sm text-gray-300">{{ $t('migration.confirmOverwrite') }}</span>
          </label>
        </div>
      </div>
    </div>

    <!-- Last migration info -->
    <div v-if="migrationStore.lastMigration" class="mb-4 p-3 bg-green-900/20 border border-green-600/30 rounded-lg">
      <p class="text-green-300 text-sm">
        {{ $t('migration.lastMigration') }}: {{ formatDate(migrationStore.lastMigration) }}
      </p>
    </div>

    <!-- File upload area -->
    <div
      class="border-2 border-dashed rounded-lg p-6 text-center transition-colors"
      :class="[
        isDragging ? 'border-purple-500 bg-purple-900/20' : 'border-gray-600 hover:border-gray-500',
        migrationStore.migrationInProgress ? 'opacity-50 pointer-events-none' : ''
      ]"
      @dragover.prevent="isDragging = true"
      @dragleave.prevent="isDragging = false"
      @drop.prevent="handleFileDrop"
    >
      <input
        type="file"
        ref="fileInput"
        accept=".json"
        class="hidden"
        @change="handleFileSelect"
        :disabled="migrationStore.migrationInProgress"
      />

      <div v-if="!selectedFile && !migrationStore.migrationInProgress">
        <svg class="w-12 h-12 mx-auto text-gray-500 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
        </svg>
        <p class="text-gray-400 mb-2">{{ $t('migration.dragDropHint') }}</p>
        <button
          @click="$refs.fileInput.click()"
          class="text-purple-400 hover:text-purple-300 font-medium"
        >
          {{ $t('migration.selectBackupFile') }}
        </button>
      </div>

      <div v-else-if="selectedFile && !migrationStore.migrationInProgress">
        <div class="flex items-center justify-center gap-3 mb-4">
          <svg class="w-8 h-8 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <div class="text-left">
            <p class="text-white font-medium">{{ selectedFile.name }}</p>
            <p class="text-gray-400 text-sm">{{ formatFileSize(selectedFile.size) }}</p>
          </div>
          <button
            @click="clearFile"
            class="ml-2 p-1 text-gray-400 hover:text-white rounded"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <div v-else class="py-4">
        <div class="animate-spin w-8 h-8 border-4 border-purple-500 border-t-transparent rounded-full mx-auto mb-3"></div>
        <p class="text-purple-300">{{ $t('migration.importing') }}</p>
      </div>
    </div>

    <!-- Start migration button -->
    <button
      v-if="selectedFile && !migrationStore.migrationInProgress"
      @click="startMigration"
      :disabled="migrationStore.hasExistingData && !overwriteConfirmed"
      class="w-full mt-4 py-3 px-4 bg-purple-600 hover:bg-purple-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
      </svg>
      {{ $t('migration.startMigration') }}
    </button>

    <!-- Migration result -->
    <div v-if="migrationStore.migrationResult" class="mt-4">
      <div
        :class="[
          'p-4 rounded-lg',
          migrationStore.migrationResult.success ? 'bg-green-900/30 border border-green-600/50' : 'bg-red-900/30 border border-red-600/50'
        ]"
      >
        <div class="flex items-center gap-2 mb-2">
          <svg v-if="migrationStore.migrationResult.success" class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <svg v-else class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
          <span :class="migrationStore.migrationResult.success ? 'text-green-300' : 'text-red-300'" class="font-medium">
            {{ migrationStore.migrationResult.success ? $t('migration.success') : $t('migration.failed') }}
          </span>
        </div>

        <!-- Records migrated -->
        <div v-if="migrationStore.migrationResult.recordsMigrated" class="mt-3">
          <p class="text-sm text-gray-400 mb-2">{{ $t('migration.recordsMigrated') }}:</p>
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 text-sm">
            <div
              v-for="(count, entity) in migrationStore.migrationResult.recordsMigrated"
              :key="entity"
              class="bg-gray-800/50 px-2 py-1 rounded"
            >
              <span class="text-gray-400">{{ entity }}:</span>
              <span class="text-white ml-1">{{ count }}</span>
            </div>
          </div>
        </div>

        <!-- Errors -->
        <div v-if="migrationStore.migrationResult.errors?.length" class="mt-3">
          <p class="text-sm text-red-400 mb-1">{{ $t('migration.errors') }}:</p>
          <ul class="text-sm text-gray-400 list-disc list-inside max-h-32 overflow-y-auto">
            <li v-for="(error, i) in migrationStore.migrationResult.errors" :key="i">{{ error }}</li>
          </ul>
        </div>
      </div>

      <button
        @click="resetState"
        class="mt-3 text-sm text-gray-400 hover:text-white"
      >
        {{ $t('migration.startNew') }}
      </button>
    </div>

    <!-- Error display -->
    <div v-if="migrationStore.migrationError && !migrationStore.migrationResult" class="mt-4 p-4 bg-red-900/30 border border-red-600/50 rounded-lg">
      <p class="text-red-300 text-sm">{{ migrationStore.migrationError }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useMigrationStore } from '../stores/migration'

const migrationStore = useMigrationStore()

const selectedFile = ref(null)
const isDragging = ref(false)
const overwriteConfirmed = ref(false)
const fileInput = ref(null)

onMounted(() => {
  migrationStore.checkMigrationStatus()
})

function handleFileDrop(event) {
  isDragging.value = false
  const files = event.dataTransfer.files
  if (files.length > 0 && files[0].name.endsWith('.json')) {
    selectedFile.value = files[0]
    migrationStore.clearState()
  }
}

function handleFileSelect(event) {
  const files = event.target.files
  if (files.length > 0) {
    selectedFile.value = files[0]
    migrationStore.clearState()
  }
}

function clearFile() {
  selectedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
  migrationStore.clearState()
}

async function startMigration() {
  if (!selectedFile.value) return

  try {
    const shouldOverwrite = migrationStore.hasExistingData && overwriteConfirmed.value
    await migrationStore.importFromFile(selectedFile.value, shouldOverwrite)
  } catch (error) {
    console.error('Migration failed:', error)
  }
}

function resetState() {
  selectedFile.value = null
  overwriteConfirmed.value = false
  if (fileInput.value) {
    fileInput.value.value = ''
  }
  migrationStore.clearState()
  migrationStore.checkMigrationStatus()
}

function formatFileSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatDate(dateStr) {
  return new Date(dateStr).toLocaleString()
}
</script>
