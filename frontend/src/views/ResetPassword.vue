<template>
  <div class="min-h-screen flex items-center justify-center px-4">
    <div class="card max-w-md w-full">
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-purple-600 to-pink-600 mb-4">
          <Key class="w-8 h-8 text-white" />
        </div>
        <h1 class="text-4xl font-bold gradient-text mb-2">Resetowanie hasła</h1>
        <p class="text-gray-400">Ustaw nowe hasło dla swojego konta</p>
      </div>

      <!-- Loading state -->
      <div v-if="validatingToken" class="text-center py-8">
        <div class="loading-spinner mx-auto mb-4"></div>
        <p class="text-gray-400">Sprawdzanie linku resetowania...</p>
      </div>

      <!-- Invalid/Expired token state -->
      <div v-else-if="tokenError" class="space-y-4">
        <div class="flex items-center gap-2 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400">
          <AlertCircle class="w-5 h-5" />
          <div>
            <p class="font-medium">Link resetowania jest nieprawidłowy lub wygasł</p>
            <p class="text-sm mt-1">{{ tokenError }}</p>
          </div>
        </div>
        <button @click="goToLogin" class="btn btn-outline w-full">
          Wróć do logowania
        </button>
      </div>

      <!-- Reset password form -->
      <form v-else @submit.prevent="handleResetPassword" class="space-y-5">
        <div>
          <label class="block text-sm font-medium mb-2 text-gray-300">
            <Lock class="w-4 h-4 inline mr-2" />
            Nowe hasło
          </label>
          <input
            v-model="newPassword"
            type="password"
            required
            minlength="8"
            class="input"
            placeholder="••••••••"
          />
          <p class="text-xs text-gray-400 mt-1">Minimum 8 znaków</p>
        </div>

        <div>
          <label class="block text-sm font-medium mb-2 text-gray-300">
            <Lock class="w-4 h-4 inline mr-2" />
            Potwierdź nowe hasło
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

        <div v-if="error" class="flex items-center gap-2 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400">
          <AlertCircle class="w-5 h-5" />
          <span>{{ error }}</span>
        </div>

        <div v-if="success" class="space-y-3">
          <div class="flex items-center gap-2 p-4 rounded-xl bg-green-500/10 border border-green-500/30 text-green-400">
            <CheckCircle class="w-5 h-5" />
            <div class="flex-1">
              <p class="font-medium">Hasło zostało zmienione!</p>
              <p class="text-sm mt-1">Możesz teraz przejść do panelu.</p>
            </div>
          </div>

          <button
            type="button"
            @click="copyPassword"
            class="btn btn-outline w-full flex items-center justify-center gap-2"
          >
            <Copy class="w-5 h-5" />
            {{ passwordCopied ? 'Skopiowano!' : 'Skopiuj hasło' }}
          </button>

          <button
            type="button"
            @click="goToDashboard"
            class="btn btn-primary w-full"
          >
            Przejdź do panelu
          </button>
        </div>

        <button
          v-if="!success"
          type="submit"
          :disabled="loading || !passwordsMatch || newPassword.length < 8"
          class="btn btn-primary w-full flex items-center justify-center gap-2"
        >
          <div v-if="loading" class="loading-spinner"></div>
          <Key v-else class="w-5 h-5" />
          {{ loading ? 'Resetowanie...' : 'Resetuj hasło' }}
        </button>

        <button
          v-if="!success"
          type="button"
          @click="goToLogin"
          class="btn btn-outline w-full"
        >
          Anuluj
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'
import { Key, Lock, AlertCircle, CheckCircle, Copy } from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const token = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const validatingToken = ref(true)
const error = ref('')
const tokenError = ref('')
const success = ref(false)
const passwordCopied = ref(false)

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
  if (strength <= 1) return 'Słabe hasło'
  if (strength === 2) return 'Średnie hasło'
  if (strength === 3) return 'Dobre hasło'
  return 'Silne hasło'
}

onMounted(async () => {
  // Get token from URL query parameter
  token.value = route.query.token

  if (!token.value) {
    tokenError.value = 'Brak tokenu w linku resetowania'
    validatingToken.value = false
    return
  }

  // Validate token
  try {
    await api.get(`/auth/validate-reset-token?token=${token.value}`)
    validatingToken.value = false
  } catch (err) {
    console.error('Token validation failed:', err)
    tokenError.value = err.response?.data?.error || 'Token jest nieprawidłowy lub wygasł'
    validatingToken.value = false
  }
})

async function handleResetPassword() {
  if (!passwordsMatch.value) {
    error.value = 'Hasła nie są identyczne'
    return
  }

  if (newPassword.value.length < 8) {
    error.value = 'Hasło musi mieć co najmniej 8 znaków'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await api.post('/auth/reset-password', {
      token: token.value,
      newPassword: newPassword.value
    })

    // Store the tokens
    authStore.setTokens(response.data.accessToken, response.data.refreshToken)

    // Load user profile
    await authStore.loadUser()

    success.value = true
  } catch (err) {
    console.error('Password reset failed:', err)
    error.value = err.response?.data?.error || 'Nie udało się zresetować hasła'
  } finally {
    loading.value = false
  }
}

function copyPassword() {
  if (newPassword.value) {
    navigator.clipboard.writeText(newPassword.value).then(() => {
      passwordCopied.value = true
      setTimeout(() => {
        passwordCopied.value = false
      }, 2000)
    }).catch(err => {
      console.error('Failed to copy password:', err)
    })
  }
}

function goToDashboard() {
  router.push('/')
}

function goToLogin() {
  router.push('/login')
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
