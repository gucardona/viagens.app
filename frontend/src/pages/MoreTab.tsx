import Checklist from '../components/Checklist'
import DocumentsTab from './DocumentsTab'
import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { OfflineSave } from '../components/OfflineBar'
import { offlineInfo } from '../offline'
import { api } from '../api'
import type { CheckItem, Trip, TripDetail } from '../types'

function ShareLink({ trip }: { trip: Trip }) {
  const qc = useQueryClient()
  const [msg, setMsg] = useState('')
  const url = trip.share_token ? `${location.origin}/compartilhado/${trip.share_token}` : ''
  const call = (method: string) => {
    setMsg('')
    api(`/api/trips/${trip.id}/share`, { method }).then(() => qc.invalidateQueries({ queryKey: ['trip'] }), (e: Error) => setMsg(e.message))
  }
  const send = () => {
    if (navigator.share) navigator.share({ title: trip.name, url }).catch(() => {})
    else navigator.clipboard.writeText(url).then(() => setMsg('Link copiado.'), () => setMsg('Não deu para copiar; selecione o link acima.'))
  }
  return (
    <section>
      <p className="mb-3 text-sm text-muted">
        Quem tiver o link vê o roteiro e os mapas, sem login e sem poder mudar nada. Códigos de reserva, observações, documentos e gastos ficam de fora.
      </p>
      {url ? (
        <>
          <input className="field w-full" readOnly value={url} onFocus={(e) => e.target.select()} aria-label="Link da viagem" />
          <div className="mt-2 flex flex-wrap gap-2">
            <button className="btn" onClick={send}>Enviar link</button>
            <button className="btn btn-quiet" onClick={() => confirm('Desativar o link? Quem tem o link antigo deixa de ver o roteiro.') && call('DELETE')}>Desativar link</button>
          </div>
        </>
      ) : (
        <button className="btn btn-quiet" onClick={() => call('POST')}>Criar link</button>
      )}
      {msg && <p role="status" className="mt-2 text-sm text-muted">{msg}</p>}
    </section>
  )
}

function Calendar({ trip }: { trip: Trip }) {
  return (
    <section>
      <p className="mb-3 text-sm text-muted">
        Leva o roteiro para o calendário do celular. No iPhone, abra o arquivo e toque em “Adicionar tudo”; no Google Agenda, use Configurações › Importar.
      </p>
      <a className="btn inline-block" href={`/api/trips/${trip.id}/calendar.ics`} download>Baixar para o calendário</a>
    </section>
  )
}

const SECTIONS = [
  { key: 'checklist', emoji: '✅', label: 'Checklist' },
  { key: 'documentos', emoji: '📄', label: 'Documentos' },
  { key: 'compartilhar', emoji: '🔗', label: 'Compartilhar viagem' },
  { key: 'calendario', emoji: '📅', label: 'Calendário' },
  { key: 'offline', emoji: '📶', label: 'Usar sem internet' },
] as const

// An index of rows, after gastos.app's Ajustes; each row opens its section (?secao=).
export default function MoreTab({ data }: { data: TripDetail }) {
  const [params, setParams] = useSearchParams()
  const secao = params.get('secao') ?? (params.get('aba') === 'documentos' ? 'documentos' : null)
  const { data: list } = useQuery({ queryKey: ['checklist', String(data.trip.id)], queryFn: () => api<CheckItem[]>(`/api/trips/${data.trip.id}/checklist`) })
  const open = SECTIONS.find((s) => s.key === secao)

  if (open) {
    return (
      <div className="mt-2">
        <button onClick={() => setParams({ aba: 'mais' }, { replace: true })} className="-ml-1 inline-flex min-h-11 items-center gap-1 pr-3 font-semibold text-ink">
          <span aria-hidden className="text-2xl leading-none">‹</span> Mais
        </button>
        <h2 className="board mb-3 text-3xl font-bold text-ink">{open.label}</h2>
        {open.key === 'checklist' && <Checklist tripId={data.trip.id} />}
        {open.key === 'documentos' && <DocumentsTab data={data} />}
        {open.key === 'compartilhar' && <ShareLink trip={data.trip} />}
        {open.key === 'calendario' && <Calendar trip={data.trip} />}
        {open.key === 'offline' && <OfflineSave data={data} />}
      </div>
    )
  }

  const value: Record<string, string> = {
    checklist: list?.length ? `${list.filter((c) => c.done).length}/${list.length}` : '',
    documentos: data.attachments.length ? String(data.attachments.length) : '',
    compartilhar: data.trip.share_token ? 'Ligado' : '',
    offline: offlineInfo(data.trip.id) ? 'Salvo' : '',
  }
  return (
    <ul className="mt-2 divide-y divide-line overflow-hidden rounded-lg bg-paper">
      {SECTIONS.map((s) => (
        <li key={s.key}>
          <button onClick={() => setParams({ aba: 'mais', secao: s.key }, { replace: true })} className="flex min-h-14 w-full items-center gap-3 px-4 text-left">
            <span aria-hidden className="grid h-9 w-9 place-items-center rounded-[0.7rem] bg-sea text-lg">{s.emoji}</span>
            <span className="flex-1 font-medium text-ink">{s.label}</span>
            {value[s.key] && <span className="text-muted">{value[s.key]}</span>}
            <span aria-hidden className="text-xl text-muted">›</span>
          </button>
        </li>
      ))}
    </ul>
  )
}
