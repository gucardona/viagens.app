import type { Trip } from './types'

const DAY = 86_400_000
const days = (from: string, to: string) => Math.round((Date.parse(to) - Date.parse(from)) / DAY) + 1

// stats sums up the trips as of today (YYYY-MM-DD): a trip counts as done once it has started.
export function stats(trips: Trip[], today: string) {
  const started = trips.filter((t) => t.start_date <= today)
  const count = new Map<string, number>()
  for (const t of started) for (const c of t.countries ?? []) count.set(c, (count.get(c) ?? 0) + 1)
  const later = new Set(trips.filter((t) => t.start_date > today).flatMap((t) => t.countries ?? []))
  return {
    done: started.length,
    planned: trips.length - started.length,
    days: started.reduce((n, t) => n + days(t.start_date, t.end_date < today ? t.end_date : today), 0),
    visited: [...count].map(([code, trips]) => ({ code, trips })).sort((a, b) => b.trips - a.trips || a.code.localeCompare(b.code)),
    upcoming: [...later].filter((c) => !count.has(c)).sort(),
  }
}
