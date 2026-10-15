import { useEffect, useState, useSyncExternalStore } from 'react'
import { offlineInfo, saveTripOffline, type OfflineInfo } from '../offline'
import type { TripDetail } from '../types'
import { clearErrors, outboxErrors, pending, subscribe } from '../outbox'

const when = (iso: string) => new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
const mb = (b: number) => `${(b / 1048576).toFixed(1).replace('.', ',')} MB`

export function useOnline() {
  const [online, setOnline] = useState(navigator.onLine)
  useEffect(() => {
    const on = () => setOnline(true)
    const off = () => setOnline(false)
    addEventListener('online', on)
    addEventListener('offline', off)
    return () => (removeEventListener('online', on), removeEventListener('offline', off))
  }, [])
  return online
}

// Banner when offline, and the "Baixar para usar offline" button.
export default function OfflineBar({ data }: { data: TripDetail }) {
  const online = useOnline()
  const [info, setInfo] = useState<OfflineInfo | null>(() => offlineInfo(data.trip.id))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  // A trip already saved refreshes its copy whenever it is opened online.
  useEffect(() => {
    if (online && offlineInfo(data.trip.id) && 'caches' in window) {
      saveTripOffline(data.trip.id, data.attachments).then(setInfo).catch(() => {})
    }
  }, [online, data])

  async function save() {
    setBusy(true)
    setError('')
    try {
      setInfo(await saveTripOffline(data.trip.id, data.attachments))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const waiting = useSyncExternalStore(subscribe, () => pending().length)
  const refused = useSyncExternalStore(subscribe, outboxErrors)
  const outbox = (
    <>
      {waiting > 0 && <p role="status" className="mb-1 font-semibold text-ink">⏳ {waiting === 1 ? '1 alteração esperando' : `${waiting} alterações esperando`} internet</p>}
      {refused.length > 0 && (
        <p role="alert" className="mb-1 text-red-700">
          {refused.length === 1 ? '1 alteração não pôde ser salva' : `${refused.length} alterações não puderam ser salvas`}: {refused.join('; ')}.{' '}
          <button type="button" className="underline" onClick={clearErrors}>OK</button>
        </p>
      )}
    </>
  )

  if (!online) {
    return (
      <div role="status" className="mb-3 rounded-lg bg-signal/40 p-2 text-sm text-ink">
        {outbox}
        <p>Sem internet{info ? `: mostrando a cópia de ${when(info.at)}` : ''}. Itens, gastos e checklist salvos agora vão quando a internet voltar.</p>
      </div>
    )
  }
  if (!('caches' in window)) return <div className="mb-3 text-sm">{outbox}</div>
  return (
    <div className="mb-3 text-sm">
      {outbox}
      {info ? (
        <p className="text-muted">✓ Disponível offline · {info.files} documentos · {mb(info.bytes)} · {when(info.at)}</p>
      ) : (
        <button type="button" className="btn btn-quiet py-1.5! text-sm" disabled={busy} onClick={save}>{busy ? 'Baixando…' : '⬇ Baixar para usar offline'}</button>
      )}
      {error && <p role="alert" className="text-red-700">{error}</p>}
    </div>
  )
}
