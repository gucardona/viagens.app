import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, json } from '../api'
import type { Trip, TripDetail } from '../types'

const empty = { name: '', destination: '', start_date: '', end_date: '', notes: '' }

export default function TripForm() {
  const { id } = useParams()
  const nav = useNavigate()
  const qc = useQueryClient()
  const [form, setForm] = useState(empty)
  const [error, setError] = useState('')
  const { data } = useQuery({ queryKey: ['trip', id], queryFn: () => api<TripDetail>(`/api/trips/${id}`), enabled: !!id })

  useEffect(() => {
    if (data) {
      const { name, destination, start_date, end_date, notes } = data.trip
      setForm({ name, destination, start_date, end_date, notes })
    }
  }, [data])

  const set = (k: keyof typeof empty) => (e: { target: { value: string } }) => setForm({ ...form, [k]: e.target.value })

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const trip = await api<Trip>(id ? `/api/trips/${id}` : '/api/trips', json(id ? 'PUT' : 'POST', form))
      qc.invalidateQueries()
      nav(`/viagens/${trip.id}`, { replace: true })
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!confirm('Apagar esta viagem com todos os itens e documentos?')) return
    await api(`/api/trips/${id}`, { method: 'DELETE' })
    qc.invalidateQueries()
    nav('/', { replace: true })
  }

  return (
    <main className="mx-auto max-w-xl px-4 pb-24 pt-6">
      <button onClick={() => nav(-1)} className="mb-4 text-ink">Voltar</button>
      <h1 className="board mb-6 text-3xl font-bold text-ink">{id ? 'Editar viagem' : 'Nova viagem'}</h1>
      <form onSubmit={submit} className="space-y-4">
        <label className="block">
          <span className="mb-1 block text-sm text-muted">Nome</span>
          <input className="field" required value={form.name} onChange={set('name')} placeholder="Lisboa e Porto" />
        </label>
        <label className="block">
          <span className="mb-1 block text-sm text-muted">Destino</span>
          <input className="field" value={form.destination} onChange={set('destination')} placeholder="Portugal" />
        </label>
        <div className="grid grid-cols-2 gap-3">
          <label className="block">
            <span className="mb-1 block text-sm text-muted">Início</span>
            <input className="field" type="date" required value={form.start_date} onChange={set('start_date')} />
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-muted">Fim</span>
            <input className="field" type="date" required min={form.start_date} value={form.end_date} onChange={set('end_date')} />
          </label>
        </div>
        <label className="block">
          <span className="mb-1 block text-sm text-muted">Notas</span>
          <textarea className="field" rows={3} value={form.notes} onChange={set('notes')} />
        </label>
        {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
        <button className="btn w-full">{id ? 'Salvar viagem' : 'Criar viagem'}</button>
        {id && <button type="button" onClick={remove} className="btn btn-danger w-full">Apagar viagem</button>}
      </form>
    </main>
  )
}
