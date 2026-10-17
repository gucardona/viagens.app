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

// Banner when offline or with changes waiting; the download control lives in Mais (OfflineSave).
export default function OfflineBar({ data }: { data: TripDetail }) {
  const online = useOnline()
  const info = offlineInfo(data.trip.id)

  // A trip already saved refreshes its copy whenever it is opened online.
  useEffect(() => {
    if (online && offlineInfo(data.trip.id) && 'caches' in window) {
      saveTripOffline(data.trip.id, data.attachments).catch(() => {})
    }
    // Keyed on the trip and its documents, not on every refetch (live updates refetch often).
  }, [online, data.trip.id, data.attachments.length])

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
      <div role="status" className="mb-3 rounded-lg bg-signal/40 p-3 text-sm text-ink">
        {outbox}
        <p>Sem internet{info ? `: mostrando a cópia de ${when(info.at)}` : ''}. Itens, gastos e checklist salvos agora vão quando a internet voltar.</p>
      </div>
    )
  }
  return waiting > 0 || refused.length > 0 ? <div className="mb-3 text-sm">{outbox}</div> : null
}

// The "Baixar para usar offline" control and what is already saved.
export function OfflineSave({ data }: { data: TripDetail }) {
  const [info, setInfo] = useState<OfflineInfo | null>(() => offlineInfo(data.trip.id))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

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

  if (!('caches' in window)) return <p className="text-sm text-muted">Este navegador não guarda a viagem para usar sem internet.</p>
  return (
    <div className="space-y-3">
      <p className="text-sm text-muted">Guarda roteiro, mapas e documentos no celular para abrir sem internet. A cópia se atualiza sozinha quando você abre a viagem online.</p>
      {info && <p className="text-sm">✓ Disponível offline: {info.files} documentos, {mb(info.bytes)}, atualizado em {when(info.at)}</p>}
      <button type="button" className="btn" disabled={busy} onClick={save}>{busy ? 'Baixando…' : info ? 'Baixar de novo' : 'Baixar para usar offline'}</button>
      {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
    </div>
  )
}
