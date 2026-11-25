import { Platform } from 'react-native'

const SERVICE = 'com.openslash.tandemesh.session'
const USERNAME = 'tandemesh'
const STORAGE_KEY = 'tandemesh.mobile.session'
const isHarmony = (Platform.OS as string) === 'harmony'

export interface MobileSession {
  host: string
  token: string
}

function keychain(): typeof import('react-native-keychain') {
  // Loaded lazily so HarmonyOS never initializes the unsupported native module.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require('react-native-keychain') as typeof import('react-native-keychain')
}

function harmonyStorage(): {
  getItem: (key: string) => Promise<string | null>
  setItem: (key: string, value: string) => Promise<void>
  removeItem: (key: string) => Promise<void>
} {
  // Harmony port of AsyncStorage (RNCAsyncStorage backed by RDB).
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require('@react-native-ohos/async-storage').default
}

export async function saveSession(host: string, token: string): Promise<void> {
  const payload = JSON.stringify({ host, token })
  if (isHarmony) {
    await harmonyStorage().setItem(STORAGE_KEY, payload)
    return
  }
  await keychain().setGenericPassword(USERNAME, payload, { service: SERVICE })
}

export async function loadSession(): Promise<MobileSession | null> {
  try {
    let raw: string | null = null
    if (isHarmony) {
      raw = await harmonyStorage().getItem(STORAGE_KEY)
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
    await harmonyStorage().removeItem(STORAGE_KEY)
    return
  }
  await keychain().resetGenericPassword({ service: SERVICE })
}
