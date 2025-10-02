/**
 * WebAuthn/Passkey utilities for Holy Home
 */

/**
 * Check if WebAuthn is supported in the current browser
 */
export function isPasskeySupported() {
  return window.PublicKeyCredential !== undefined &&
         typeof window.PublicKeyCredential === 'function'
}

/**
 * Check if platform authenticator (like Touch ID, Face ID) is available
 */
export async function isPlatformAuthenticatorAvailable() {
  if (!isPasskeySupported()) return false

  try {
    return await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()
  } catch (err) {
    console.error('Error checking platform authenticator:', err)
    return false
  }
}

/**
 * Convert ArrayBuffer to Base64URL string
 */
function arrayBufferToBase64Url(buffer) {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '')
}

/**
 * Convert Base64URL string to ArrayBuffer
 */
function base64UrlToArrayBuffer(base64url) {
  const base64 = base64url
    .replace(/-/g, '+')
    .replace(/_/g, '/')
  const padLen = (4 - (base64.length % 4)) % 4
  const padded = base64 + '='.repeat(padLen)
  const binary = atob(padded)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer
}

/**
 * Prepare credential creation options from server response
 */
function prepareCreationOptions(serverOptions) {
  return {
    ...serverOptions,
    challenge: base64UrlToArrayBuffer(serverOptions.challenge),
    user: {
      ...serverOptions.user,
      id: base64UrlToArrayBuffer(serverOptions.user.id)
    },
    excludeCredentials: serverOptions.excludeCredentials?.map(cred => ({
      ...cred,
      id: base64UrlToArrayBuffer(cred.id)
    }))
  }
}

/**
 * Prepare assertion options from server response
 */
function prepareAssertionOptions(serverOptions) {
  return {
    ...serverOptions,
    challenge: base64UrlToArrayBuffer(serverOptions.challenge),
    allowCredentials: serverOptions.allowCredentials?.map(cred => ({
      ...cred,
      id: base64UrlToArrayBuffer(cred.id)
    }))
  }
}

/**
 * Format credential for server
 */
function formatCredentialForServer(credential) {
  return {
    id: credential.id,
    rawId: arrayBufferToBase64Url(credential.rawId),
    type: credential.type,
    response: {
      clientDataJSON: arrayBufferToBase64Url(credential.response.clientDataJSON),
      attestationObject: credential.response.attestationObject
        ? arrayBufferToBase64Url(credential.response.attestationObject)
        : undefined,
      authenticatorData: credential.response.authenticatorData
        ? arrayBufferToBase64Url(credential.response.authenticatorData)
        : undefined,
      signature: credential.response.signature
        ? arrayBufferToBase64Url(credential.response.signature)
        : undefined,
      userHandle: credential.response.userHandle
        ? arrayBufferToBase64Url(credential.response.userHandle)
        : undefined
    }
  }
}

/**
 * Register a new passkey
 * @param {Object} creationOptions - Options from server's begin registration endpoint
 * @returns {Promise<Object>} Formatted credential for server
 */
export async function registerPasskey(creationOptions) {
  if (!isPasskeySupported()) {
    throw new Error('Passkeys nie są obsługiwane w tej przeglądarce')
  }

  try {
    const options = prepareCreationOptions(creationOptions)
    const credential = await navigator.credentials.create({
      publicKey: options
    })

    if (!credential) {
      throw new Error('Nie udało się utworzyć passkey')
    }

    return formatCredentialForServer(credential)
  } catch (err) {
    console.error('Passkey registration failed:', err)

    if (err.name === 'NotAllowedError') {
      throw new Error('Rejestracja passkey została anulowana lub upłynął limit czasu')
    } else if (err.name === 'InvalidStateError') {
      throw new Error('Ten passkey jest już zarejestrowany')
    } else {
      throw new Error('Nie udało się zarejestrować passkey: ' + err.message)
    }
  }
}

/**
 * Authenticate with a passkey
 * @param {Object} assertionOptions - Options from server's begin login endpoint
 * @param {boolean} conditional - Use conditional mediation (for auto-login)
 * @param {AbortSignal} signal - Optional AbortSignal to cancel the request
 * @returns {Promise<Object>} Formatted credential for server
 */
export async function authenticateWithPasskey(assertionOptions, conditional = false, signal = null) {
  console.log('[Passkey] Starting authentication', { conditional, assertionOptions })

  if (!isPasskeySupported()) {
    throw new Error('Passkeys nie są obsługiwane w tej przeglądarce')
  }

  try {
    console.log('[Passkey] Preparing assertion options...')
    const options = prepareAssertionOptions(assertionOptions)
    console.log('[Passkey] Prepared options:', options)

    const credentialRequestOptions = {
      publicKey: options
    }

    // Add conditional mediation for auto-login (non-modal UI)
    if (conditional) {
      console.log('[Passkey] Using conditional mediation')
      credentialRequestOptions.mediation = 'conditional'
    }

    // Add abort signal if provided
    if (signal) {
      credentialRequestOptions.signal = signal
    }

    console.log('[Passkey] Calling navigator.credentials.get with:', credentialRequestOptions)
    const credential = await navigator.credentials.get(credentialRequestOptions)
    console.log('[Passkey] Got credential:', credential)

    if (!credential) {
      throw new Error('Nie udało się pobrać passkey')
    }

    console.log('[Passkey] Formatting credential for server...')
    const formatted = formatCredentialForServer(credential)
    console.log('[Passkey] Formatted credential:', formatted)
    return formatted
  } catch (err) {
    console.error('[Passkey] Authentication failed with error:', {
      name: err.name,
      message: err.message,
      stack: err.stack,
      err
    })

    if (err.name === 'AbortError') {
      throw new Error('Uwierzytelnianie passkey zostało przerwane')
    } else if (err.name === 'NotAllowedError') {
      throw new Error('Uwierzytelnianie passkey zostało anulowane lub upłynął limit czasu')
    } else {
      throw new Error('Nie udało się uwierzytelnić za pomocą passkey: ' + err.message)
    }
  }
}
