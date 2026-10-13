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
    </div>
  )
}
