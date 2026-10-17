import { expect, it } from 'vitest'
import { byCurrency, categoryFor, money, parseMoney } from './money'

it('formats cents in pt-BR', () => {
  expect(money(123456)).toBe('R$ 1.234,56')
  expect(money(1050, 'EUR')).toBe('€ 10,50')
})

it('parses typed amounts into cents', () => {
  expect(parseMoney('12,50')).toBe(1250)
  expect(parseMoney('1.234,5')).toBe(123450)
  expect(parseMoney('12.50')).toBe(1250)
  expect(parseMoney('7')).toBe(700)
  expect(parseMoney('abc')).toBe(null)
  expect(parseMoney('0')).toBe(null)
})

it('suggests a category from the item kind', () => {
  expect(categoryFor('lodging')).toBe('lodging')
  expect(categoryFor('flight')).toBe('transport')
  expect(categoryFor('restaurant')).toBe('food')
  expect(categoryFor('event')).toBe('activity')
  expect(categoryFor('note')).toBe('other')
})

it('totals expenses per original currency, real first then alphabetical', () => {
  const e = (amount_cents: number, currency: string) => ({ amount_cents, currency })
  expect(byCurrency([e(100, 'EUR'), e(5000, 'HUF'), e(300, 'BRL'), e(50, 'EUR')])).toEqual([
    { currency: 'BRL', cents: 300 },
    { currency: 'EUR', cents: 150 },
    { currency: 'HUF', cents: 5000 },
  ])
  expect(byCurrency([])).toEqual([])
})
