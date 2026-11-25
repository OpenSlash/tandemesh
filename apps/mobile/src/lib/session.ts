import { Platform } from 'react-native'

const SERVICE = 'com.openslash.tandemesh.session'
const USERNAME = 'tandemesh'
const isHarmony = Platform.OS === 'harmony'

export interface MobileSession {
  host: string
  token: string
}

// RNOH does not ship the react-native-keychain native module. Keep the session
// in memory on HarmonyOS until a Harmony secure-storage bridge is added.
let harmonySession: MobileSession | null = null

function keychain(): typeof import('react-native-keychain') {
  // Loaded lazily so HarmonyOS never initializes the unsupported native module.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require('react-native-keychain') as typeof import('react-native-keychain')
}

export async function saveSession(host: string, token: string): Promise<void> {
  if (isHarmony) {
    harmonySession = { host, token }
    return
  }
  await keychain().setGenericPassword(USERNAME, JSON.stringify({ host, token }), { service: SERVICE })
}

export async function loadSession(): Promise<MobileSession | null> {
  if (isHarmony) return harmonySession
  const stored = await keychain().getGenericPassword({ service: SERVICE })
  if (!stored) return null
  try {
    const parsed = JSON.parse(stored.password) as Partial<MobileSession>
    if (!parsed.host || !parsed.token) return null
    return { host: parsed.host, token: parsed.token }
  } catch {
    return null
  }
}

export async function clearSession(): Promise<void> {
  if (isHarmony) {
    harmonySession = null
    return
  }
  await keychain().resetGenericPassword({ service: SERVICE })
}
