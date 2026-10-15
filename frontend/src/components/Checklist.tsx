import { useState, type FormEvent } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, json } from '../api'
import type { CheckItem, Trip } from '../types'

export default function Checklist({ tripId }: { tripId: number }) {
  const qc = useQueryClient()
  const key = ['checklist', String(tripId)]
  const { data: list } = useQuery({ queryKey: key, queryFn: () => api<CheckItem[]>(`/api/trips/${tripId}/checklist`) })
  const { data: trips } = useQuery({ queryKey: ['trips'], queryFn: () => api<Trip[]>('/api/trips') })
  const [text, setText] = useState('')
  const [error, setError] = useState('')
  const refresh = () => qc.invalidateQueries({ queryKey: key })
  const run = async (p: Promise<unknown>) => {
    setError('')
    try {
      await p
    } catch (err) {
      setError((err as Error).message)
    }
    refresh()
  }

  function add(e: FormEvent) {
    e.preventDefault()
    if (!text.trim()) return
    run(api(`/api/trips/${tripId}/checklist`, json('POST', { text }))).then(() => setText(''))
  }

  const others = trips?.filter((t) => t.id !== tripId) ?? []
  const done = list?.filter((c) => c.done).length ?? 0

  return (
    <section>
      <h2 className="board text-3xl font-bold text-ink">Checklist {list?.length ? <span className="text-lg text-muted">{done}/{list.length}</span> : null}</h2>
      <ul className="mt-2 space-y-1">
        {list?.map((c) => (
          <li key={c.id} className="flex items-center gap-3 rounded-lg bg-paper p-3">
            <input type="checkbox" className="h-5 w-5 shrink-0" checked={c.done} aria-label={c.text}
              onChange={() => {
                qc.setQueryData<CheckItem[]>(key, (l) => l?.map((x) => (x.id === c.id ? { ...x, done: !c.done } : x)))
                run(api(`/api/checklist/${c.id}`, json('PUT', { text: c.text, done: !c.done })))
              }} />
            <span className={`flex-1 ${c.done ? 'text-muted line-through' : ''}`}>{c.text}{(c as { pending?: boolean }).pending && <span className="ml-1 text-xs font-normal text-muted">⏳ esperando internet</span>}</span>
            <button className="text-sm text-muted" aria-label={`Apagar ${c.text}`} onClick={() => run(api(`/api/checklist/${c.id}`, { method: 'DELETE' }))}>✕</button>
          </li>
        ))}
      </ul>
      <form onSubmit={add} className="mt-2 flex gap-2">
        <input className="field" value={text} maxLength={200} onChange={(e) => setText(e.target.value)} placeholder="Passaporte, adaptador de tomada…" />
        <button className="btn shrink-0">Adicionar</button>
      </form>
      {others.length > 0 && (
        <label className="mt-2 flex items-center gap-2 text-sm text-muted">
          Copiar de
          <select className="field flex-1 py-1!" value="" onChange={(e) => e.target.value && run(api(`/api/trips/${tripId}/checklist/copy`, json('POST', { from_trip_id: Number(e.target.value) })))}>
            <option value="">outra viagem…</option>
            {others.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
          </select>
        </label>
      )}
      {error && <p role="alert" className="mt-1 text-sm text-red-700">{error}</p>}
    </section>
  )
}
