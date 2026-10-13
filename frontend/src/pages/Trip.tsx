import { useQuery } from '@tanstack/react-query'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { buildDays, timeOn, type DayGroup } from '../days'
import { KINDS } from '../kinds'
import type { TripDetail } from '../types'

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

function Day({ day, tripId }: { day: DayGroup; tripId: string }) {
  return (
    <section className="py-5">
      <DayHeader date={day.date} />
      {day.items.length === 0 ? (
        <p className="mt-2 text-sm text-muted">Nada planejado.</p>
      ) : (
        <ul className="mt-3 space-y-2">
          {day.items.map((i) => {
            const checkout = day.date && i.kind === 'lodging' && day.date !== i.start_date
            const time = day.date ? timeOn(i, day.date) : ''
            return (
              <li key={`${i.id}-${day.date}`}>
                <Link to={`/viagens/${tripId}/itens/${i.id}`} className="flex gap-3 rounded-lg bg-paper p-3" style={{ borderLeft: `4px solid ${KINDS[i.kind].color}` }}>
                  <span className="board w-12 shrink-0 pt-0.5 text-lg text-ink">{time || '—'}</span>
                  <span className="min-w-0">
                    <span className="block font-semibold">
                      <span aria-hidden>{KINDS[i.kind].emoji} </span>
                      {checkout ? `Check-out: ${i.title}` : i.title}
                    </span>
                    {i.address && <span className="block truncate text-sm text-muted">{i.address}</span>}
                  </span>
                </Link>
              </li>
            )
          })}
        </ul>
      )}
    </section>
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
        {data.trip.destination && <p className="text-muted">{data.trip.destination}</p>}
      </header>

      {tab === 'roteiro' && (
        <div className="divide-y divide-line">
          {buildDays(data.trip, data.items).map((d) => <Day key={d.date ?? 'none'} day={d} tripId={id} />)}
        </div>
      )}

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
