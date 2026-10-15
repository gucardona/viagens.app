// Outbox: writes made without internet wait here (localStorage) and are sent when it comes back.
// Reads show the queued changes on top of what came from the network or the offline copy.

type Kind = 'item' | 'expense' | 'check'
type Target = { kind: Kind; action: 'create' | 'update' | 'delete'; id?: number; tripId?: number }
export type Op = Target & { method: string; path: string; body?: Record<string, unknown> }
type Row = { id: number; pending?: boolean } & Record<string, unknown>

const COLLECTIONS: Record<string, Kind> = { items: 'item', expenses: 'expense', checklist: 'check' }
const SINGLE: Record<string, Kind> = { items: 'item', expenses: 'expense', checklist: 'check' }

export function classify(method: string, path: string): Target | null {
  let m = path.match(/^\/api\/trips\/(\d+)\/(items|expenses|checklist)$/)
  if (m && method === 'POST') return { kind: COLLECTIONS[m[2]], action: 'create', tripId: Number(m[1]) }
  m = path.match(/^\/api\/(items|expenses|checklist)\/(-?\d+)$/)
  if (m && (method === 'PUT' || method === 'DELETE')) return { kind: SINGLE[m[1]], action: method === 'PUT' ? 'update' : 'delete', id: Number(m[2]) }
  return null
}

const DEFAULTS: Record<Kind, Record<string, unknown>> = {
  item: { country_code: '', area: '', created_at: '', updated_at: '' },
  expense: { brl_cents: null, created_at: '' },
  check: { done: false },
}

const created = (op: Op): Row => ({ ...DEFAULTS[op.kind], ...op.body, id: op.id!, trip_id: op.tripId, pending: true })

// queue adds a write to the outbox and returns what the caller would have got from the server.
// Edits and deletes of something still waiting to be created change that create instead.
export function queue(ops: Op[], t: Target, body: Record<string, unknown> | undefined, path: string, method: string): { ops: Op[]; result: Row | undefined } {
  if (t.action !== 'create' && t.id! < 0) {
    const i = ops.findIndex((o) => o.kind === t.kind && o.action === 'create' && o.id === t.id)
    if (i < 0) return { ops, result: undefined }
    if (t.action === 'delete') return { ops: ops.filter((_, k) => k !== i), result: undefined }
    const op = { ...ops[i], body: { ...ops[i].body, ...body } }
    return { ops: ops.map((o, k) => (k === i ? op : o)), result: created(op) }
  }
  const op: Op = { ...t, method, path, body }
  if (t.action === 'create') op.id = Math.min(0, ...ops.map((o) => o.id ?? 0)) - 1
  return { ops: [...ops, op], result: t.action === 'create' ? created(op) : t.action === 'update' ? { id: t.id!, ...body, pending: true } : undefined }
}

function patch(rows: Row[], ops: Op[], kind: Kind, tripId: number): Row[] {
  let out = rows
  for (const op of ops) {
    if (op.kind !== kind) continue
    if (op.action === 'create' && op.tripId === tripId) out = [...out, created(op)]
    if (op.action === 'update') out = out.map((r) => (r.id === op.id ? { ...r, ...op.body, pending: true } : r))
    if (op.action === 'delete') out = out.filter((r) => r.id !== op.id)
  }
  return out
}

export function overlay(path: string, data: unknown, ops: Op[]): unknown {
  if (!ops.length) return data
  let m = path.match(/^\/api\/trips\/(\d+)$/)
  if (m) {
    const d = data as { items: Row[] }
    return { ...d, items: patch(d.items, ops, 'item', Number(m[1])) }
  }
  m = path.match(/^\/api\/trips\/(\d+)\/expenses$/)
  if (m) {
    const d = data as { expenses: Row[] }
    return { ...d, expenses: patch(d.expenses, ops, 'expense', Number(m[1])) }
  }
  m = path.match(/^\/api\/trips\/(\d+)\/checklist$/)
  if (m) return patch(data as Row[], ops, 'check', Number(m[1]))
  return data
}

// flush sends the outbox in order. A network failure (or 401/5xx) stops and keeps the rest for later;
// any other refusal drops that write and reports the server's message.
export async function flush(ops: Op[], fetcher: (path: string, init: RequestInit) => Promise<Response>) {
  const ids = new Map<number, number>()
  const errors: string[] = []
  let k = 0
  for (; k < ops.length; k++) {
    const op = ops[k]
    const body = op.body && { ...op.body }
    if (body && typeof body.item_id === 'number' && body.item_id < 0) body.item_id = ids.get(body.item_id) ?? null
    let res: Response
    try {
      res = await fetcher(op.path, { method: op.method, body: body && JSON.stringify(body), headers: { 'X-Requested-With': 'viagens', 'Content-Type': 'application/json' }, credentials: 'same-origin' })
    } catch {
      break
    }
    if (res.status === 401 || res.status >= 500) break
    if (!res.ok) {
      errors.push((await res.json().catch(() => ({}))).error ?? 'Algo deu errado')
      continue
    }
    if (op.action === 'create') ids.set(op.id!, (await res.json()).id)
  }
  return { remaining: ops.slice(k), errors }
}

// Storage and change notification for the screens.
const KEY = 'outbox'
const listeners = new Set<() => void>()
let errors: string[] = []

export function pending(): Op[] {
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? '[]')
  } catch {
    return []
  }
}

export function save(ops: Op[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify(ops))
  } catch { /* storage full or blocked: the write is lost, as before this feature */ }
  listeners.forEach((l) => l())
}

export function subscribe(l: () => void) {
  listeners.add(l)
  return () => listeners.delete(l)
}

export const outboxErrors = () => errors
export function clearErrors() {
  errors = []
  listeners.forEach((l) => l())
}

let sending: Promise<boolean> | null = null
// send flushes the stored outbox once at a time; resolves true when something was sent.
export function send(): Promise<boolean> {
  if (sending) return sending
  const ops = pending()
  if (!ops.length) return Promise.resolve(false)
  sending = flush(ops, (p, i) => fetch(p, i)).then((r) => {
    // keep writes queued while we were sending
    save([...r.remaining, ...pending().slice(ops.length)])
    errors = [...errors, ...r.errors]
    sending = null
    return r.remaining.length < ops.length
  })
  return sending
}
