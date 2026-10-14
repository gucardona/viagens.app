export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('X-Requested-With', 'viagens')
  if (init.body && !(init.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  const res = await fetch(path, { ...init, headers, credentials: 'same-origin' }).catch(() => {
    throw new Error('Sem internet. Tente de novo quando estiver online.')
  })
  if (res.status === 401 && location.pathname !== '/login') {
    location.assign('/login')
    return new Promise<T>(() => {})
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error ?? 'Algo deu errado. Tente de novo.')
  }
  return res.status === 204 ? (undefined as T) : res.json()
}

export const json = (method: string, body: unknown): RequestInit => ({ method, body: JSON.stringify(body) })

// "2026-11-03" -> "03/11"
export const shortDate = (d: string) => (d ? `${d.slice(8, 10)}/${d.slice(5, 7)}` : '')
