import { useCallback, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api'
import { flag } from '../geo'
import { stats } from '../stats'
import TripMap from '../components/TripMap'
import type { Item, Trip } from '../types'

// shortcut: with more than 8 trips the colors repeat; the legend tells them apart.
const PALETTE = ['#2563eb', '#dc2626', '#16a34a', '#9333ea', '#ea580c', '#0891b2', '#db2777', '#65a30d']
const names = new Intl.DisplayNames(['pt-BR'], { type: 'region' })
const today = () => new Date().toLocaleDateString('sv') // YYYY-MM-DD, local

function Stat({ n, label }: { n: number; label: string }) {
  return (
    <div className="rounded-lg bg-paper p-3">
      <span className="board block text-4xl font-bold leading-none text-ink">{n}</span>
      <span className="text-sm text-muted">{label}</span>
    </div>
  )
}

export default function Stats() {
  const nav = useNavigate()
  const { data: trips, error } = useQuery({ queryKey: ['trips'], queryFn: () => api<Trip[]>('/api/trips') })
  const { data: pins } = useQuery({ queryKey: ['pins'], queryFn: () => api<Item[]>('/api/pins') })
  const s = useMemo(() => stats(trips ?? [], today()), [trips])
  const color = useMemo(() => new Map(trips?.map((t, k) => [t.id, PALETTE[k % PALETTE.length]])), [trips])
  const colors = useMemo(() => new Map(pins?.map((i) => [i.id, color.get(i.trip_id) ?? PALETTE[0]])), [pins, color])
  const open = useCallback((i: Item) => nav(`/viagens/${i.trip_id}/itens/${i.id}`), [nav])
  const mapped = new Set(pins?.map((i) => i.trip_id))

  if (error) return <p role="alert" className="p-6 text-red-700">{error.message}</p>
  if (!trips) return null

  return (
    <main className="mx-auto max-w-2xl px-4 pb-24 pt-[max(1rem,env(safe-area-inset-top))]">
      <Link to="/" className="-ml-1 inline-flex min-h-11 items-center gap-1 pr-3 font-semibold text-ink"><span aria-hidden className="text-2xl leading-none">‹</span> Viagens</Link>
      <h1 className="board mt-2 text-4xl font-bold text-ink">Estatísticas</h1>

      <div className="mt-4 grid grid-cols-2 gap-2">
        <Stat n={s.done} label={s.done === 1 ? 'viagem feita' : 'viagens feitas'} />
        <Stat n={s.planned} label={s.planned === 1 ? 'viagem planejada' : 'viagens planejadas'} />
        <Stat n={s.days} label={s.days === 1 ? 'dia viajando' : 'dias viajando'} />
        <Stat n={s.visited.length} label={s.visited.length === 1 ? 'país visitado' : 'países visitados'} />
      </div>

      {s.visited.length > 0 && (
        <section className="mt-8">
          <h2 className="board text-3xl font-bold text-ink">Países</h2>
          <ul className="mt-2 divide-y divide-line">
            {s.visited.map((c) => (
              <li key={c.code} className="flex items-center gap-3 py-2">
                <span aria-hidden className="text-2xl">{flag(c.code)}</span>
                <span className="flex-1 font-semibold">{names.of(c.code.toUpperCase())}</span>
                <span className="text-sm text-muted">{c.trips === 1 ? '1 viagem' : `${c.trips} viagens`}</span>
              </li>
            ))}
          </ul>
          {s.upcoming.length > 0 && (
            <p className="mt-2 text-sm text-muted">Planejados: {s.upcoming.map((c) => `${flag(c)} ${names.of(c.toUpperCase())}`).join(', ')}</p>
          )}
        </section>
      )}

      <section className="mt-8">
        <h2 className="board text-3xl font-bold text-ink">Mapa de todas as viagens</h2>
        {pins && pins.length > 0 ? (
          <>
            <TripMap items={pins} colors={colors} onSelect={open} cluster className="mt-3 h-96 rounded-lg" />
            <ul className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-sm">
              {trips.filter((t) => mapped.has(t.id)).map((t) => (
                <li key={t.id} className="flex items-center gap-1.5">
                  <span aria-hidden className="inline-block h-3 w-3 rounded-full" style={{ background: color.get(t.id) }} />
                  {t.name}
                </li>
              ))}
            </ul>
          </>
        ) : (
          <p className="mt-2 text-sm text-muted">Os lugares com endereço aparecem aqui.</p>
        )}
      </section>
    </main>
  )
}
