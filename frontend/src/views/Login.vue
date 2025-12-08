<template>
  <div class="min-h-screen flex items-center justify-center px-4 relative">
    <!-- Language Switcher in top-right corner -->
    <div class="absolute top-4 right-4">
      <LanguageSwitcher :showLabel="false" />
    </div>

    <div class="card max-w-md w-full">
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-purple-600 to-pink-600 mb-4">
          <Home class="w-8 h-8 text-white" />
        </div>
        <h1 class="text-4xl font-bold gradient-text mb-2">{{ appName }}</h1>
        <p class="text-gray-400">{{ $t('dashboard.overview') }}</p>
      </div>

      <form @submit.prevent="handleLogin" class="space-y-5">
        <div>
          <label class="block text-sm font-medium mb-2 text-gray-300">
            <Mail v-if="authConfig.allowEmailLogin && !authConfig.allowUsernameLogin" class="w-4 h-4 inline mr-2" />
            <User v-else-if="!authConfig.allowEmailLogin && authConfig.allowUsernameLogin" class="w-4 h-4 inline mr-2" />
            <AtSign v-else class="w-4 h-4 inline mr-2" />
            {{ identifierLabel }}
          </label>
          <input
            v-model="identifier"
            :type="authConfig.allowEmailLogin && !authConfig.allowUsernameLogin ? 'email' : 'text'"
            required
            class="input"
            :placeholder="identifierPlaceholder"
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
            <span class="px-2 bg-gray-800 text-gray-400">{{ $t('auth.or') }}</span>
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
          {{ loading ? $t('common.loading') : $t('auth.passkeyLogin') }}
        </button>
      </form>
    </div>

    <!-- Migration Panel (shown when MIGRATION_MODE is enabled) -->
    <div class="absolute bottom-4 left-4 right-4 max-w-md mx-auto">
      <MigrationPanel />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useAppSettingsStore } from '../stores/appSettings'
import { usePasskey } from '../composables/usePasskey'
import { Home, Mail, Lock, LogIn, AlertCircle, Key, User, AtSign } from 'lucide-vue-next'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import MigrationPanel from '../components/MigrationPanel.vue'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()
const appSettingsStore = useAppSettingsStore()
const { checkSupport, loginWithConditionalUI } = usePasskey()
const authConfig = computed(() => authStore.authConfig)
const appName = computed(() => appSettingsStore.appName)

const identifier = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const passkeySupported = ref(false)

const identifierLabel = computed(() => {
  if (authConfig.value.allowEmailLogin && !authConfig.value.allowUsernameLogin) {
    return t('auth.email')
  } else if (!authConfig.value.allowEmailLogin && authConfig.value.allowUsernameLogin) {
    return t('auth.username')
  }
  return t('auth.emailOrUsername')
})

const identifierPlaceholder = computed(() => {
  if (authConfig.value.allowEmailLogin && !authConfig.value.allowUsernameLogin) {
    return 'john@example.com'
  } else if (!authConfig.value.allowEmailLogin && authConfig.value.allowUsernameLogin) {
    return 'john_doe'
  }
  return t('auth.identifierPlaceholder')
})

onMounted(async () => {
  // Fetch auth configuration
  await authStore.fetchAuthConfig()

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
    await authStore.login(identifier.value, password.value)
    router.push('/')
  } catch (err) {
    error.value = err.response?.data?.error || t('auth.loginError')
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
    error.value = err.message || t('auth.loginError')
  } finally {
    loading.value = false
  }
}
</script>
