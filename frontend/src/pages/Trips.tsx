import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api, shortDate } from '../api'
import type { Trip } from '../types'

export default function Trips() {
  const { data: trips, error } = useQuery({ queryKey: ['trips'], queryFn: () => api<Trip[]>('/api/trips') })

  return (
    <main className="mx-auto max-w-2xl px-4 pb-24 pt-6">
      <header className="mb-6 flex items-center justify-between">
        <h1 className="board text-4xl font-bold text-ink">Viagens</h1>
        <Link to="/viagens/nova" className="btn">Nova viagem</Link>
      </header>
      {error && <p role="alert" className="text-red-700">{error.message}</p>}
      {trips?.length === 0 && (
        <p className="mt-16 text-center text-muted">Nenhuma viagem ainda. Crie a primeira para começar o roteiro.</p>
      )}
      <ul className="divide-y divide-line border-y border-line">
        {trips?.map((t) => (
          <li key={t.id}>
            <Link to={`/viagens/${t.id}`} className="flex items-baseline gap-4 py-4">
              <span className="board w-24 shrink-0 text-lg text-ink">{shortDate(t.start_date)}–{shortDate(t.end_date)}</span>
              <span className="min-w-0">
                <span className="block truncate text-lg font-semibold">{t.name}</span>
                {t.destination && <span className="block truncate text-sm text-muted">{t.destination}</span>}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </main>
  )
}
