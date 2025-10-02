<template>
  <div class="min-h-screen flex items-center justify-center px-4">
    <div class="card max-w-md w-full">
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-purple-600 to-pink-600 mb-4">
          <Home class="w-8 h-8 text-white" />
        </div>
        <h1 class="text-4xl font-bold gradient-text mb-2">Holy Home</h1>
        <p class="text-gray-400">Zarządzanie gospodarstwem domowym</p>
      </div>

      <form @submit.prevent="handleLogin" class="space-y-5">
        <div>
          <label class="block text-sm font-medium mb-2 text-gray-300">
            <Mail class="w-4 h-4 inline mr-2" />
            {{ $t('auth.email') }}
          </label>
          <input
            v-model="email"
            type="email"
            required
            class="input"
            placeholder="admin@example.pl"
          />
        </div>

        <div>
          <label class="block text-sm font-medium mb-2 text-gray-300">
            <Lock class="w-4 h-4 inline mr-2" />
            {{ $t('auth.password') }}
          </label>
          <input
            v-model="password"
            type="password"
            required
            class="input"
            placeholder="••••••••"
          />
        </div>

        <div v-if="error" class="flex items-center gap-2 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400">
          <AlertCircle class="w-5 h-5" />
          <span>{{ error }}</span>
        </div>

        <button type="submit" :disabled="loading" class="btn btn-primary w-full flex items-center justify-center gap-2">
          <div v-if="loading" class="loading-spinner"></div>
          <LogIn v-else class="w-5 h-5" />
          {{ loading ? $t('common.loading') : $t('auth.loginButton') }}
        </button>

        <!-- Passkey Login Option -->
        <div v-if="passkeySupported" class="relative">
          <div class="absolute inset-0 flex items-center">
            <div class="w-full border-t border-gray-700"></div>
          </div>
          <div class="relative flex justify-center text-sm">
            <span class="px-2 bg-gray-800 text-gray-400">lub</span>
          </div>
        </div>

        <button
          v-if="passkeySupported"
          type="button"
          @click="handlePasskeyLogin"
          :disabled="loading"
          class="btn btn-outline w-full flex items-center justify-center gap-2">
          <div v-if="loading" class="loading-spinner"></div>
          <Key v-else class="w-5 h-5" />
          {{ loading ? 'Logowanie...' : 'Zaloguj się passkey' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { usePasskey } from '../composables/usePasskey'
import { Home, Mail, Lock, LogIn, AlertCircle, Key } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const { checkSupport, loginWithConditionalUI } = usePasskey()

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const passkeySupported = ref(false)

onMounted(async () => {
  const support = await checkSupport()
  passkeySupported.value = support.supported

  // Auto-trigger passkey login if user previously used passkey
  // This provides a one-click experience similar to conditional mediation
  const hasUsedPasskey = localStorage.getItem('hasUsedPasskey')
  if (passkeySupported.value && hasUsedPasskey === 'true') {
    // Small delay to let the page render
    setTimeout(() => {
      console.log('[Login] Auto-triggering passkey login for returning user')
      handlePasskeyLogin()
    }, 500)
  }
})

async function handleLogin() {
  loading.value = true
  error.value = ''

  try {
    await authStore.login(email.value, password.value)
    router.push('/')
  } catch (err) {
    error.value = err.response?.data?.error || 'Błąd logowania'
  } finally {
    loading.value = false
  }
}

async function handlePasskeyLogin() {
  loading.value = true
  error.value = ''

  try {
    // Use modal UI for manual button click
    const tokens = await loginWithConditionalUI(null, false)

    // Remember that user has used passkey for auto-login next time
    localStorage.setItem('hasUsedPasskey', 'true')

    authStore.setTokens(tokens.access, tokens.refresh)
    await authStore.loadUser()
    router.push('/')
  } catch (err) {
    error.value = err.message || 'Błąd logowania passkey'
  } finally {
    loading.value = false
  }
}
</script>
