import { expect, it } from 'vitest'
import { flag } from './geo'

it('builds a flag emoji from a country code', () => {
  expect(flag('pt')).toBe('🇵🇹')
  expect(flag('')).toBe('')
})
