import Checklist from '../components/Checklist'
import DocumentsTab from './DocumentsTab'
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api } from '../api'
import type { Trip, TripDetail } from '../types'

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
      <h2 className="board text-3xl font-bold text-ink">Compartilhar viagem</h2>
      <p className="my-3 text-sm text-muted">
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

export default function MoreTab({ data }: { data: TripDetail }) {
  return (
    <div className="space-y-8">
      <div className="mt-4"><Checklist tripId={data.trip.id} /></div>
      <section>
        <h2 className="board text-3xl font-bold text-ink">Documentos</h2>
        <DocumentsTab data={data} />
      </section>
      <ShareLink trip={data.trip} />
      <section>
        <h2 className="board text-3xl font-bold text-ink">Calendário</h2>
        <p className="my-3 text-sm text-muted">
          Leva o roteiro para o calendário do celular. No iPhone, abra o arquivo e toque em “Adicionar tudo”; no Google Agenda, use Configurações › Importar.
        </p>
        <a className="btn btn-quiet inline-block" href={`/api/trips/${data.trip.id}/calendar.ics`} download>
          Baixar para o calendário
        </a>
      </section>
    </div>
  )
}
