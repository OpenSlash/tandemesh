import { Platform } from 'react-native'

const SERVICE = 'com.openslash.tandemesh.session'
const USERNAME = 'tandemesh'
const STORAGE_KEY = 'tandemesh.mobile.session'
const isHarmony = (Platform.OS as string) === 'harmony'

export interface MobileSession {
  host: string
  token: string
}

interface HarmonySecureStore {
  set(key: string, value: string): Promise<string>
  get(key: string): Promise<string>
  remove(key: string): Promise<string>
}

function keychain(): typeof import('react-native-keychain') {
  // Loaded lazily so HarmonyOS never initializes the unsupported native module.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require('react-native-keychain') as typeof import('react-native-keychain')
}

function harmonySecure(): HarmonySecureStore {
  // HUKS-backed adapter: ciphertext is encrypted with a hardware-protected key.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require('@react-native-ohos/react-native-secure-key-store').default as HarmonySecureStore
}

function harmonyLegacy(): {
  getItem: (key: string) => Promise<string | null>
  setItem: (key: string, value: string) => Promise<void>
  removeItem: (key: string) => Promise<void>
} {
  // Old AsyncStorage token, kept only for migration.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require('@react-native-ohos/async-storage').default
}

async function loadHarmonySession(): Promise<string | null> {
  try {
    const secure = await harmonySecure().get(STORAGE_KEY)
    if (secure) return secure
  } catch {
    // No secure entry yet.
  }
  try {
    const legacy = await harmonyLegacy().getItem(STORAGE_KEY)
    if (legacy) {
      await harmonySecure().set(STORAGE_KEY, legacy)
      await harmonyLegacy().removeItem(STORAGE_KEY)
      return legacy
    }
  } catch {
    // ignore migration failures
  }
  return null
}

export async function saveSession(host: string, token: string): Promise<void> {
  const payload = JSON.stringify({ host, token })
  if (isHarmony) {
    await harmonySecure().set(STORAGE_KEY, payload)
    await harmonyLegacy().removeItem(STORAGE_KEY).catch(() => undefined)
    return
  }
  await keychain().setGenericPassword(USERNAME, payload, { service: SERVICE })
}

export async function loadSession(): Promise<MobileSession | null> {
  try {
    let raw: string | null = null
    if (isHarmony) {
      raw = await loadHarmonySession()
    } else {
      const stored = await keychain().getGenericPassword({ service: SERVICE })
      raw = stored ? stored.password : null
    }
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<MobileSession>
    if (!parsed.host || !parsed.token) return null
    return { host: parsed.host, token: parsed.token }
  } catch {
    return null
  }
}

export async function clearSession(): Promise<void> {
  if (isHarmony) {
    await harmonySecure().remove(STORAGE_KEY).catch(() => undefined)
    await harmonyLegacy().removeItem(STORAGE_KEY).catch(() => undefined)
    return
  }
  await keychain().resetGenericPassword({ service: SERVICE })
}
