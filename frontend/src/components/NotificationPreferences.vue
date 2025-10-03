<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="close">
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm" @click="close" />

        <div class="relative bg-gray-800 rounded-xl shadow-2xl w-full max-w-md border border-gray-700">
          <!-- Header -->
          <div class="p-6 border-b border-gray-700">
            <div class="flex items-center justify-between">
              <h2 class="text-xl font-bold text-white">Ustawienia powiadomień</h2>
              <button
                @click="close"
                class="p-2 hover:bg-gray-700 rounded-lg transition-colors">
                <X class="w-5 h-5 text-gray-400" />
              </button>
            </div>
          </div>

          <!-- Content -->
          <div class="p-6 space-y-4">
            <p class="text-sm text-gray-400 mb-4">
              Wybierz, które typy powiadomień chcesz otrzymywać
            </p>

            <!-- Notification Type Toggles -->
            <div class="space-y-3">
              <div
                v-for="(config, type) in notificationTypes"
                :key="type"
                class="flex items-center justify-between p-3 bg-gray-700/30 rounded-lg hover:bg-gray-700/50 transition-colors">
                <div class="flex items-center gap-3">
                  <component
                    :is="config.icon"
                    :class="['w-5 h-5', getIconColor(type)]" />
                  <div>
                    <p class="text-white font-medium">{{ config.label }}</p>
                    <p class="text-xs text-gray-400">{{ config.description }}</p>
                  </div>
                </div>

                <label class="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    :checked="localPreferences[type]"
                    @change="togglePreference(type)"
                    class="sr-only peer" />
                  <div class="w-11 h-6 bg-gray-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:bg-purple-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all" />
                </label>
              </div>
            </div>

            <!-- Quick Actions -->
            <div class="flex gap-2 pt-4">
              <button
                @click="enableAll"
                class="btn btn-secondary btn-sm flex-1">
                <Check class="w-4 h-4" />
                Włącz wszystkie
              </button>
              <button
                @click="disableAll"
                class="btn btn-outline btn-sm flex-1">
                <X class="w-4 h-4" />
                Wyłącz wszystkie
              </button>
            </div>
          </div>

          <!-- Footer -->
          <div class="p-6 border-t border-gray-700 flex justify-end gap-3">
            <button @click="close" class="btn btn-outline">
              Anuluj
            </button>
            <button @click="save" class="btn btn-primary">
              <Save class="w-4 h-4" />
              Zapisz
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useNotificationStore } from '../stores/notification'
import { X, Check, Save, FileText, CheckSquare, ShoppingCart, DollarSign, UserPlus } from 'lucide-vue-next'

const props = defineProps({
  isOpen: Boolean
})

const emit = defineEmits(['close'])

const notificationStore = useNotificationStore()
const localPreferences = ref({ ...notificationStore.preferences })

const notificationTypes = {
  bill: {
    label: 'Rachunki',
    description: 'Nowe rachunki i odczyty',
    icon: FileText
  },
  chore: {
    label: 'Obowiązki',
    description: 'Nowe i zaktualizowane obowiązki',
    icon: CheckSquare
  },
  supply: {
    label: 'Zaopatrzenie',
    description: 'Nowe pozycje i zakupy',
    icon: ShoppingCart
  },
  loan: {
    label: 'Pożyczki',
    description: 'Nowe pożyczki i spłaty',
    icon: DollarSign
  },
  permission: {
    label: 'Uprawnienia',
    description: 'Zmiany w uprawnieniach',
    icon: UserPlus
  }
}

// Watch for changes to store preferences
watch(() => notificationStore.preferences, (newPrefs) => {
  localPreferences.value = { ...newPrefs }
}, { deep: true })

// Watch for modal opening to reset local state
watch(() => props.isOpen, (isOpen) => {
  if (isOpen) {
    localPreferences.value = { ...notificationStore.preferences }
  }
})

function togglePreference(type) {
  localPreferences.value[type] = !localPreferences.value[type]
}

function enableAll() {
  Object.keys(localPreferences.value).forEach(key => {
    localPreferences.value[key] = true
  })
}

function disableAll() {
  Object.keys(localPreferences.value).forEach(key => {
    localPreferences.value[key] = false
  })
}

function save() {
  // Save to store
  Object.keys(localPreferences.value).forEach(key => {
    notificationStore.setPreference(key, localPreferences.value[key])
  })

  emit('close')
}

function close() {
  emit('close')
}

function getIconColor(type) {
  const colors = {
    bill: 'text-blue-400',
    chore: 'text-purple-400',
    supply: 'text-green-400',
    loan: 'text-yellow-400',
    permission: 'text-pink-400'
  }
  return colors[type] || 'text-gray-400'
}
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: all 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
  transform: scale(0.95);
}
</style>
