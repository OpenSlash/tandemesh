/** Calls to the desktop shell's own server (not the plane), for updates. */

export interface DesktopUpdate {
  version: string
  /** False for dev builds and platforms without self-update. */
  supported: boolean
  available?: string
  notes?: string
}

async function desktopRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    // The shell rejects requests without this header, so other sites cannot
    // trigger an install through the local server.
    headers: { 'X-Tandemesh-Desktop': '1', ...init?.headers },
  })
  const body = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error((body as { error?: string }).error ?? response.statusText)
  return body as T
}

export const desktopApi = {
  checkUpdate: () => desktopRequest<DesktopUpdate>('/desktop/update'),
  installUpdate: () => desktopRequest<{ version: string }>('/desktop/update/install', { method: 'POST' }),
}
