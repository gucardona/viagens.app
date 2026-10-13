import { useCallback, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { buildDays, pinNumbers, timeLabel } from '../days'
import { KINDS } from '../kinds'
import TripMap from '../components/TripMap'
import type { Item, TripDetail } from '../types'

export default function TripMapTab({ data }: { data: TripDetail }) {
  const days = useMemo(() => buildDays(data.trip, data.items).filter((d) => d.date), [data])
  const [params, setParams] = useSearchParams()
  const day = params.get('dia')
  const setDay = (d: string | null) => setParams(d ? { aba: 'mapa', dia: d } : { aba: 'mapa' }, { replace: true })
  const [selected, setSelected] = useState<Item | null>(null)
  const items = useMemo(() => {
    const list = day ? days.find((d) => d.date === day)?.items ?? [] : data.items
    return list.filter((i) => i.lat != null)
  }, [day, days, data.items])
  const numbers = useMemo(() => (day ? pinNumbers(items) : undefined), [day, items])
  const onSelect = useCallback((i: Item) => setSelected(i), [])
  const hasPins = data.items.some((i) => i.lat != null)

  return (
    <div className="mt-2">
      <div className="-mx-4 mb-3 flex gap-2 overflow-x-auto px-4 pb-1">
        <button onClick={() => setDay(null)} aria-pressed={day === null}
          className={`board shrink-0 rounded-full border px-3 py-1 text-lg ${day === null ? 'border-ink bg-ink text-white' : 'border-line bg-paper'}`}>Todos</button>
        {days.map((d) => (
          <button key={d.date} onClick={() => setDay(d.date)} aria-pressed={day === d.date}
            className={`board shrink-0 rounded-full border px-3 py-1 text-lg ${day === d.date ? 'border-ink bg-ink text-white' : 'border-line bg-paper'}`}>
            {d.date!.slice(8, 10)}/{d.date!.slice(5, 7)}
          </button>
        ))}
      </div>
      {!hasPins && <p className="mb-2 text-sm text-muted">Nenhum item com local ainda. Abra um item e toque no mapa para marcar onde ele fica.</p>}
      <TripMap items={items} line={day !== null} numbers={numbers} onSelect={onSelect} geolocate className="h-[60dvh]" />
      {hasPins && day && items.length === 0 && <p className="mt-2 text-sm text-muted">Nenhum item com local neste dia.</p>}
      {selected && (
        <Link to={`/viagens/${data.trip.id}/itens/${selected.id}`} className="mt-3 flex gap-3 rounded-lg bg-paper p-3" style={{ borderLeft: `4px solid ${KINDS[selected.kind].color}` }}>
          <span className="board shrink-0 text-lg text-ink">{timeLabel(selected, day ?? selected.start_date) || '—'}</span>
          <span className="min-w-0">
            <span className="block font-semibold">{KINDS[selected.kind].emoji} {selected.title}</span>
            {selected.address && <span className="block truncate text-sm text-muted">{selected.address}</span>}
          </span>
        </Link>
      )}
    </div>
  )
}
