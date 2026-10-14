type Pt = { lat: number; lng: number }

export function distanceKm(a: Pt, b: Pt) {
  const r = Math.PI / 180
  const h = Math.sin(((b.lat - a.lat) * r) / 2) ** 2 + Math.cos(a.lat * r) * Math.cos(b.lat * r) * Math.sin(((b.lng - a.lng) * r) / 2) ** 2
  return 2 * 6371 * Math.asin(Math.sqrt(h))
}

// DBSCAN with minPts = 2: points chained by neighbours within epsKm form a group.
export function cluster<T extends Pt>(points: T[], epsKm: number): { groups: T[][]; alone: T[] } {
  const label = new Array<number>(points.length).fill(-1)
  const groups: T[][] = []
  const alone: T[] = []
  const near = (i: number) => points.flatMap((p, j) => (j !== i && distanceKm(points[i], p) <= epsKm ? [j] : []))
  points.forEach((p, i) => {
    if (label[i] !== -1) return
    if (!near(i).length) return alone.push(p)
    const g = groups.length
    groups.push([])
    const queue = [i]
    label[i] = g
    while (queue.length) {
      const k = queue.shift()!
      groups[g].push(points[k])
      for (const j of near(k)) if (label[j] === -1) (label[j] = g), queue.push(j)
    }
  })
  return { groups, alone }
}

// Nearest-neighbour walk starting at the westmost point (straight lines, not real routes).
export function walkOrder<T extends Pt>(points: T[]): T[] {
  const left = [...points].sort((a, b) => a.lng - b.lng)
  const out: T[] = left.length ? [left.shift()!] : []
  while (left.length) {
    const last = out[out.length - 1]
    let best = 0
    left.forEach((p, i) => distanceKm(last, p) < distanceKm(last, left[best]) && (best = i))
    out.push(left.splice(best, 1)[0])
  }
  return out
}

// Approximate times for n items of a day: 10:00, 11:30, ...; '' once past 23:59.
export const dayTimes = (n: number) =>
  Array.from({ length: n }, (_, k) => {
    const m = 600 + 90 * k
    return m < 1440 ? `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}` : ''
  })

// groupNear joins screen points (px) closer than `px` to the first point of a group; returns index groups.
export function groupNear(points: { x: number; y: number }[], px: number): number[][] {
  const groups: number[][] = []
  points.forEach((p, i) => {
    const g = groups.find((g) => Math.hypot(points[g[0]].x - p.x, points[g[0]].y - p.y) < px)
    if (g) g.push(i)
    else groups.push([i])
  })
  return groups
}
