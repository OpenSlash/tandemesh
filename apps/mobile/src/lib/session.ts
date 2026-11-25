import * as Keychain from 'react-native-keychain'

const SERVICE = 'com.openslash.tandemesh.session'
const USERNAME = 'tandemesh'

export interface MobileSession {
  host: string
  token: string
}

export async function saveSession(host: string, token: string): Promise<void> {
  await Keychain.setGenericPassword(USERNAME, JSON.stringify({ host, token }), { service: SERVICE })
}

export async function loadSession(): Promise<MobileSession | null> {
  const stored = await Keychain.getGenericPassword({ service: SERVICE })
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
  await Keychain.resetGenericPassword({ service: SERVICE })
}
