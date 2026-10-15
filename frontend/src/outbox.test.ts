import { describe, expect, it } from 'vitest'
import { classify, overlay, queue, flush, type Op } from './outbox'

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
    expect(result).toMatchObject({ id: -1, trip_id: 1, title: 'Museu', pending: true })
    expect(ops).toHaveLength(1)
    const next = queue(ops, { kind: 'item', action: 'create', tripId: 1 }, { title: 'Bar' }, '/api/trips/1/items', 'POST')
    expect(next.result?.id).toBe(-2)
  })
  it('folds edits and deletes of something created offline into the create', () => {
    let { ops } = queue([], { kind: 'item', action: 'create', tripId: 1 }, { title: 'Museu' }, '/api/trips/1/items', 'POST')
    ops = queue(ops, { kind: 'item', action: 'update', id: -1 }, { title: 'Museu do Fado' }, '/api/items/-1', 'PUT').ops
    expect(ops).toHaveLength(1)
    expect(ops[0].body).toEqual({ title: 'Museu do Fado' })
    ops = queue(ops, { kind: 'item', action: 'delete', id: -1 }, undefined, '/api/items/-1', 'DELETE').ops
    expect(ops).toEqual([])
  })
})

describe('overlay', () => {
  const ops: Op[] = [
    { kind: 'item', action: 'create', tripId: 1, id: -1, method: 'POST', path: '/api/trips/1/items', body: { title: 'Novo' } },
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
      { kind: 'item', action: 'create', tripId: 1, id: -1, method: 'POST', path: '/api/trips/1/items', body: { title: 'Novo' } },
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
