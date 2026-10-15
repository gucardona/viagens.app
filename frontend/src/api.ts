import { classify, overlay, pending, queue, save } from './outbox'

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const method = (init.method ?? 'GET').toUpperCase()
  // Item, expense and checklist writes can wait in the outbox when there is no internet.
  const target = typeof init.body !== 'object' ? classify(method, path) : null
  const body = typeof init.body === 'string' ? JSON.parse(init.body) : undefined
  const later = () => {
    const q = queue(pending(), target!, body, path, method)
    save(q.ops)
    return q.result as T
  }
  if (target?.id != null && target.id < 0) return later() // still waiting to be created
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
