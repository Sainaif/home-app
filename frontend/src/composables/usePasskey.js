import { ref } from 'vue'
import api from '../api/client'
import { registerPasskey, authenticateWithPasskey, isPasskeySupported, isPlatformAuthenticatorAvailable } from '../utils/passkey'

export function usePasskey() {
  const loading = ref(false)
  const error = ref(null)
  const passkeys = ref([])

  /**
   * Check if passkeys are supported
   */
  const checkSupport = async () => {
    const supported = isPasskeySupported()
    const platformAvailable = await isPlatformAuthenticatorAvailable()
    return { supported, platformAvailable }
  }

  /**
   * List all registered passkeys
   */
  const listPasskeys = async () => {
    loading.value = true
    error.value = null
    try {
      const response = await api.get('/auth/passkeys')
      passkeys.value = response.data
      return response.data
    } catch (err) {
      error.value = err.response?.data?.error || 'Nie udało się pobrać listy passkeys'
      throw err
    } finally {
      loading.value = false
    }
  }

  /**
   * Register a new passkey
   */
  const register = async (name = 'My Passkey') => {
    loading.value = true
    error.value = null
    try {
      // Step 1: Begin registration
      const beginResponse = await api.post('/auth/passkey/register/begin')
      const creationOptions = beginResponse.data.publicKey

      // Step 2: Create credential using WebAuthn
      const credential = await registerPasskey(creationOptions)

      // Step 3: Finish registration
      await api.post('/auth/passkey/register/finish', {
        name,
        credential
      })

      // Refresh passkey list
      await listPasskeys()

      return true
    } catch (err) {
      error.value = err.message || err.response?.data?.error || 'Nie udało się zarejestrować passkey'
      throw err
    } finally {
      loading.value = false
    }
  }

  /**
   * Authenticate with a passkey
   * @param {string} email - Email (can be empty for discoverable credentials)
   * @param {boolean} conditional - Use conditional mediation for auto-login
   * @param {AbortSignal} signal - Optional AbortSignal to cancel the request
   */
  const login = async (email = '', conditional = false, signal = null) => {
    loading.value = true
    error.value = null
    try {
      // Step 1: Begin login (email can be empty for discoverable)
      const beginResponse = await api.post('/auth/passkey/login/begin', { email: email || '' })
      const assertionOptions = beginResponse.data.publicKey

      // Step 2: Get credential using WebAuthn
      const credential = await authenticateWithPasskey(assertionOptions, conditional, signal)

      // Step 3: Finish login
      const finishResponse = await api.post('/auth/passkey/login/finish', {
        email: email || '',
        credential
      })

      return finishResponse.data // Returns { access, refresh, mustChangePassword }
    } catch (err) {
      error.value = err.message || err.response?.data?.error || 'Nie udało się zalogować za pomocą passkey'
      throw err
    } finally {
      loading.value = false
    }
  }

  /**
   * Login with conditional UI - properly handles conditional mediation
   * @param {AbortSignal} signal - Optional AbortSignal to cancel conditional mediation
   * @param {boolean} useConditional - Whether to use conditional mediation (default true)
   */
  const loginWithConditionalUI = async (signal = null, useConditional = true) => {
    loading.value = true
    error.value = null
    try {
      // Step 1: Begin discoverable login
      const beginResponse = await api.post('/auth/passkey/login/begin', { email: '' })
      const assertionOptions = beginResponse.data.publicKey

      // Step 2: Get credential with conditional or modal UI
      const credential = await authenticateWithPasskey(assertionOptions, useConditional, signal)

      // Step 3: Finish login
      const finishResponse = await api.post('/auth/passkey/login/finish', {
        email: '',
        credential
      })

      return finishResponse.data
    } catch (err) {
      error.value = err.message || err.response?.data?.error || 'Nie udało się zalogować za pomocą passkey'
      throw err
    } finally {
      loading.value = false
    }
  }

  /**
   * Delete a passkey
   */
  const deletePasskey = async (credentialId) => {
    loading.value = true
    error.value = null
    try {
      await api.delete('/auth/passkeys', {
        data: { credentialId }
      })

      // Refresh passkey list
      await listPasskeys()

      return true
    } catch (err) {
      error.value = err.response?.data?.error || 'Nie udało się usunąć passkey'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    passkeys,
    checkSupport,
    listPasskeys,
    register,
    login,
    loginWithConditionalUI,
    deletePasskey
  }
}
