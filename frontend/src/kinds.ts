import type { Kind } from './types'

export const KINDS: Record<Kind, { label: string; emoji: string; color: string }> = {
  flight: { label: 'Voo', emoji: '✈️', color: '#2563eb' },
  lodging: { label: 'Hospedagem', emoji: '🏨', color: '#7c3aed' },
  car: { label: 'Carro', emoji: '🚗', color: '#0891b2' },
  transport: { label: 'Transporte', emoji: '🚆', color: '#0d9488' },
  restaurant: { label: 'Restaurante', emoji: '🍽️', color: '#ea580c' },
  activity: { label: 'Passeio', emoji: '🎟️', color: '#16a34a' },
  event: { label: 'Evento', emoji: '🎭', color: '#db2777' },
  note: { label: 'Nota', emoji: '📝', color: '#64748b' },
}
