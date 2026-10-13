import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, json } from '../api'
import { parseTakeout, type Place } from '../takeout'

type FileResult = { name: string; places: Place[]; error?: string }

// Import Google Takeout files (starred places GeoJSON, saved list CSVs) as saved places of a trip.
export default function ImportDialog({ tripId, onClose }: { tripId: number; onClose: () => void }) {
  const qc = useQueryClient()
  const [files, setFiles] = useState<FileResult[]>([])
  const [busy, setBusy] = useState(false)
  const [done, setDone] = useState<{ created: number; skipped: number } | null>(null)
  const [error, setError] = useState('')
  const all = files.flatMap((f) => f.places)

  async function read(list: FileList) {
    setDone(null)
    setFiles(await Promise.all([...list].map(async (f) => {
      try {
        return { name: f.name, places: parseTakeout(f.name, await f.text()) }
      } catch {
        return { name: f.name, places: [], error: 'arquivo não reconhecido' }
      }
    })))
  }

  async function send() {
    setBusy(true)
    setError('')
    const total = { created: 0, skipped: 0 }
    try {
      for (let i = 0; i < all.length; i += 500) {
        const r = await api<typeof total>(`/api/trips/${tripId}/import`, json('POST', all.slice(i, i + 500)))
        total.created += r.created
        total.skipped += r.skipped
      }
      setDone(total)
      setFiles([])
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
      qc.invalidateQueries({ queryKey: ['trip', String(tripId)] })
    }
  }

  return (
    <div className="mt-3 rounded-lg border border-line bg-paper p-4">
      <div className="flex items-start justify-between gap-2">
        <h3 className="font-semibold">Importar do Google</h3>
        <button type="button" onClick={onClose} aria-label="Fechar" className="text-muted">✕</button>
      </div>
      <p className="mt-1 text-sm text-muted">
        Em takeout.google.com, exporte "Salvos" e "Maps (seus lugares)", descompacte e escolha aqui os arquivos <b>.csv</b> das listas e o <b>Saved Places.json</b>.
      </p>
      <label className="btn btn-quiet mt-3 inline-block cursor-pointer">
        Escolher arquivos
        <input type="file" multiple accept=".csv,.json,.geojson" className="sr-only" onChange={(e) => e.target.files && read(e.target.files)} />
      </label>
      {files.length > 0 && (
        <ul className="mt-3 space-y-1 text-sm">
          {files.map((f) => (
            <li key={f.name} className="flex justify-between gap-2">
              <span className="truncate">{f.name}</span>
              <span className="shrink-0 text-muted">{f.error ?? `${f.places.length} lugares`}</span>
            </li>
          ))}
        </ul>
      )}
      {all.length > 0 && (
        <button type="button" className="btn mt-3 w-full" disabled={busy} onClick={send}>
          {busy ? 'Importando…' : `Importar ${all.length} lugares`}
        </button>
      )}
      {error && <p role="alert" className="mt-2 text-sm text-red-700">{error}</p>}
      {done && (
        <p className="mt-3 text-sm">
          {done.created} {done.created === 1 ? 'lugar novo' : 'lugares novos'}
          {done.skipped > 0 && `, ${done.skipped} já estavam salvos`}. Os que vieram sem local são procurados pelo nome em segundo plano.
        </p>
      )}
    </div>
  )
}
