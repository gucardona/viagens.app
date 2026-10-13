import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { api, shortDate } from '../api'
import { buildDays, pinNumbers, timeLabel, type DayGroup } from '../days'
import { flag } from '../geo'
import { KINDS } from '../kinds'
import DayMap from '../components/DayMap'
import type { Item, TripDetail } from '../types'
import TripMapTab from './TripMapTab'
import DocumentsTab from './DocumentsTab'

const weekday = new Intl.DateTimeFormat('pt-BR', { weekday: 'long', timeZone: 'UTC' })
const month = new Intl.DateTimeFormat('pt-BR', { month: 'long', timeZone: 'UTC' })

function DayHeader({ date }: { date: string | null }) {
  if (!date) return <h2 className="board text-3xl font-bold text-ink">Sem data</h2>
  const d = new Date(date + 'T00:00:00Z')
  return (
    <h2 className="flex items-baseline gap-3">
      <span className="board text-5xl font-bold leading-none text-ink">{date.slice(8, 10)}</span>
      <span className="text-muted">{weekday.format(d)}, {month.format(d)}</span>
    </h2>
  )
}

const TICKET = {
  needed: { label: '🎫 comprar', className: 'bg-signal/40 text-ink' },
  bought: { label: '🎫 comprado', className: 'bg-green-100 text-green-800' },
}

function Day({ day, tripId }: { day: DayGroup; tripId: string }) {
  const numbers = useMemo(() => pinNumbers(day.items), [day.items])
  const pinned = useMemo(() => day.items.filter((i) => i.lat != null), [day.items])
  return (
    <section className="py-5">
      <DayHeader date={day.date} />
      {day.date && pinned.length > 0 && <DayMap items={pinned} numbers={numbers} to={`/viagens/${tripId}?aba=mapa&dia=${day.date}`} />}
      {day.items.length === 0 ? (
        <p className="mt-2 text-sm text-muted">Nada planejado.</p>
      ) : (
        <ul className="mt-3 space-y-2">
          {day.items.map((i) => {
            const checkout = day.date && i.kind === 'lodging' && day.date !== i.start_date
            const [start, until] = (day.date ? timeLabel(i, day.date) : '').split('–')
            const n = numbers.get(i.id)
            return (
              <li key={`${i.id}-${day.date}`}>
                <Link to={`/viagens/${tripId}/itens/${i.id}`} className="flex gap-3 rounded-lg bg-paper p-3" style={{ borderLeft: `4px solid ${KINDS[i.kind].color}` }}>
                  <span className={`board w-14 shrink-0 pt-0.5 text-lg ${i.time_approx ? 'text-muted' : 'text-ink'}`}>{start || '—'}</span>
                  <span className="min-w-0 flex-1">
                    <span className="block font-semibold">
                      <span aria-hidden>{KINDS[i.kind].emoji} </span>
                      {checkout ? `Check-out: ${i.title}` : i.title}
                    </span>
                    {until && <span className="board block text-ink">até {until}</span>}
                    {i.address && <span className="block truncate text-sm text-muted">{i.address}</span>}
                    {i.ticket && <span className={`mt-1 inline-block rounded px-1.5 text-sm font-semibold ${TICKET[i.ticket].className}`}>{TICKET[i.ticket].label}</span>}
                  </span>
                  {n && (
                    <span className="board grid h-6 w-6 shrink-0 place-items-center rounded-full text-sm font-bold text-white" style={{ background: KINDS[i.kind].color }} aria-label={`Ponto ${n} no mapa`}>{n}</span>
                  )}
                </Link>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}

function TicketsToBuy({ items, tripId }: { items: Item[]; tripId: string }) {
  const [open, setOpen] = useState(false)
  const todo = items.filter((i) => i.ticket === 'needed').sort((a, b) => (a.start_date || '9').localeCompare(b.start_date || '9'))
  if (!todo.length) return null
  return (
    <div className="mt-4 rounded-lg bg-signal/30 p-3">
      <button onClick={() => setOpen(!open)} aria-expanded={open} className="flex w-full items-center justify-between font-semibold text-ink">
        <span>🎫 {todo.length === 1 ? '1 ingresso para comprar' : `${todo.length} ingressos para comprar`}</span>
        <span aria-hidden>{open ? '▴' : '▾'}</span>
      </button>
      {open && (
        <ul className="mt-2 space-y-1">
          {todo.map((i) => (
            <li key={i.id}>
              <Link to={`/viagens/${tripId}/itens/${i.id}`} className="flex gap-3 rounded bg-paper p-2">
                <span className="board w-12 shrink-0 text-ink">{shortDate(i.start_date) || '—'}</span>
                <span className="truncate">{i.title}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

const TABS = [
  { key: 'roteiro', label: 'Roteiro' },
  { key: 'mapa', label: 'Mapa' },
  { key: 'documentos', label: 'Documentos' },
] as const

export default function Trip() {
  const { id = '' } = useParams()
  const [params, setParams] = useSearchParams()
  const tab = params.get('aba') ?? 'roteiro'
  const { data, error } = useQuery({ queryKey: ['trip', id], queryFn: () => api<TripDetail>(`/api/trips/${id}`) })

  const days = useMemo(() => (data ? buildDays(data.trip, data.items) : []), [data])
  const flags = useMemo(() => [...new Set(data?.items.map((i) => i.country_code).filter(Boolean))].sort().map(flag).join(' '), [data])

  if (error) return <p role="alert" className="p-6 text-red-700">{error.message}</p>
  if (!data) return null

  return (
    <div className="mx-auto max-w-2xl px-4 pb-28 pt-6">
      <header className="mb-2">
        <Link to="/" className="text-ink">Viagens</Link>
        <div className="mt-2 flex items-start justify-between gap-4">
          <h1 className="board text-4xl font-bold leading-tight text-ink">{data.trip.name}</h1>
          <Link to={`/viagens/${id}/editar`} className="btn btn-quiet shrink-0">Editar</Link>
        </div>
        {(data.trip.destination || flags) && <p className="text-muted">{data.trip.destination} {flags && <span aria-label="Países">{flags}</span>}</p>}
      </header>

      {tab === 'roteiro' && (
        <>
          <TicketsToBuy items={data.items} tripId={id} />
          <div className="divide-y divide-line">
            {days.map((d) => <Day key={d.date ?? 'none'} day={d} tripId={id} />)}
          </div>
        </>
      )}

      {tab === 'mapa' && <TripMapTab data={data} />}
      {tab === 'documentos' && <DocumentsTab data={data} />}

      {tab === 'roteiro' && (
        <Link to={`/viagens/${id}/itens/novo`} className="btn fixed bottom-20 right-4 z-10 bg-signal! text-ink! shadow-lg">+ Item</Link>
      )}

      <nav className="fixed inset-x-0 bottom-0 z-20 flex border-t border-line bg-paper pb-[env(safe-area-inset-bottom)]">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setParams({ aba: t.key }, { replace: true })}
            aria-current={tab === t.key ? 'page' : undefined}
            className={`flex-1 py-3 font-semibold ${tab === t.key ? 'text-ink shadow-[inset_0_3px_0_var(--color-signal)]' : 'text-muted'}`}
          >
            {t.label}
          </button>
        ))}
      </nav>
    </div>
  )
}
