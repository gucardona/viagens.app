import { expect, it } from 'vitest'
import { addFor, navFor } from './nav'

it('opens the itinerary by default', () => {
  expect(navFor(null)).toEqual({ view: 'roteiro', bar: 'roteiro' })
})

it('keeps old links working while lighting the new bar item', () => {
  expect(navFor('salvos')).toEqual({ view: 'salvos', bar: 'roteiro' })
  expect(navFor('documentos')).toEqual({ view: 'mais', bar: 'mais' })
  expect(navFor('chat')).toEqual({ view: 'chat', bar: null })
  expect(navFor('mapa')).toEqual({ view: 'mapa', bar: 'mapa' })
})

it('falls back to the itinerary on an unknown tab', () => {
  expect(navFor('xyz')).toEqual({ view: 'roteiro', bar: 'roteiro' })
})

it('points the round + at what each view adds', () => {
  expect(addFor('roteiro', 7)).toEqual({ to: '/viagens/7/itens/novo', label: 'Novo item' })
  expect(addFor('salvos', 7)).toEqual({ to: '/viagens/7/itens/novo?salvo=1', label: 'Novo lugar' })
  expect(addFor('gastos', 7)).toEqual({ to: '/viagens/7/gastos/novo', label: 'Novo gasto' })
  expect(addFor('mapa', 7)).toBe(null)
})
