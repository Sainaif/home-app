import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api/client'

// Migration store - handles MongoDB to SQLite migration (v1.5 bridge release)
export const useMigrationStore = defineStore('migration', () => {
  const migrationEnabled = ref(false)
  const hasExistingData = ref(false)
  const lastMigration = ref(null)
  const migrationInProgress = ref(false)
  const migrationResult = ref(null)
  const migrationError = ref(null)

  // Check if migration mode is enabled on the server
  async function checkMigrationStatus() {
    try {
      const response = await api.get('/migrate/status')
      migrationEnabled.value = response.data.migrationEnabled
      hasExistingData.value = response.data.hasExistingData
      lastMigration.value = response.data.lastMigration
      return response.data
    } catch (error) {
      // If endpoint doesn't exist, migration mode is disabled
      if (error.response?.status === 404) {
        migrationEnabled.value = false
        return { migrationEnabled: false }
      }
      console.warn('Failed to check migration status:', error)
      migrationEnabled.value = false
      return { migrationEnabled: false }
    }
  }

  // Import backup data from MongoDB export
  async function importBackup(backupData, overwrite = false) {
    migrationInProgress.value = true
    migrationError.value = null
    migrationResult.value = null

    try {
      const url = overwrite ? '/migrate/import?overwrite=true' : '/migrate/import'
      const response = await api.post(url, backupData, {
        headers: {
          'Content-Type': 'application/json'
        }
      })

      migrationResult.value = response.data.result
      hasExistingData.value = true

      if (response.data.result?.completedAt) {
        lastMigration.value = response.data.result.completedAt
      }

      return response.data
    } catch (error) {
      const errorMessage = error.response?.data?.error || error.message || 'Migration failed'
      migrationError.value = errorMessage

      // Return partial result if available
      if (error.response?.data?.result) {
        migrationResult.value = error.response.data.result
      }

      throw new Error(errorMessage)
    } finally {
      migrationInProgress.value = false
    }
  }

  // Import from a file
  async function importFromFile(file, overwrite = false) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader()

      reader.onload = async (e) => {
        try {
          const backupData = JSON.parse(e.target.result)
          const result = await importBackup(backupData, overwrite)
          resolve(result)
        } catch (error) {
          if (error instanceof SyntaxError) {
            migrationError.value = 'Invalid JSON file'
            reject(new Error('Invalid JSON file'))
          } else {
            reject(error)
          }
        }
      }

      reader.onerror = () => {
        migrationError.value = 'Failed to read file'
        reject(new Error('Failed to read file'))
      }

      reader.readAsText(file)
    })
  }

  // Clear migration state
  function clearState() {
    migrationResult.value = null
    migrationError.value = null
  }

  return {
    // State
    migrationEnabled,
    hasExistingData,
    lastMigration,
    migrationInProgress,
    migrationResult,
    migrationError,

    // Actions
    checkMigrationStatus,
    importBackup,
    importFromFile,
    clearState
  }
})
