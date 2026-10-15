import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api, shortDate } from '../api'
import { CATEGORIES, money, type Category } from '../money'
import type { ExpenseSummary, TripDetail, User } from '../types'

export const useExpenses = (tripId: number | string) =>
  useQuery({ queryKey: ['expenses', String(tripId)], queryFn: () => api<ExpenseSummary>(`/api/trips/${tripId}/expenses`) })
export const useUsers = () => useQuery({ queryKey: ['users'], queryFn: () => api<User[]>('/api/users'), staleTime: Infinity })

export default function ExpensesTab({ data }: { data: TripDetail }) {
  const { data: s, error } = useExpenses(data.trip.id)
  const { data: users } = useUsers()
  const name = (id: number) => users?.find((u) => u.id === id)?.name ?? ''
  if (error) return <p role="alert" className="mt-6 text-red-700">{error.message}</p>
  if (!s) return null
  const max = Math.max(1, ...Object.values(s.by_category))
  const days = [...new Set(s.expenses.map((e) => e.date))]

  return (
    <div className="mt-2">
      <div className="rounded-lg bg-ink p-4 text-white">
        <p className="text-sm opacity-80">Total da viagem</p>
        <p className="board text-4xl font-bold">{money(s.total_cents)}</p>
        <p className="mt-1">
          {s.balance ? <>{s.balance.from} deve <b>{money(s.balance.cents)}</b> para {s.balance.to}</> : s.expenses.length ? 'Tudo certo entre vocês' : ''}
        </p>
        {s.missing_rate > 0 && <p className="mt-1 text-sm text-signal">{s.missing_rate} gasto(s) sem cotação ficaram fora do total. Abra e informe a cotação.</p>}
      </div>

      <Link to={`/viagens/${data.trip.id}/gastos/novo`} className="btn mt-3 block text-center">+ Gasto</Link>

      {s.expenses.length === 0 ? (
        <p className="mt-10 text-center text-muted">Nenhum gasto ainda.</p>
      ) : (
        <>
          <ul className="mt-5 space-y-2">
            {(Object.keys(CATEGORIES) as Category[]).filter((c) => s.by_category[c]).map((c) => (
              <li key={c}>
                <div className="flex justify-between text-sm"><span>{CATEGORIES[c].emoji} {CATEGORIES[c].label}</span><span className="board text-base">{money(s.by_category[c])}</span></div>
                <div className="mt-1 h-2 rounded-full bg-line"><div className="h-2 rounded-full" style={{ width: `${(100 * s.by_category[c]) / max}%`, background: CATEGORIES[c].color }} /></div>
              </li>
            ))}
          </ul>

          {days.map((d) => (
            <section key={d} className="mt-6">
              <h3 className="flex justify-between border-b border-line pb-1">
                <span className="board text-2xl font-bold text-ink">{shortDate(d)}</span>
                <span className="board text-lg text-muted">{money(s.by_day[d] ?? 0)}</span>
              </h3>
              <ul className="mt-2 space-y-1">
                {s.expenses.filter((e) => e.date === d).map((e) => (
                  <li key={e.id}>
                    <Link to={`/viagens/${data.trip.id}/gastos/${e.id}`} className="flex justify-between gap-3 rounded-lg bg-paper p-3">
                      <span className="min-w-0">
                        <span className="block truncate font-semibold">{CATEGORIES[e.category].emoji} {e.title}{(e as { pending?: boolean }).pending && <span className="ml-1 text-xs font-normal text-muted">⏳ esperando internet</span>}</span>
                        <span className="block text-sm text-muted">
                          {name(e.paid_by)} pagou{e.payer_share === 100 ? ' · sem dividir' : e.payer_share !== 50 ? ` · ${e.payer_share}% de quem pagou` : ''}
                        </span>
                      </span>
                      <span className="shrink-0 text-right">
                        <span className="board block text-lg text-ink">{money(e.amount_cents, e.currency)}</span>
                        {e.currency !== 'BRL' && <span className="block text-sm text-muted">{e.brl_cents != null ? money(e.brl_cents) : 'sem cotação'}</span>}
                      </span>
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </>
      )}
    </div>
  )
}
