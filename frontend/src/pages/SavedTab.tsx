import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { api, json, shortDate } from '../api'
import { cluster, dayTimes, walkOrder } from '../cluster'
import { KINDS } from '../kinds'
import TripMap from '../components/TripMap'
import ImportDialog from '../components/ImportDialog'
import type { Item, TripDetail } from '../types'

type Pinned = Item & { lat: number; lng: number }

const PALETTE = ['#2563eb', '#db2777', '#16a34a', '#ea580c', '#7c3aed', '#0891b2', '#ca8a04', '#be123c']
const ALONE = '#64748b'

const addDay = (d: string) => new Date(Date.parse(d + 'T00:00:00Z') + 86400000).toISOString().slice(0, 10)

function tripDays(start: string, end: string) {
  const out: string[] = []
  for (let d = start; d <= end && out.length < 366; d = addDay(d)) out.push(d)
  return out
}

function mostCommon(values: string[]) {
  const count = new Map<string, number>()
  values.filter(Boolean).forEach((v) => count.set(v, (count.get(v) ?? 0) + 1))
  return [...count].sort((a, b) => b[1] - a[1])[0]?.[0] ?? ''
}

function Place({ i, tripId, color }: { i: Item; tripId: number; color: string }) {
  return (
    <li>
      <Link to={`/viagens/${tripId}/itens/${i.id}`} className="flex items-center gap-3 px-4 py-3">
        <span aria-hidden className="inline-block h-2.5 w-2.5 shrink-0 rounded-full" style={{ background: color }} />
        <span aria-hidden>{KINDS[i.kind].emoji}</span>
        <span className="min-w-0">
          <span className="block truncate font-semibold">{i.title}{(i as { pending?: boolean }).pending && <span className="ml-1 text-xs font-normal text-muted">⏳ esperando internet</span>}</span>
          {(i.area || i.notes) && <span className="block truncate text-sm text-muted">{i.area || i.notes}</span>}
        </span>
      </Link>
    </li>
  )
}

function ToDay({ items, data }: { items: Pinned[]; data: TripDetail }) {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const days = tripDays(data.trip.start_date, data.trip.end_date)

  async function move(date: string) {
    setBusy(true)
    setError('')
    const times = dayTimes(items.length)
    try {
      for (const [k, i] of walkOrder(items).entries()) {
        await api(`/api/items/${i.id}`, json('PUT', {
          kind: i.kind, title: i.title, start_date: date, start_time: times[k], end_date: '', end_time: '',
          address: i.address, lat: i.lat, lng: i.lng, confirmation_code: i.confirmation_code, notes: i.notes,
          ticket: i.ticket, time_approx: times[k] !== '',
        }))
      }
      setOpen(false)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
      qc.invalidateQueries({ queryKey: ['trip', String(data.trip.id)] })
    }
  }

  if (!open) return <button type="button" className="text-sm font-semibold text-ink underline" onClick={() => setOpen(true)}>Levar para um dia</button>
  return (
    <div className="mt-2 w-full">
      <p className="text-sm text-muted">Escolha o dia. Os lugares entram na ordem de menor caminhada, com horários aproximados a partir das 10:00.</p>
      <div className="mt-2 flex flex-wrap gap-2">
        {days.map((d) => (
          <button key={d} type="button" disabled={busy} onClick={() => move(d)} className="board rounded-full border border-line bg-paper px-3 py-1 text-lg">{shortDate(d)}</button>
        ))}
        <button type="button" onClick={() => setOpen(false)} className="px-2 text-sm text-muted">Cancelar</button>
      </div>
      {error && <p role="alert" className="mt-1 text-sm text-red-700">{error}</p>}
    </div>
  )
}

export default function SavedTab({ data }: { data: TripDetail }) {
  const qc = useQueryClient()
  const [km, setKm] = useState(1)
  const [importing, setImporting] = useState(false)
  const [msg, setMsg] = useState('')
  const saved = useMemo(() => data.items.filter((i) => !i.start_date), [data.items])
  const pinned = useMemo(() => saved.filter((i) => i.lat != null) as Pinned[], [saved])
  const unpinned = saved.filter((i) => i.lat == null)
  const { groups, alone } = useMemo(() => {
    const c = cluster(pinned, km)
    return { groups: c.groups.sort((a, b) => b.length - a.length), alone: c.alone }
  }, [pinned, km])
  const colors = useMemo(() => {
    const m = new Map<number, string>()
    groups.forEach((g, k) => g.forEach((i) => m.set(i.id, PALETTE[k % PALETTE.length])))
    alone.forEach((i) => m.set(i.id, ALONE))
    return m
  }, [groups, alone])

  async function pasteLink() {
    const url = prompt('Cole o link do lugar no Google Maps')
    if (!url) return
    setMsg('')
    try {
      const r = await api<{ title: string; lat: number | null; lng: number | null }>('/api/maps-link', json('POST', { url }))
      const title = r.title || prompt('Nome do lugar')?.trim()
      if (!title) return
      const res = await api<{ created: number }>(`/api/trips/${data.trip.id}/import`, json('POST', [{ title, url, lat: r.lat, lng: r.lng }]))
      setMsg(res.created ? `"${title}" salvo${r.lat == null ? '. Sem local no link: vou procurar pelo nome.' : '.'}` : `"${title}" já estava salvo.`)
      qc.invalidateQueries({ queryKey: ['trip', String(data.trip.id)] })
    } catch (err) {
      setMsg((err as Error).message)
    }
  }

  return (
    <div className="mt-2">
      <div className="flex gap-2 text-sm">
        <button type="button" className="min-h-10 rounded-full border border-line px-4 font-medium text-ink" onClick={pasteLink}>Colar link do Maps</button>
        <button type="button" className="min-h-10 rounded-full border border-line px-4 font-medium text-ink" onClick={() => setImporting(true)}>Importar do Google</button>
      </div>
      {msg && <p role="status" className="mt-2 text-sm">{msg}</p>}
      {importing && <ImportDialog tripId={data.trip.id} onClose={() => setImporting(false)} />}

      {saved.length === 0 ? (
        <p className="mt-10 text-center text-muted">Nenhum lugar salvo ainda. Toque no + para guardar uma ideia sem dia, ou traga lugares do Google Maps.</p>
      ) : (
        <>
          {pinned.length > 0 && <TripMap items={pinned} colors={colors} cluster className="mt-4 h-64" />}
          <label className="mt-3 flex items-center gap-3 text-sm">
            <span className="shrink-0 text-muted">Agrupar a até</span>
            <input type="range" min={0.5} max={3} step={0.5} value={km} onChange={(e) => setKm(Number(e.target.value))} className="flex-1" />
            <span className="board w-12 shrink-0 text-lg text-ink">{String(km).replace('.', ',')} km</span>
          </label>

          {groups.map((g, k) => (
            <section key={g[0].id} className="mt-5">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <h3 className="flex items-center gap-2 font-semibold">
                  <span className="inline-block h-3 w-3 rounded-full" style={{ background: PALETTE[k % PALETTE.length] }} />
                  {mostCommon(g.map((i) => i.area)) || `Grupo ${k + 1}`} · {g.length} lugares
                </h3>
                <ToDay items={g} data={data} />
              </div>
              <ul className="mt-2 divide-y divide-line overflow-hidden rounded-lg bg-paper">{g.map((i) => <Place key={i.id} i={i} tripId={data.trip.id} color={PALETTE[k % PALETTE.length]} />)}</ul>
            </section>
          ))}
          {alone.length > 0 && (
            <section className="mt-5">
              <h3 className="font-semibold">Avulsos · {alone.length}</h3>
              <ul className="mt-2 divide-y divide-line overflow-hidden rounded-lg bg-paper">{alone.map((i) => <Place key={i.id} i={i} tripId={data.trip.id} color={ALONE} />)}</ul>
            </section>
          )}
          {unpinned.length > 0 && (
            <section className="mt-5">
              <h3 className="font-semibold">Sem local · {unpinned.length}</h3>
              <p className="text-sm text-muted">Abra o lugar e toque no mapa para marcar onde ele fica.</p>
              <ul className="mt-2 divide-y divide-line overflow-hidden rounded-lg bg-paper">{unpinned.map((i) => <Place key={i.id} i={i} tripId={data.trip.id} color={ALONE} />)}</ul>
            </section>
          )}
        </>
      )}
    </div>
  )
}
