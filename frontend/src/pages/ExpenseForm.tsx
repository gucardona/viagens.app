import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, json } from '../api'
import { CATEGORIES, categoryFor, parseMoney, type Category } from '../money'
import { useExpenses, useUsers } from './ExpensesTab'
import { BackBar, SaveCancel, useBack } from '../components/FormBar'
import { knownIds } from '../outbox'
import type { Expense, TripDetail } from '../types'

const CURRENCIES = ['BRL', 'EUR', 'USD', 'GBP', 'CHF', 'JPY', 'ARS', 'CLP', 'UYU', 'COP', 'PEN', 'MXN', 'CAD', 'AUD', 'CZK', 'HUF', 'PLN', 'DKK', 'SEK', 'NOK', 'TRY', 'CNY']
const SPLITS = [{ v: 50, label: 'Metade' }, { v: 100, label: 'Quem pagou' }]
const today = () => new Date().toLocaleDateString('sv')

export default function ExpenseForm() {
  const { id = '', expenseId = 'novo' } = useParams()
  const [params] = useSearchParams()
  const isNew = expenseId === 'novo'
  const nav = useNavigate()
  const qc = useQueryClient()
  const { data: trip } = useQuery({ queryKey: ['trip', id], queryFn: () => api<TripDetail>(`/api/trips/${id}`) })
  const { data: s } = useExpenses(id)
  const { data: users } = useUsers()
  const { data: me } = useQuery({ queryKey: ['me'], queryFn: () => api<{ id: number }>('/api/me') })
  const existing = s?.expenses.find((e) => String(e.id) === expenseId)

  const [title, setTitle] = useState('')
  const [amount, setAmount] = useState('')
  const [currency, setCurrency] = useState('BRL')
  const [category, setCategory] = useState<Category>('food')
  const [date, setDate] = useState('')
  const [paidBy, setPaidBy] = useState(0)
  const [share, setShare] = useState(50)
  const [rate, setRate] = useState('')
  const [itemId, setItemId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [otherShare, setOtherShare] = useState(false)
  const back = useBack(`/viagens/${id}?aba=gastos`)
  // An expense created offline was just sent: move to its real id.
  const synced = !existing && Number(expenseId) < 0 ? knownIds()[expenseId] : undefined
  useEffect(() => {
    if (synced != null) nav(`/viagens/${id}/gastos/${synced}`, { replace: true })
  }, [synced, id, nav])

  const loaded = useRef(false)
  useEffect(() => {
    if (loaded.current || !trip || !s || !me) return
    if (existing) {
      setTitle(existing.title)
      setAmount((existing.amount_cents / 100).toFixed(2).replace('.', ','))
      setCurrency(existing.currency)
      setCategory(existing.category)
      setDate(existing.date)
      setPaidBy(existing.paid_by)
      setShare(existing.payer_share)
      setOtherShare(existing.payer_share !== 50 && existing.payer_share !== 100)
      setRate(existing.rate != null ? String(existing.rate).replace('.', ',') : '')
      setItemId(existing.item_id)
    } else if (isNew) {
      const item = trip.items.find((i) => String(i.id) === params.get('item'))
      const t = today()
      setDate(item?.start_date || (t >= trip.trip.start_date && t <= trip.trip.end_date ? t : trip.trip.start_date))
      setCurrency(s.expenses.at(-1)?.currency ?? 'BRL')
      setPaidBy(me.id)
      if (item) (setTitle(item.title), setCategory(categoryFor(item.kind)), setItemId(item.id))
    } else return
    loaded.current = true
  }, [trip, s, me, existing, isNew, params])

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    const cents = parseMoney(amount)
    if (cents == null) return setError('Digite um valor, por exemplo 12,50')
    if (!/^[A-Z]{3}$/.test(currency)) return setError('Moeda com três letras, por exemplo EUR')
    const r = rate.trim() ? Number(rate.replace(',', '.')) : null
    const body = { item_id: itemId, title, amount_cents: cents, currency, category, paid_by: paidBy, payer_share: share, date, rate: r }
    try {
      await api<Expense>(isNew ? `/api/trips/${id}/expenses` : `/api/expenses/${expenseId}`, json(isNew ? 'POST' : 'PUT', body))
      await qc.invalidateQueries({ queryKey: ['expenses', id] })
      back()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!confirm('Apagar este gasto?')) return
    await api(`/api/expenses/${expenseId}`, { method: 'DELETE' })
    await qc.invalidateQueries({ queryKey: ['expenses', id] })
    back()
  }

  if (!isNew && s && !existing) return <p className="p-6">{synced ? 'Abrindo…' : 'Gasto não encontrado.'}</p>
  const seg = (on: boolean) => `rounded-lg border px-2 py-2.5 text-sm ${on ? 'border-ink bg-ink text-white' : 'border-line bg-paper'}`
  const chip = (on: boolean) => `rounded-full border px-3 py-1.5 text-sm ${on ? 'border-ink bg-ink text-white' : 'border-line bg-paper'}`

  return (
    <main className="mx-auto max-w-xl px-4 pb-24">
      <BackBar label={trip?.trip.name ?? 'Voltar'} onBack={back} />
      <form onSubmit={submit} className="space-y-4">
        <div className="flex gap-2">
          <label className="block flex-1"><span className="mb-1 block text-sm text-muted">Valor</span>
            <input className="field board text-2xl" inputMode="decimal" required value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0,00" /></label>
          <label className="block w-28"><span className="mb-1 block text-sm text-muted">Moeda</span>
            <input className="field uppercase" list="moedas" required maxLength={3} autoCapitalize="characters" value={currency}
              onChange={(e) => setCurrency(e.target.value.toUpperCase().trim())} onFocus={(e) => e.target.select()} />
            <datalist id="moedas">{CURRENCIES.map((c) => <option key={c} value={c} />)}</datalist></label>
        </div>
        <label className="block"><span className="mb-1 block text-sm text-muted">Descrição</span>
          <input className="field" required maxLength={200} value={title} onChange={(e) => setTitle(e.target.value)} /></label>
        <div className="flex flex-wrap gap-2" role="radiogroup" aria-label="Categoria">
          {(Object.keys(CATEGORIES) as Category[]).map((c) => (
            <button type="button" key={c} role="radio" aria-checked={category === c} onClick={() => setCategory(c)} className={chip(category === c)}>
              {CATEGORIES[c].emoji} {CATEGORIES[c].label}
            </button>
          ))}
        </div>
        <label className="block"><span className="mb-1 block text-sm text-muted">Data</span>
          <input className="field" type="date" required value={date} onChange={(e) => setDate(e.target.value)} /></label>
        <div>
          <span className="mb-1 block text-sm text-muted">Quem pagou</span>
          <div className="flex gap-2" role="radiogroup" aria-label="Quem pagou">
            {users?.map((u) => <button type="button" key={u.id} role="radio" aria-checked={paidBy === u.id} onClick={() => setPaidBy(u.id)} className={`flex-1 ${chip(paidBy === u.id)}`}>{u.name}</button>)}
          </div>
        </div>
        <div>
          <span className="mb-1 block text-sm text-muted">Divisão</span>
          <div className="grid grid-cols-3 gap-2" role="radiogroup" aria-label="Divisão">
            {SPLITS.map((o) => <button type="button" key={o.v} role="radio" aria-checked={!otherShare && share === o.v} onClick={() => (setOtherShare(false), setShare(o.v))} className={seg(!otherShare && share === o.v)}>{o.label}</button>)}
            <button type="button" role="radio" aria-checked={otherShare} onClick={() => setOtherShare(true)} className={seg(otherShare)}>Outra</button>
          </div>
          {otherShare && (
            <label className="mt-2 flex items-center gap-2 text-sm">
              <input className="field w-20" type="number" inputMode="numeric" min={0} max={100} value={share} onChange={(e) => setShare(Number(e.target.value))} aria-label="Parte de quem pagou (%)" />
              % fica com quem pagou
            </label>
          )}
        </div>
        {currency !== 'BRL' && (
          <label className="block"><span className="mb-1 block text-sm text-muted">Cotação (R$ por 1 {currency}, opcional)</span>
            <input className="field" inputMode="decimal" value={rate} onChange={(e) => setRate(e.target.value)} placeholder="Automática pela data" /></label>
        )}
        {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
        <SaveCancel save={isNew ? 'Adicionar' : 'Salvar'} onCancel={back} />
        {!isNew && <button type="button" onClick={remove} className="btn btn-danger w-full">Apagar gasto</button>}
      </form>
    </main>
  )
}
