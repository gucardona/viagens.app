import { describe, expect, it } from 'vitest'
import { classify, overlay, queue, flush, reconcile, type Op } from './outbox'

const item = (id: number, title: string) => ({ id, trip_id: 1, title, kind: 'activity' })

describe('classify', () => {
  it('recognizes the writes that can wait for internet', () => {
    expect(classify('POST', '/api/trips/1/items')).toEqual({ kind: 'item', action: 'create', tripId: 1 })
    expect(classify('PUT', '/api/items/7')).toEqual({ kind: 'item', action: 'update', id: 7 })
    expect(classify('DELETE', '/api/expenses/3')).toEqual({ kind: 'expense', action: 'delete', id: 3 })
    expect(classify('POST', '/api/trips/2/checklist')).toEqual({ kind: 'check', action: 'create', tripId: 2 })
    expect(classify('PUT', '/api/checklist/-4')).toEqual({ kind: 'check', action: 'update', id: -4 })
  })
  it('leaves everything else online-only', () => {
    expect(classify('POST', '/api/trips/1/attachments')).toBeNull()
    expect(classify('POST', '/api/trips/1/checklist/copy')).toBeNull()
    expect(classify('DELETE', '/api/trips/1')).toBeNull()
    expect(classify('POST', '/api/trips/1/import')).toBeNull()
  })
})

describe('queue', () => {
  it('gives creates a temporary negative id', () => {
    const { ops, result } = queue([], { kind: 'item', action: 'create', tripId: 1 }, { title: 'Museu' }, '/api/trips/1/items', 'POST')
    expect(result).toMatchObject({ trip_id: 1, title: 'Museu', pending: true })
    expect(result!.id).toBeLessThan(0)
    expect(ops).toHaveLength(1)
    const next = queue(ops, { kind: 'item', action: 'create', tripId: 1 }, { title: 'Bar' }, '/api/trips/1/items', 'POST')
    expect(next.result!.id).toBeLessThan(result!.id)
  })
  it('folds edits and deletes of something created offline into the create', () => {
    let { ops, result } = queue([], { kind: 'item', action: 'create', tripId: 1 }, { title: 'Museu' }, '/api/trips/1/items', 'POST')
    const t = result!.id
    ops = queue(ops, { kind: 'item', action: 'update', id: t }, { title: 'Museu do Fado' }, `/api/items/${t}`, 'PUT').ops
    expect(ops).toHaveLength(1)
    expect(ops[0].body).toEqual({ title: 'Museu do Fado' })
    ops = queue(ops, { kind: 'item', action: 'delete', id: t }, undefined, `/api/items/${t}`, 'DELETE').ops
    expect(ops).toEqual([])
  })
  it('sends edits of something already created to its real id', () => {
    const { ops } = queue([], { kind: 'item', action: 'update', id: -5 }, { title: 'X' }, '/api/items/-5', 'PUT', { '-5': 42 })
    expect(ops).toEqual([expect.objectContaining({ action: 'update', id: 42, path: '/api/items/42', body: { title: 'X' } })])
  })
})

describe('overlay', () => {
  const ops: Op[] = [
    { kind: 'item', action: 'create', tripId: 1, id: -1, seq: 1, method: 'POST', path: '/api/trips/1/items', body: { title: 'Novo' } },
    { kind: 'item', action: 'create', tripId: 2, id: -2, method: 'POST', path: '/api/trips/2/items', body: { title: 'Outra viagem' } },
    { kind: 'item', action: 'update', id: 10, method: 'PUT', path: '/api/items/10', body: { title: 'Editado' } },
    { kind: 'item', action: 'delete', id: 11, method: 'DELETE', path: '/api/items/11' },
    { kind: 'expense', action: 'create', tripId: 1, id: -3, method: 'POST', path: '/api/trips/1/expenses', body: { title: 'Táxi', amount_cents: 2000 } },
    { kind: 'check', action: 'update', id: 5, method: 'PUT', path: '/api/checklist/5', body: { text: 'Passaporte', done: true } },
  ]
  it('shows queued changes on top of the trip', () => {
    const trip = { trip: { id: 1 }, items: [item(10, 'Velho'), item(11, 'Apagado')], attachments: [] }
    const got = overlay('/api/trips/1', trip, ops) as typeof trip
    expect(got.items.map((i) => i.title)).toEqual(['Editado', 'Novo'])
    expect(got.items[1]).toMatchObject({ id: -1, trip_id: 1, pending: true })
  })
  it('shows queued expenses and checklist changes', () => {
    const s = overlay('/api/trips/1/expenses', { expenses: [], total_cents: 0 }, ops) as { expenses: { title: string; pending?: boolean }[] }
    expect(s.expenses).toEqual([expect.objectContaining({ title: 'Táxi', pending: true })])
    const c = overlay('/api/trips/1/checklist', [{ id: 5, text: 'Passaporte', done: false }], ops)
    expect(c).toEqual([{ id: 5, text: 'Passaporte', done: true, pending: true }])
  })
  it('leaves other reads untouched', () => {
    const trips = [{ id: 1 }]
    expect(overlay('/api/trips', trips, ops)).toBe(trips)
  })
})

describe('flush', () => {
  const res = (status: number, body: unknown) => new Response(JSON.stringify(body), { status })
  it('sends in order, swaps temporary ids, and drops what the server refuses', async () => {
    const ops: Op[] = [
      { kind: 'item', action: 'create', tripId: 1, id: -1, seq: 1, method: 'POST', path: '/api/trips/1/items', body: { title: 'Novo' } },
      { kind: 'expense', action: 'create', tripId: 1, id: -2, method: 'POST', path: '/api/trips/1/expenses', body: { title: 'Ingresso', item_id: -1 } },
      { kind: 'item', action: 'delete', id: 99, method: 'DELETE', path: '/api/items/99' },
    ]
    const sent: { path: string; body: unknown }[] = []
    const fake = async (path: string, init: RequestInit) => {
      sent.push({ path, body: init.body ? JSON.parse(init.body as string) : undefined })
      if (path === '/api/trips/1/items') return res(201, { id: 42 })
      if (path === '/api/items/99') return res(404, { error: 'Item não encontrado' })
      return res(201, { id: 7 })
    }
    const out = await flush(ops, fake)
    expect(sent[1].body).toEqual({ title: 'Ingresso', item_id: 42 })
    expect(out.remaining).toEqual([])
    expect(out.errors).toEqual(['Item não encontrado'])
  })
  it('stops at the first network failure and keeps the rest', async () => {
    const ops: Op[] = [
      { kind: 'check', action: 'update', id: 5, method: 'PUT', path: '/api/checklist/5', body: { text: 'a', done: true } },
      { kind: 'check', action: 'delete', id: 6, method: 'DELETE', path: '/api/checklist/6' },
    ]
    const out = await flush(ops, async () => { throw new TypeError('Failed to fetch') })
    expect(out.remaining).toEqual(ops)
    expect(out.errors).toEqual([])
  })
})

describe('flush across reconnects', () => {
  it('keeps the link to an item created in an earlier flush', async () => {
    const ops: Op[] = [{ kind: 'expense', action: 'create', tripId: 1, id: -9, seq: 1, method: 'POST', path: '/api/trips/1/expenses', body: { item_id: -3 } }]
    let sent: unknown
    await flush(ops, async (_p, init) => ((sent = JSON.parse(init.body as string)), new Response('{"id":5}', { status: 201 })), { '-3': 77 })
    expect(sent).toEqual({ item_id: 77 })
  })
})

describe('reconcile', () => {
  const op = (seq: number, id: number, body: Record<string, unknown>): Op =>
    ({ kind: 'item', action: 'create', tripId: 1, id, seq, method: 'POST', path: '/api/trips/1/items', body })
  it('keeps writes made while sending and turns changes to sent creates into real writes', () => {
    const A = op(1, -1, { title: 'A' }), B = op(2, -2, { title: 'B' })
    const C = op(3, -3, { title: 'C' })
    // while A and B were being sent: B deleted, C created, A edited
    const current = [{ ...A, body: { title: 'A2' } }, C]
    const next = reconcile([A, B], 2, current, { '-1': 10, '-2': 11 })
    expect(next).toEqual([
      expect.objectContaining({ action: 'update', id: 10, method: 'PUT', path: '/api/items/10', body: { title: 'A2' } }),
      C,
      expect.objectContaining({ action: 'delete', id: 11, method: 'DELETE', path: '/api/items/11' }),
    ])
  })
  it('keeps what was not sent', () => {
    const A = op(1, -1, { title: 'A' }), B = op(2, -2, { title: 'B' })
    expect(reconcile([A, B], 1, [A, B], { '-1': 10 })).toEqual([B])
  })
})
