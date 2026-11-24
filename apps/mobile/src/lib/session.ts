// Session storage seam. The first scaffold keeps the token in memory; the next
// step replaces this with Keychain / Keystore / HarmonyOS secure storage.

let token: string | null = null
let host: string | null = null

export function saveSession(nextHost: string, nextToken: string) {
  host = nextHost
  token = nextToken
}

export function clearSession() {
  host = null
  token = null
}

export function session() {
  return token ? { host: host ?? '', token } : null
}
