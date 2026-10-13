import { describe, expect, it } from 'vitest'
import { cluster, dayTimes, distanceKm, walkOrder } from './cluster'

const p = (id: number, lat: number, lng: number) => ({ id, lat, lng })

describe('distanceKm', () => {
  it('measures real distance', () => {
    expect(distanceKm(p(0, 38.7, -9.1), p(0, 38.709, -9.1))).toBeCloseTo(1.0, 1)
  })
})

describe('cluster', () => {
  const near = [p(1, 38.7, -9.1), p(2, 38.7027, -9.1), p(3, 38.7054, -9.1)] // ~300 m apart
  const far = p(4, 38.745, -9.1) // ~5 km
  it('groups points within the radius and leaves isolated ones alone', () => {
    const { groups, alone } = cluster([...near, far], 1)
    expect(groups.map((g) => g.map((x) => x.id))).toEqual([[1, 2, 3]])
    expect(alone.map((x) => x.id)).toEqual([4])
  })
  it('a tiny radius leaves everything alone', () => {
    expect(cluster([...near, far], 0.1).groups).toEqual([])
  })
})

describe('walkOrder', () => {
  it('starts westmost and follows the nearest neighbour', () => {
    const pts = [p(1, 0, 0.03), p(2, 0, 0), p(3, 0, 0.01), p(4, 0.001, 0.0205)]
    expect(walkOrder(pts).map((x) => x.id)).toEqual([2, 3, 4, 1])
  })
})

describe('dayTimes', () => {
  it('spaces items 1h30 from 10:00 and stops before midnight', () => {
    expect(dayTimes(3)).toEqual(['10:00', '11:30', '13:00'])
    expect(dayTimes(11)[9]).toBe('23:30')
    expect(dayTimes(11)[10]).toBe('')
  })
})
