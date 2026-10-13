// Saving a trip for offline use writes straight to Cache Storage, so it works even before the service worker controls the page.
export type OfflineInfo = { at: string; files: number; bytes: number }

const key = (id: number) => `offline:${id}`

export function offlineInfo(id: number): OfflineInfo | null {
  try {
    return JSON.parse(localStorage.getItem(key(id)) ?? 'null')
  } catch {
    return null
  }
}

async function put(cacheName: string, url: string) {
  const res = await fetch(url, { credentials: 'same-origin' })
  if (!res.ok) throw new Error('Não foi possível baixar tudo. Tente de novo.')
  await (await caches.open(cacheName)).put(url, res)
}

export async function saveTripOffline(id: number, attachments: { id: number; size: number }[]): Promise<OfflineInfo> {
  const shell = ['/', ...[...document.querySelectorAll<HTMLScriptElement | HTMLLinkElement>('script[src], link[rel=stylesheet][href^="/"]')]
    .map((el) => ('src' in el ? el.src : el.href))]
  await Promise.all(shell.map((u) => put('shell-v1', u)))
  const api = ['/api/me', '/api/users', '/api/trips', `/api/trips/${id}`, `/api/trips/${id}/expenses`, `/api/trips/${id}/checklist`]
  await Promise.all(api.map((u) => put('data-v1', u)))
  // Attachments never change after upload, so only the ones not saved yet are downloaded.
  const data = await caches.open('data-v1')
  for (const a of attachments) if (!(await data.match(`/api/attachments/${a.id}`))) await put('data-v1', `/api/attachments/${a.id}`)
  const info = { at: new Date().toISOString(), files: attachments.length, bytes: attachments.reduce((n, a) => n + a.size, 0) }
  localStorage.setItem(key(id), JSON.stringify(info))
  return info
}
