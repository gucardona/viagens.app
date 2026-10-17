import { expect, it } from 'vitest'
import { navFor } from './nav'

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
