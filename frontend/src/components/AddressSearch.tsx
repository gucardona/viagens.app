import { useEffect, useRef, useState } from 'react'
import { api } from '../api'

type GeoResult = { label: string; lat: number; lng: number }

// AddressSearch suggests places while typing (after a short pause) near `near`, the trip's region.
export default function AddressSearch({ value, near, onChange, onPick }: {
  value: string
  near?: { lat: number; lng: number }
  onChange: (v: string) => void
  onPick: (r: GeoResult) => void
}) {
  const [results, setResults] = useState<GeoResult[]>([])
  const [status, setStatus] = useState('')
  const typed = useRef(false)

  useEffect(() => {
    const q = value.trim()
    if (!typed.current || q.length < 3) return setResults([])
    const ctl = new AbortController()
    const t = setTimeout(async () => {
      setStatus('Buscando…')
      const bias = near ? `&lat=${near.lat.toFixed(3)}&lng=${near.lng.toFixed(3)}` : ''
      try {
        const r = await api<GeoResult[]>(`/api/geocode?q=${encodeURIComponent(q)}${bias}`, { signal: ctl.signal })
        setResults(r)
        setStatus(r.length ? '' : 'Nenhum lugar encontrado. Toque no mapa para marcar o ponto.')
      } catch (err) {
        if (!ctl.signal.aborted) setStatus(`${(err as Error).message}. Toque no mapa para marcar o ponto.`)
      }
    }, 350)
    return () => (clearTimeout(t), ctl.abort())
  }, [value, near?.lat, near?.lng])

  function pick(r: GeoResult) {
    typed.current = false
    setResults([])
    setStatus('')
    onPick(r)
  }

  return (
    <div>
      <input className="field" value={value} autoComplete="off" enterKeyHint="search" placeholder="Nome do lugar ou endereço"
        onChange={(e) => { typed.current = true; onChange(e.target.value) }}
        onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); if (results[0]) pick(results[0]) } }} />
      {status && <p role="status" className="mt-1 text-sm text-muted">{status}</p>}
      {results.length > 0 && (
        <ul className="mt-1 divide-y divide-line overflow-hidden rounded-lg border border-line bg-paper shadow-sm">
          {results.map((r) => (
            <li key={`${r.lat},${r.lng}`}>
              <button type="button" className="w-full px-3 py-3 text-left text-sm" onClick={() => pick(r)}>{r.label}</button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
