<template>
  <div class="min-h-screen flex items-center justify-center px-4">
    <div class="card max-w-md w-full">
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-amber-600 to-orange-600 mb-4">
          <Shield class="w-8 h-8 text-white" />
        </div>
        <h1 class="text-4xl font-bold gradient-text mb-2">{{ $t('auth.forceChangePasswordTitle') }}</h1>
        <p class="text-gray-400">{{ $t('auth.forceChangePasswordDescription') }}</p>
      </div>

      <!-- Change password form -->
      <form @submit.prevent="handleChangePassword" class="space-y-5">
        <div>
          <label class="block text-sm font-medium mb-2 text-gray-300">
            <Lock class="w-4 h-4 inline mr-2" />
            {{ $t('settings.currentPassword') }}
          </label>
          <input
            v-model="currentPassword"
            type="password"
            required
            class="input"
            placeholder="••••••••"
          />
        </div>

        <div>
          <label class="block text-sm font-medium mb-2 text-gray-300">
            <Lock class="w-4 h-4 inline mr-2" />
            {{ $t('auth.newPassword') }}
          </label>
          <input
            v-model="newPassword"
            type="password"
            required
            minlength="8"
            class="input"
            placeholder="••••••••"
          />
          <p class="text-xs text-gray-400 mt-1">{{ $t('auth.minPasswordLength') }}</p>
        </div>

        <div>
          <label class="block text-sm font-medium mb-2 text-gray-300">
            <Lock class="w-4 h-4 inline mr-2" />
            {{ $t('auth.confirmNewPassword') }}
          </label>
          <input
            v-model="confirmPassword"
            type="password"
            required
            minlength="8"
            class="input"
            placeholder="••••••••"
          />
        </div>

        <!-- Password strength indicator -->
        <div v-if="newPassword" class="space-y-1">
          <div class="flex gap-1">
            <div
              v-for="i in 4"
              :key="i"
              class="h-1 flex-1 rounded-full transition-colors"
              :class="passwordStrength >= i ? getStrengthColor(passwordStrength) : 'bg-gray-700'"
            ></div>
          </div>
          <p class="text-xs" :class="getStrengthTextColor(passwordStrength)">
            {{ getStrengthText(passwordStrength) }}
          </p>
        </div>

        <!-- Password mismatch warning -->
        <div v-if="confirmPassword && !passwordsMatch" class="flex items-center gap-2 p-4 rounded-xl bg-yellow-500/10 border border-yellow-500/30 text-yellow-400">
          <AlertCircle class="w-5 h-5" />
          <span>{{ $t('errors.passwordsMismatch') }}</span>
        </div>

        <div v-if="error" class="flex items-center gap-2 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400">
          <AlertCircle class="w-5 h-5" />
          <span>{{ error }}</span>
        </div>

        <div v-if="success" class="space-y-3">
          <div class="flex items-center gap-2 p-4 rounded-xl bg-green-500/10 border border-green-500/30 text-green-400">
            <CheckCircle class="w-5 h-5" />
            <div class="flex-1">
              <p class="font-medium">{{ $t('auth.passwordChanged') }}</p>
              <p class="text-sm mt-1">{{ $t('auth.redirectToDashboard') }}</p>
            </div>
          </div>

          <button
            type="button"
            @click="goToDashboard"
            class="btn btn-primary w-full"
          >
            {{ $t('auth.goToDashboard') }}
          </button>
        </div>

        <button
          v-if="!success"
          type="submit"
          :disabled="loading || !passwordsMatch || newPassword.length < 8 || !currentPassword"
          class="btn btn-primary w-full flex items-center justify-center gap-2"
        >
          <div v-if="loading" class="loading-spinner"></div>
          <Shield v-else class="w-5 h-5" />
          {{ loading ? $t('auth.changingPassword') : $t('auth.changePasswordButton') }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'
import { Shield, Lock, AlertCircle, CheckCircle } from 'lucide-vue-next'

const { t } = useI18n()

const router = useRouter()
const authStore = useAuthStore()

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')
const success = ref(false)

const passwordsMatch = computed(() => {
  if (!confirmPassword.value) return true
  return newPassword.value === confirmPassword.value
})

const passwordStrength = computed(() => {
  const password = newPassword.value
  if (!password) return 0

  let strength = 0

  // Length
  if (password.length >= 8) strength++
  if (password.length >= 12) strength++

  // Complexity
  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) strength++
  if (/\d/.test(password)) strength++
  if (/[^a-zA-Z\d]/.test(password)) strength++

  return Math.min(strength, 4)
})

function getStrengthColor(strength) {
  if (strength <= 1) return 'bg-red-500'
  if (strength === 2) return 'bg-yellow-500'
  if (strength === 3) return 'bg-blue-500'
  return 'bg-green-500'
}

function getStrengthTextColor(strength) {
  if (strength <= 1) return 'text-red-400'
  if (strength === 2) return 'text-yellow-400'
  if (strength === 3) return 'text-blue-400'
  return 'text-green-400'
}

function getStrengthText(strength) {
  if (strength <= 1) return t('auth.weakPassword')
  if (strength === 2) return t('auth.mediumPassword')
  if (strength === 3) return t('auth.goodPassword')
  return t('auth.strongPassword')
}

async function handleChangePassword() {
  if (!passwordsMatch.value) {
    error.value = t('errors.passwordsMismatch')
    return
  }

  if (newPassword.value.length < 8) {
    error.value = t('errors.passwordTooShort')
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await api.post('/users/change-password', {
      oldPassword: currentPassword.value,
      newPassword: newPassword.value
    })

    // Update tokens and clear mustChangePassword flag
    authStore.clearMustChangePassword(response.data.accessToken, response.data.refreshToken)

    // Load user profile to refresh permissions
    await authStore.loadUser()

    success.value = true

    // Auto-redirect after 2 seconds
    setTimeout(() => {
      router.push('/')
    }, 2000)
  } catch (err) {
    console.error('Password change failed:', err)
    error.value = err.response?.data?.error || t('errors.changePasswordFailed')
  } finally {
    loading.value = false
  }
}

function goToDashboard() {
  router.push('/')
}
</script>

<style scoped>
.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
