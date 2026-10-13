import { describe, expect, it } from 'vitest'
import { buildDays } from './days'
import type { Item, Trip } from './types'

const trip = { id: 1, name: 'Lisboa', start_date: '2026-11-01', end_date: '2026-11-03' } as Trip
let nextId = 1
const item = (p: Partial<Item>): Item =>
  ({ id: nextId++, kind: 'activity', title: 'x', start_date: '', start_time: '', end_date: '', end_time: '', ...p }) as Item

describe('buildDays', () => {
  it('orders a day by time, untimed last, and keeps empty days', () => {
    const late = item({ start_date: '2026-11-02', start_time: '14:00' })
    const early = item({ start_date: '2026-11-02', start_time: '09:00' })
    const untimed = item({ start_date: '2026-11-02' })
    const days = buildDays(trip, [untimed, late, early])
    expect(days.map((d) => d.date)).toEqual(['2026-11-01', '2026-11-02', '2026-11-03'])
    expect(days[1].items).toEqual([early, late, untimed])
    expect(days[0].items).toEqual([])
  })

  it('breaks time ties by id', () => {
    const a = item({ start_date: '2026-11-01', start_time: '10:00' })
    const b = item({ start_date: '2026-11-01', start_time: '10:00' })
    expect(buildDays(trip, [b, a])[0].items).toEqual([a, b])
  })

  it('shows lodging on check-in and check-out days only', () => {
    const hotel = item({ kind: 'lodging', start_date: '2026-11-01', end_date: '2026-11-03' })
    const days = buildDays(trip, [hotel])
    expect(days[0].items).toEqual([hotel])
    expect(days[1].items).toEqual([])
    expect(days[2].items).toEqual([hotel])
  })

  it('sorts lodging on check-out day by its end time', () => {
    const hotel = item({ kind: 'lodging', start_date: '2026-11-01', start_time: '15:00', end_date: '2026-11-03', end_time: '11:00' })
    const lunch = item({ start_date: '2026-11-03', start_time: '12:00' })
    expect(buildDays(trip, [lunch, hotel])[2].items).toEqual([hotel, lunch])
  })

  it('extends the range for items outside the trip dates', () => {
    const flight = item({ kind: 'flight', start_date: '2026-10-31', start_time: '22:00' })
    const days = buildDays(trip, [flight])
    expect(days[0]).toEqual({ date: '2026-10-31', items: [flight] })
    expect(days).toHaveLength(4)
  })

  it('puts undated items last and omits that group when empty', () => {
    const idea = item({})
    const days = buildDays(trip, [idea])
    expect(days.at(-1)).toEqual({ date: null, items: [idea] })
    expect(buildDays(trip, []).some((d) => d.date === null)).toBe(false)
  })
})
