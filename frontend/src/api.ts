import { classify, knownIds, overlay, pending, queue, save } from './outbox'

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const method = (init.method ?? 'GET').toUpperCase()
  // Item, expense and checklist writes can wait in the outbox when there is no internet.
  let target = typeof init.body !== 'object' ? classify(method, path) : null
  const body = typeof init.body === 'string' ? JSON.parse(init.body) : undefined
  const later = () => {
    const q = queue(pending(), target!, body, path, method, knownIds())
    save(q.ops)
    return q.result as T
  }
  // Something created offline: once sent, a screen still holding its temporary id writes to the real one.
  const ids = knownIds()
  if (target?.id != null && target.id < 0) {
    if (ids[target.id] == null) return later() // still waiting to be created
    path = path.replace(/-\d+$/, String(ids[target.id]))
    target = { ...target, id: ids[target.id] }
  }
  if (body && typeof body.item_id === 'number' && body.item_id < 0 && ids[body.item_id] != null) {
    init = { ...init, body: JSON.stringify({ ...body, item_id: ids[body.item_id] }) }
  }
  const headers = new Headers(init.headers)
  headers.set('X-Requested-With', 'viagens')
  if (init.body && !(init.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  let res: Response
  try {
    res = await fetch(path, { ...init, headers, credentials: 'same-origin' })
  } catch (err) {
    if (target && !(init.signal?.aborted)) return later()
    throw err instanceof DOMException && err.name === 'AbortError' ? err : new Error('Sem internet. Tente de novo quando estiver online.')
  }
  if (res.status === 401 && location.pathname !== '/login') {
    location.assign('/login')
    return new Promise<T>(() => {})
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error ?? 'Algo deu errado. Tente de novo.')
  }
  if (res.status === 204) return undefined as T
  const data = await res.json()
  return (method === 'GET' ? overlay(path, data, pending()) : data) as T
}

export const json = (method: string, body: unknown): RequestInit => ({ method, body: JSON.stringify(body) })

// "2026-11-03" -> "03/11"
export const shortDate = (d: string) => (d ? `${d.slice(8, 10)}/${d.slice(5, 7)}` : '')
