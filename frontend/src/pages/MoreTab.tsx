import Checklist from '../components/Checklist'
import DocumentsTab from './DocumentsTab'
import type { TripDetail } from '../types'

export default function MoreTab({ data }: { data: TripDetail }) {
  return (
    <div className="space-y-8">
      <div className="mt-4"><Checklist tripId={data.trip.id} /></div>
      <section>
        <h2 className="board text-3xl font-bold text-ink">Documentos</h2>
        <DocumentsTab data={data} />
      </section>
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
