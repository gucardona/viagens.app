import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from '../api'
import { buildDays } from '../days'
import { flag } from '../geo'
import type { Item, Trip } from '../types'
import { Day } from './Trip'

// Read-only itinerary for the family, opened from the secret link without logging in.
export default function Shared() {
  const { token = '' } = useParams()
  const { data, error } = useQuery({ queryKey: ['shared', token], queryFn: () => api<{ trip: Trip; items: Item[] }>(`/api/shared/${token}`) })
  const days = useMemo(() => (data ? buildDays(data.trip, data.items).filter((d) => d.date) : []), [data])
  const flags = useMemo(() => [...new Set(data?.items.map((i) => i.country_code).filter(Boolean))].sort().map(flag).join(' '), [data])

  if (error) return <p role="alert" className="p-6 text-red-700">{error.message}</p>
  if (!data) return null

  return (
    <div className="mx-auto max-w-2xl px-4 pb-16 pt-[max(1rem,env(safe-area-inset-top))]">
      <header className="mb-2">
        <h1 className="board text-4xl font-bold leading-tight text-ink">{data.trip.name}</h1>
        {(data.trip.destination || flags) && <p className="text-muted">{data.trip.destination} {flags && <span aria-label="Países">{flags}</span>}</p>}
        <p className="mt-2 text-sm text-muted">Roteiro compartilhado, só para ver. Abra de novo para ver as mudanças.</p>
      </header>
      <div className="divide-y divide-line">
        {days.map((d) => <Day key={d.date ?? 'none'} day={d} />)}
      </div>
    </div>
  )
}
