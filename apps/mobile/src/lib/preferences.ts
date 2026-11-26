import { Platform } from 'react-native'

const isHarmony = (Platform.OS as string) === 'harmony'
const KEY_PREFIX = 'tandemesh.pref.'
const KEYCHAIN_SERVICE_PREFIX = 'com.openslash.tandemesh.pref.'

function harmonyStorage(): {
  getItem: (key: string) => Promise<string | null>
  setItem: (key: string, value: string) => Promise<void>
} {
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require('@react-native-ohos/async-storage').default
}

function keychain(): typeof import('react-native-keychain') {
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require('react-native-keychain') as typeof import('react-native-keychain')
}

export async function loadPreference(key: string): Promise<string | null> {
  try {
    if (isHarmony) {
      return await harmonyStorage().getItem(KEY_PREFIX + key)
    }
    const stored = await keychain().getGenericPassword({ service: KEYCHAIN_SERVICE_PREFIX + key })
    return stored ? stored.password : null
  } catch {
    return null
  }
}

export async function savePreference(key: string, value: string): Promise<void> {
  try {
    if (isHarmony) {
      await harmonyStorage().setItem(KEY_PREFIX + key, value)
      return
    }
    await keychain().setGenericPassword(key, value, { service: KEYCHAIN_SERVICE_PREFIX + key })
  } catch {
    // preferences are best-effort; the app still works with defaults
  }
}
