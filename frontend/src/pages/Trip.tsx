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
import ExpensesTab from './ExpensesTab'
import MoreTab from './MoreTab'
import OfflineBar from '../components/OfflineBar'
import SavedTab from './SavedTab'
import ChatTab from './ChatTab'
import { navFor, type Bar, type View } from '../nav'

const weekday = new Intl.DateTimeFormat('pt-BR', { weekday: 'long', timeZone: 'UTC' })
const month = new Intl.DateTimeFormat('pt-BR', { month: 'long', timeZone: 'UTC' })

export function DayHeader({ date }: { date: string | null }) {
  if (!date) return <h2 className="board text-3xl font-bold text-ink">Sem data</h2>
  const d = new Date(date + 'T00:00:00Z')
  return (
    <h2 className="flex items-baseline gap-3">
      <span className="board text-5xl font-bold leading-none text-ink">{date.slice(8, 10)}</span>
      <span className="text-muted">{weekday.format(d)}, {month.format(d)}</span>
    </h2>
  )
}

const ROW = 'flex gap-3 rounded-lg bg-paper p-3'

const TICKET = {
  needed: { label: '🎫 comprar', className: 'bg-signal/40 text-ink' },
  bought: { label: '🎫 comprado', className: 'bg-green-100 text-green-800' },
}

// Without tripId (shared link) the day is read-only: no links to items or to the big map.
export function Day({ day, tripId }: { day: DayGroup; tripId?: string }) {
  const numbers = useMemo(() => pinNumbers(day.items), [day.items])
  const pinned = useMemo(() => day.items.filter((i) => i.lat != null), [day.items])
  return (
    <section className="py-5">
      <DayHeader date={day.date} />
      {day.date && pinned.length > 0 && <DayMap items={pinned} numbers={numbers} to={tripId && `/viagens/${tripId}?aba=mapa&dia=${day.date}`} />}
      {day.items.length === 0 ? (
        <p className="mt-2 text-sm text-muted">Nada planejado.</p>
      ) : (
        <ul className="mt-3 space-y-2">
          {day.items.map((i) => {
            const checkout = day.date && i.kind === 'lodging' && day.date !== i.start_date
            const [start, until] = (day.date ? timeLabel(i, day.date) : '').split('–')
            const n = numbers.get(i.id)
            const row = (
              <>
                <span className={`board w-14 shrink-0 pt-0.5 text-lg ${i.time_approx ? 'text-muted' : 'text-ink'}`}>{start || '—'}</span>
                <span className="min-w-0 flex-1">
                  <span className="block font-semibold">
                    <span aria-hidden>{KINDS[i.kind].emoji} </span>
                    {checkout ? `Check-out: ${i.title}` : i.title}{(i as { pending?: boolean }).pending && <span className="ml-1 text-xs font-normal text-muted">⏳ esperando internet</span>}
                  </span>
                  {until && <span className="board block text-ink">até {until}</span>}
                  {i.address && <span className="block truncate text-sm text-muted">{i.address}</span>}
                  {i.ticket && <span className={`mt-1 inline-block rounded px-1.5 text-sm font-semibold ${TICKET[i.ticket].className}`}>{TICKET[i.ticket].label}</span>}
                </span>
                {n && (
                  <span className="board grid h-6 w-6 shrink-0 place-items-center rounded-full text-sm font-bold text-white" style={{ background: KINDS[i.kind].color }} aria-label={`Ponto ${n} no mapa`}>{n}</span>
                )}
              </>
            )
            return (
              <li key={`${i.id}-${day.date}`}>
                {tripId ? (
                  <Link to={`/viagens/${tripId}/itens/${i.id}`} className={ROW} style={{ borderLeft: `4px solid ${KINDS[i.kind].color}` }}>{row}</Link>
                ) : (
                  <div className={ROW} style={{ borderLeft: `4px solid ${KINDS[i.kind].color}` }}>{row}</div>
                )}
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

const icon = (d: string) => (
  <svg aria-hidden viewBox="0 0 24 24" className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d={d} /></svg>
)

const TABS: { key: Bar; label: string; icon: string }[] = [
  { key: 'roteiro', label: 'Roteiro', icon: 'M8 6h12M8 12h12M8 18h12M4 6h.01M4 12h.01M4 18h.01' },
  { key: 'mapa', label: 'Mapa', icon: 'M9 4 3 6v14l6-2 6 2 6-2V4l-6 2-6-2zM9 4v14M15 6v14' },
  { key: 'gastos', label: 'Gastos', icon: 'M2 6h20v12H2zM12 9.5a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5zM6 12h.01M18 12h.01' },
  { key: 'mais', label: 'Mais', icon: 'M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z' },
]

// Dias and Salvos are two views of the same plan: what has a day and what is still an idea.
function PlanSwitch({ view, go }: { view: View; go: (aba: View) => void }) {
  return (
    <div className="mt-4 mb-2 grid grid-cols-2 rounded-md bg-ink/8 p-1" role="group" aria-label="Ver roteiro">
      {(['roteiro', 'salvos'] as const).map((v) => (
        <button key={v} onClick={() => go(v)} aria-pressed={view === v}
          className={`board min-h-10 rounded-[0.6rem] text-lg font-semibold ${view === v ? 'bg-paper text-ink shadow-sm' : 'text-muted'}`}>
          {v === 'roteiro' ? 'Dias' : 'Salvos'}
        </button>
      ))}
    </div>
  )
}

export default function Trip() {
  const { id = '' } = useParams()
  const [params, setParams] = useSearchParams()
  const { view: tab, bar } = navFor(params.get('aba'))
  const go = (aba: View) => setParams({ aba }, { replace: true })
  const { data, error } = useQuery({ queryKey: ['trip', id], queryFn: () => api<TripDetail>(`/api/trips/${id}`) })

  const days = useMemo(() => (data ? buildDays(data.trip, data.items).filter((d) => d.date) : []), [data])
  const flags = useMemo(() => [...new Set(data?.items.map((i) => i.country_code).filter(Boolean))].sort().map(flag).join(' '), [data])

  if (error) return <p role="alert" className="p-6 text-red-700">{error.message}</p>
  if (!data) return null

  return (
    <div className={`mx-auto max-w-2xl px-4 ${tab === 'chat' ? 'pb-44' : 'pb-32'} pt-[max(1rem,env(safe-area-inset-top))]`}>
      <header className="mb-2">
        <div className="flex items-center justify-between">
          <Link to="/" className="-ml-1 inline-flex min-h-11 items-center gap-1 pr-3 font-semibold text-ink"><span aria-hidden className="text-2xl leading-none">‹</span> Viagens</Link>
          <button onClick={() => go('chat')} aria-pressed={tab === 'chat'}
            className={`inline-flex min-h-11 items-center gap-2 rounded-full px-4 font-semibold ${tab === 'chat' ? 'bg-ink text-white' : 'bg-paper text-ink'}`}>
            {icon('M21 12a8 8 0 0 1-11.6 7.1L4 20l1-4.6A8 8 0 1 1 21 12z')} Assistente
          </button>
        </div>
        <div className="mt-2 flex items-start justify-between gap-4">
          <h1 className="board text-4xl font-bold leading-tight text-ink">{data.trip.name}</h1>
          <Link to={`/viagens/${id}/editar`} className="btn btn-quiet shrink-0">Editar</Link>
        </div>
        {(data.trip.destination || flags) && <p className="text-muted">{data.trip.destination} {flags && <span aria-label="Países">{flags}</span>}</p>}
      </header>
      <OfflineBar data={data} />

      {bar === 'roteiro' && <PlanSwitch view={tab} go={go} />}

      {tab === 'roteiro' && (
        <>
          <TicketsToBuy items={data.items} tripId={id} />
          <div className="divide-y divide-line">
            {days.map((d) => <Day key={d.date ?? 'none'} day={d} tripId={id} />)}
          </div>
        </>
      )}

      {tab === 'salvos' && <SavedTab data={data} />}
      {tab === 'mapa' && <TripMapTab data={data} />}
      {tab === 'gastos' && <ExpensesTab data={data} />}
      {tab === 'chat' && <ChatTab tripId={data.trip.id} />}
      {tab === 'mais' && <MoreTab data={data} />}

      {/* Floating pill with the destinations and, apart from it, the round + (after gastos.app). */}
      <nav className="fixed inset-x-0 bottom-[calc(1rem+env(safe-area-inset-bottom))] z-20 px-3.5">
        <div className="mx-auto flex max-w-2xl items-center gap-2.5">
          <div className="flex h-[3.6rem] min-w-0 flex-1 items-center justify-around rounded-full bg-paper/90 px-1 shadow-[0_8px_28px_rgba(18,53,91,0.14),0_0_0_1px_rgba(18,53,91,0.06)] backdrop-blur-md">
            {TABS.map((t) => (
              <button
                key={t.key}
                onClick={() => go(t.key)}
                aria-current={bar === t.key ? 'page' : undefined}
                className={`flex min-w-11 flex-1 flex-col items-center gap-0.5 rounded-full py-1 ${bar === t.key ? 'text-ink' : 'text-muted'}`}
              >
                <span className={`grid h-7 w-12 place-items-center rounded-full ${bar === t.key ? 'bg-signal/45' : ''}`}>{icon(t.icon)}</span>
                <span className={`text-[0.7rem] leading-none ${bar === t.key ? 'font-semibold' : 'font-medium'}`}>{t.label}</span>
              </button>
            ))}
          </div>
          {tab === 'roteiro' && (
            <Link to={`/viagens/${id}/itens/novo`} aria-label="Novo item"
              className="grid h-[3.6rem] w-[3.6rem] shrink-0 place-items-center rounded-full bg-ink text-white shadow-[0_8px_22px_rgba(18,53,91,0.35)]">
              <svg aria-hidden viewBox="0 0 24 24" className="h-7 w-7" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round"><path d="M12 5v14M5 12h14" /></svg>
            </Link>
          )}
        </div>
      </nav>
    </div>
  )
}
