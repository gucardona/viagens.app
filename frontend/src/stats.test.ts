import { describe, expect, it } from 'vitest'
import { stats } from './stats'
import type { Trip } from './types'

const trip = (id: number, start_date: string, end_date: string, countries?: string[]) => ({ id, name: `T${id}`, start_date, end_date, countries }) as Trip

describe('stats', () => {
  const trips = [
    trip(1, '2025-05-01', '2025-05-10', ['pt', 'es']),
    trip(2, '2026-10-10', '2026-10-20', ['pt']), // in progress on 2026-10-15
    trip(3, '2027-01-01', '2027-01-05', ['it', 'pt']),
    trip(4, '2027-03-01', '2027-03-02'),
  ]
  const s = stats(trips, '2026-10-15')

  it('counts done and planned trips', () => {
    expect(s.done).toBe(2)
    expect(s.planned).toBe(2)
  })

  it('counts days traveled up to today', () => {
    expect(s.days).toBe(10 + 6)
  })

  it('splits visited countries from planned ones', () => {
    expect(s.visited).toEqual([{ code: 'pt', trips: 2 }, { code: 'es', trips: 1 }])
    expect(s.upcoming).toEqual(['it'])
  })
})
