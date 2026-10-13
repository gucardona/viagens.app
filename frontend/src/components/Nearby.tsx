import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { api, json } from '../api'
import { distanceKm } from '../cluster'
import type { Extra } from './TripMap'
import type { TripDetail } from '../types'

type Result = { name: string; lat: number; lng: number; distance: number; kind: string; cuisine: string; opening_hours: string; fee: string; wheelchair: string }
type What = 'restaurant' | 'toilets'

const meters = (m: number) => (m < 1000 ? `${Math.round(m / 10) * 10} m` : `${(m / 1000).toFixed(1).replace('.', ',')} km`)
const walk = (r: { lat: number; lng: number }) => `https://www.google.com/maps/dir/?api=1&destination=${r.lat},${r.lng}&travelmode=walking`

// "Restaurantes perto" / "Banheiros perto" around a point: saved restaurants first, then OpenStreetMap.
export default function Nearby({ center, data, excludeId, onPins }: {
  center: { lat: number; lng: number } | null
  data: TripDetail
  excludeId?: number
  onPins?: (pins: Extra[]) => void
}) {
  const qc = useQueryClient()
  const [what, setWhat] = useState<What | null>(null)
  const [results, setResults] = useState<Result[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [savedNow, setSavedNow] = useState<Set<string>>(new Set())

  const saved = what === 'restaurant' && center
    ? data.items
        .filter((i) => i.kind === 'restaurant' && i.lat != null && i.id !== excludeId)
        .map((i) => ({ item: i, d: distanceKm(center, { lat: i.lat!, lng: i.lng! }) * 1000 }))
        .filter((x) => x.d <= 1000)
        .sort((a, b) => a.d - b.d)
    : []
  // Not repeating anything already in the trip (same name, less than 50 m away).
  const fresh = results.filter((r) => !data.items.some((i) => i.lat != null && i.title.toLowerCase() === r.name.toLowerCase() && distanceKm(r, { lat: i.lat, lng: i.lng! }) < 0.05))

  useEffect(() => {
    onPins?.(fresh.map((r) => ({ lat: r.lat, lng: r.lng, emoji: what === 'toilets' ? '🚻' : '🍽️', label: r.name })))
  }, [results, what])

  async function search(w: What) {
    if (!center) return
    setWhat(w)
    setError('')
    setResults([])
    setLoading(true)
    try {
      setResults(await api<Result[]>(`/api/nearby?lat=${center.lat.toFixed(5)}&lng=${center.lng.toFixed(5)}&what=${w}`))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  async function save(r: Result) {
    const notes = [r.cuisine && `Cozinha: ${r.cuisine}`, r.opening_hours && `Horário: ${r.opening_hours}`].filter(Boolean).join('\n')
    await api(`/api/trips/${data.trip.id}/import`, json('POST', [{ kind: 'restaurant', title: r.name, notes, lat: r.lat, lng: r.lng }]))
    setSavedNow(new Set(savedNow).add(r.name + r.lat))
    qc.invalidateQueries({ queryKey: ['trip', String(data.trip.id)] })
  }

  return (
    <div>
      <div className="flex gap-2">
        <button type="button" className={`btn btn-quiet flex-1 text-sm ${what === 'restaurant' ? 'border-ink!' : ''}`} disabled={!center || loading} onClick={() => search('restaurant')}>🍽️ Restaurantes perto</button>
        <button type="button" className={`btn btn-quiet flex-1 text-sm ${what === 'toilets' ? 'border-ink!' : ''}`} disabled={!center || loading} onClick={() => search('toilets')}>🚻 Banheiros perto</button>
      </div>
      {loading && <p className="mt-2 text-sm text-muted">Procurando…</p>}
      {error && <p role="alert" className="mt-2 text-sm text-red-700">{error}</p>}

      {saved.length > 0 && (
        <>
          <h3 className="mt-3 text-sm font-semibold text-muted">Que vocês salvaram</h3>
          <ul className="mt-1 space-y-1">
            {saved.map(({ item, d }) => (
              <li key={item.id}>
                <Link to={`/viagens/${data.trip.id}/itens/${item.id}`} className="flex justify-between gap-2 rounded-lg bg-paper p-3">
                  <span className="truncate font-semibold">⭐ {item.title}</span>
                  <span className="board shrink-0 text-ink">{meters(d)}</span>
                </Link>
              </li>
            ))}
          </ul>
        </>
      )}

      {what && !loading && !error && (
        <>
          {what === 'restaurant' && <h3 className="mt-3 text-sm font-semibold text-muted">Novos por perto</h3>}
          {fresh.length === 0 && <p className="mt-2 text-sm text-muted">Nada encontrado a até 800 m.</p>}
          <ul className="mt-1 space-y-1">
            {fresh.map((r) => (
              <li key={r.name + r.lat} className="flex items-start justify-between gap-2 rounded-lg bg-paper p-3">
                <span className="min-w-0">
                  <span className="block truncate font-semibold">{r.name}</span>
                  <span className="block text-sm text-muted">
                    {[r.cuisine, r.opening_hours, r.fee === 'yes' && 'pago', r.fee === 'no' && 'grátis', r.wheelchair === 'yes' && '♿'].filter(Boolean).join(' · ')}
                  </span>
                </span>
                <span className="flex shrink-0 flex-col items-end gap-1">
                  <span className="board text-ink">{meters(r.distance)}</span>
                  {what === 'restaurant' ? (
                    savedNow.has(r.name + r.lat)
                      ? <span className="text-sm text-green-700">Salvo</span>
                      : <button type="button" className="text-sm font-semibold text-ink underline" onClick={() => save(r)}>Salvar</button>
                  ) : (
                    <a className="text-sm font-semibold text-ink underline" href={walk(r)} target="_blank" rel="noreferrer">Como chegar</a>
                  )}
                </span>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  )
}
