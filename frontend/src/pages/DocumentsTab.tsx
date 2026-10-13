import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api } from '../api'
import type { Attachment, TripDetail } from '../types'

function FileRow({ a, onDelete }: { a: Attachment; onDelete: (a: Attachment) => void }) {
  return (
    <li className="flex items-center justify-between gap-2 rounded-lg bg-paper p-3">
      <a href={`/api/attachments/${a.id}`} target="_blank" rel="noreferrer" className="min-w-0 truncate text-ink underline">{a.filename}</a>
      <button className="btn-danger shrink-0 text-sm" onClick={() => onDelete(a)}>Apagar</button>
    </li>
  )
}

export default function DocumentsTab({ data }: { data: TripDetail }) {
  const qc = useQueryClient()
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const refresh = () => qc.invalidateQueries({ queryKey: ['trip', String(data.trip.id)] })
  const general = data.attachments.filter((a) => a.item_id === null)
  const byItem = data.items
    .map((i) => ({ item: i, files: data.attachments.filter((a) => a.item_id === i.id) }))
    .filter((g) => g.files.length)

  async function add(file: File) {
    const body = new FormData()
    body.append('file', file)
    setUploading(true)
    setError('')
    try {
      await api(`/api/trips/${data.trip.id}/attachments`, { method: 'POST', body })
      await refresh()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setUploading(false)
    }
  }

  async function remove(a: Attachment) {
    if (!confirm(`Apagar ${a.filename}?`)) return
    await api(`/api/attachments/${a.id}`, { method: 'DELETE' })
    await refresh()
  }

  return (
    <div className="mt-4 space-y-6">
      <section>
        <h2 className="board mb-2 text-2xl font-bold text-ink">Gerais</h2>
        {general.length === 0 && <p className="mb-2 text-sm text-muted">Seguro, passaportes, vistos: o que vale para a viagem toda.</p>}
        <ul className="space-y-1">{general.map((a) => <FileRow key={a.id} a={a} onDelete={remove} />)}</ul>
        <label className="btn btn-quiet mt-2 inline-block cursor-pointer">
          {uploading ? 'Enviando…' : 'Anexar documento'}
          <input type="file" accept="image/*,application/pdf" className="sr-only" disabled={uploading}
            onChange={(e) => { const f = e.target.files?.[0]; if (f) add(f); e.target.value = '' }} />
        </label>
        {error && <p role="alert" className="mt-2 text-sm text-red-700">{error}</p>}
      </section>
      {byItem.map(({ item, files }) => (
        <section key={item.id}>
          <h2 className="board mb-2 text-2xl font-bold text-ink">{item.title}</h2>
          <ul className="space-y-1">{files.map((a) => <FileRow key={a.id} a={a} onDelete={remove} />)}</ul>
        </section>
      ))}
    </div>
  )
}
