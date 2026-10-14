import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, json } from '../api'
import { KINDS } from '../kinds'
import { mapLinks } from '../links'
import TripMap, { type Extra } from '../components/TripMap'
import Nearby from '../components/Nearby'
import AddressSearch from '../components/AddressSearch'
import { BackBar, SaveCancel, useBack } from '../components/FormBar'
import { useExpenses } from './ExpensesTab'
import { money } from '../money'
import type { Attachment, Item, Kind, TripDetail } from '../types'

type Form = Omit<Item, 'id' | 'trip_id' | 'created_at' | 'updated_at' | 'country_code' | 'area'>

const empty: Form = {
  kind: 'activity', title: '', start_date: '', start_time: '', end_date: '', end_time: '',
  address: '', lat: null, lng: null, confirmation_code: '', notes: '', ticket: '', time_approx: false,
}

const TICKETS: { value: Item['ticket']; label: string }[] = [
  { value: '', label: 'Não precisa' },
  { value: 'needed', label: 'A comprar' },
  { value: 'bought', label: 'Comprado' },
]

export default function ItemSheet() {
  const { id = '', itemId = 'novo' } = useParams()
  const isNew = itemId === 'novo'
  const nav = useNavigate()
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['trip', id], queryFn: () => api<TripDetail>(`/api/trips/${id}`) })
  const item = data?.items.find((i) => String(i.id) === itemId)
  const files = data?.attachments.filter((a) => String(a.item_id) === itemId) ?? []

  const [form, setForm] = useState<Form>(empty)
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [extra, setExtra] = useState<Extra[]>([])
  const [params] = useSearchParams()
  const { data: expenses } = useExpenses(id)
  const spent = expenses?.expenses.filter((e) => String(e.item_id) === itemId) ?? []

  // Fill the form once per item; later refetches (upload, focus) must not overwrite unsaved edits.
  const loadedFor = useRef<string>()
  useEffect(() => {
    if (loadedFor.current === itemId || !data) return
    if (item) {
      const { id: _i, trip_id: _t, created_at: _c, updated_at: _u, country_code: _cc, area: _a, ...rest } = item
      setForm(rest)
    } else if (isNew) setForm({ ...empty, start_date: params.has('salvo') ? '' : data.trip.start_date })
    else return
    loadedFor.current = itemId
  }, [item, isNew, data, itemId])

  const set = (k: keyof Form) => (e: { target: { value: string } }) => setForm({ ...form, [k]: e.target.value })
  const { lat, lng, kind } = form
  const pinItems = useMemo(() => (lat != null ? [{ ...empty, kind, lat, lng, id: 0, trip_id: 0 } as Item] : []), [lat, lng, kind])
  const links = mapLinks(form)
  const refresh = () => qc.invalidateQueries({ queryKey: ['trip', id] })
  const back = useBack(`/viagens/${id}`)
  const near = useMemo(() => {
    const pins = data?.items.filter((i) => i.lat != null) ?? []
    if (!pins.length) return undefined
    return { lat: pins.reduce((s, i) => s + i.lat!, 0) / pins.length, lng: pins.reduce((s, i) => s + i.lng!, 0) / pins.length }
  }, [data])

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const saved = await api<Item>(isNew ? `/api/trips/${id}/items` : `/api/items/${itemId}`, json(isNew ? 'POST' : 'PUT', form))
      await refresh()
      if (isNew) nav(`/viagens/${id}/itens/${saved.id}`, { replace: true })
      else back()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!confirm('Apagar este item e os documentos dele?')) return
    await api(`/api/items/${itemId}`, { method: 'DELETE' })
    await refresh()
    nav(`/viagens/${id}`, { replace: true })
  }

  async function addFile(file: File) {
    const body = new FormData()
    body.append('item_id', itemId)
    body.append('file', file)
    setUploading(true)
    try {
      await api<Attachment>(`/api/trips/${id}/attachments`, { method: 'POST', body })
      await refresh()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setUploading(false)
    }
  }

  async function removeFile(a: Attachment) {
    if (!confirm(`Apagar ${a.filename}?`)) return
    await api(`/api/attachments/${a.id}`, { method: 'DELETE' })
    await refresh()
  }

  if (!isNew && data && !item) return <p className="p-6">Item não encontrado.</p>

  return (
    <main className="mx-auto max-w-xl px-4 pb-24">
      <BackBar label={data?.trip.name ?? 'Voltar'} onBack={back} />
      <form onSubmit={submit} className="space-y-4">
        <div className="flex flex-wrap gap-2" role="radiogroup" aria-label="Tipo">
          {(Object.keys(KINDS) as Kind[]).map((k) => (
            <button
              type="button" key={k} role="radio" aria-checked={form.kind === k}
              onClick={() => setForm({ ...form, kind: k })}
              className={`rounded-full border px-3 py-1.5 text-sm ${form.kind === k ? 'border-ink bg-ink text-white' : 'border-line bg-paper'}`}
            >
              {KINDS[k].emoji} {KINDS[k].label}
            </button>
          ))}
        </div>
        <label className="block">
          <span className="mb-1 block text-sm text-muted">Título</span>
          <input className="field" required maxLength={200} value={form.title} onChange={set('title')} placeholder={form.kind === 'flight' ? 'TAP 1234 Lisboa → Porto' : ''} />
        </label>
        <div className="grid grid-cols-2 gap-3">
          <label className="block"><span className="mb-1 block text-sm text-muted">{form.kind === 'lodging' ? 'Check-in' : 'Data'}</span>
            <input className="field" type="date" value={form.start_date} onChange={set('start_date')} /></label>
          <label className="block"><span className="mb-1 block text-sm text-muted">Hora</span>
            <input className="field" type="time" value={form.start_time} onChange={set('start_time')} /></label>
          <label className="block"><span className="mb-1 block text-sm text-muted">{form.kind === 'lodging' ? 'Check-out' : 'Até (data)'}</span>
            <input className="field" type="date" min={form.start_date} value={form.end_date} onChange={set('end_date')} /></label>
          <label className="block"><span className="mb-1 block text-sm text-muted">Hora</span>
            <input className="field" type="time" value={form.end_time} onChange={set('end_time')} /></label>
        </div>
        <label className="flex items-center gap-2">
          <input type="checkbox" className="h-5 w-5" checked={form.time_approx} onChange={(e) => setForm({ ...form, time_approx: e.target.checked })} />
          <span>Horário aproximado</span>
        </label>

        <div>
          <span className="mb-1 block text-sm text-muted">Ingresso</span>
          <div className="flex gap-2" role="radiogroup" aria-label="Ingresso">
            {TICKETS.map((t) => (
              <button type="button" key={t.value} role="radio" aria-checked={form.ticket === t.value}
                onClick={() => setForm({ ...form, ticket: t.value })}
                className={`flex-1 rounded-lg border px-2 py-2 text-sm ${form.ticket === t.value ? 'border-ink bg-ink text-white' : 'border-line bg-paper'}`}>
                {t.label}
              </button>
            ))}
          </div>
        </div>

        <div>
          <span className="mb-1 block text-sm text-muted">Endereço</span>
          <AddressSearch value={form.address} near={near} onChange={(address) => setForm((f) => ({ ...f, address }))}
            onPick={(r) => setForm((f) => ({ ...f, address: r.label, lat: r.lat, lng: r.lng }))} />
          <p className="mt-2 text-sm text-muted">Toque no mapa para colocar ou mover o pin.</p>
          <TripMap items={pinItems} extra={extra} onPick={(p) => setForm((f) => ({ ...f, ...p }))} className="mt-1 h-56" />
          <div className="mt-2 flex flex-wrap gap-2 text-sm">
            {links && <a className="btn btn-quiet" href={links.google} target="_blank" rel="noreferrer">Abrir no Google Maps</a>}
            {links && <a className="btn btn-quiet" href={links.waze} target="_blank" rel="noreferrer">Abrir no Waze</a>}
            {form.lat != null && <button type="button" className="btn btn-danger" onClick={() => setForm({ ...form, lat: null, lng: null })}>Tirar pin</button>}
          </div>
        </div>

        {!isNew && data && item?.lat != null && (
          <Nearby center={{ lat: item.lat, lng: item.lng! }} data={data} excludeId={item.id} onPins={setExtra} />
        )}

        <label className="block"><span className="mb-1 block text-sm text-muted">Código da reserva</span>
          <input className="field board text-lg tracking-wider" value={form.confirmation_code} onChange={set('confirmation_code')} /></label>
        <label className="block"><span className="mb-1 block text-sm text-muted">Notas</span>
          <textarea className="field" rows={3} value={form.notes} onChange={set('notes')} /></label>

        {!isNew && (
          <div>
            <span className="mb-1 block text-sm text-muted">Documentos</span>
            <ul className="space-y-1">
              {files.map((a) => (
                <li key={a.id} className="flex items-center justify-between gap-2 rounded-lg bg-paper p-3">
                  <a href={`/api/attachments/${a.id}`} target="_blank" rel="noreferrer" className="min-w-0 truncate text-ink underline">{a.filename}</a>
                  <button type="button" className="btn-danger shrink-0 text-sm" onClick={() => removeFile(a)}>Apagar</button>
                </li>
              ))}
            </ul>
            <label className="btn btn-quiet mt-2 inline-block cursor-pointer">
              {uploading ? 'Enviando…' : 'Anexar foto ou PDF'}
              <input type="file" accept="image/*,application/pdf" className="sr-only" disabled={uploading}
                onChange={(e) => { const f = e.target.files?.[0]; if (f) addFile(f); e.target.value = '' }} />
            </label>
          </div>
        )}

        {!isNew && (
          <div>
            <span className="mb-1 block text-sm text-muted">Gastos</span>
            {spent.map((e) => (
              <Link key={e.id} to={`/viagens/${id}/gastos/${e.id}`} className="mb-1 flex justify-between rounded-lg bg-paper p-3">
                <span className="truncate">{e.title}</span><span className="board text-lg text-ink">{money(e.amount_cents, e.currency)}</span>
              </Link>
            ))}
            <Link to={`/viagens/${id}/gastos/novo?item=${itemId}`} className="btn btn-quiet inline-block">+ Adicionar gasto</Link>
          </div>
        )}

        {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
        <SaveCancel save={isNew ? 'Criar item' : 'Salvar'} onCancel={back} />
        {isNew && <p className="text-center text-sm text-muted">Depois de criar, você pode anexar documentos.</p>}
        {!isNew && <button type="button" onClick={remove} className="btn btn-danger w-full">Apagar item</button>}
      </form>
    </main>
  )
}
