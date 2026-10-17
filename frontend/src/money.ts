import type { Kind } from './types'

export type Category = 'lodging' | 'transport' | 'food' | 'activity' | 'shopping' | 'other'

export const CATEGORIES: Record<Category, { label: string; emoji: string; color: string }> = {
  lodging: { label: 'Hospedagem', emoji: '🏨', color: '#7c3aed' },
  transport: { label: 'Transporte', emoji: '🚆', color: '#0891b2' },
  food: { label: 'Comida', emoji: '🍽️', color: '#ea580c' },
  activity: { label: 'Passeios', emoji: '🎟️', color: '#16a34a' },
  shopping: { label: 'Compras', emoji: '🛍️', color: '#db2777' },
  other: { label: 'Outros', emoji: '📦', color: '#64748b' },
}

export const money = (cents: number, currency = 'BRL') =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(cents / 100)

// Trip totals in the currency each expense was paid in: real first, then alphabetical.
export function byCurrency(expenses: { amount_cents: number; currency: string }[]) {
  const sum = new Map<string, number>()
  for (const e of expenses) sum.set(e.currency, (sum.get(e.currency) ?? 0) + e.amount_cents)
  return [...sum].map(([currency, cents]) => ({ currency, cents }))
    .sort((a, b) => (a.currency === 'BRL' ? -1 : b.currency === 'BRL' ? 1 : a.currency.localeCompare(b.currency)))
}

// "1.234,50" / "12.50" / "7" -> cents; null if not a positive amount.
export function parseMoney(s: string): number | null {
  let t = s.trim().replace(/\s/g, '')
  if (t.includes(',')) t = t.replace(/\./g, '').replace(',', '.')
  if (!/^\d+(\.\d{1,2})?$/.test(t)) return null
  const cents = Math.round(Number(t) * 100)
  return cents > 0 ? cents : null
}

const BY_KIND: Partial<Record<Kind, Category>> = {
  lodging: 'lodging', flight: 'transport', car: 'transport', transport: 'transport',
  restaurant: 'food', activity: 'activity', event: 'activity',
}
export const categoryFor = (k: Kind): Category => BY_KIND[k] ?? 'other'
