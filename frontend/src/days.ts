import type { Item, Trip } from './types'

export type DayGroup = { date: string | null; items: Item[] }

const addDay = (d: string) => new Date(Date.parse(d + 'T00:00:00Z') + 86400000).toISOString().slice(0, 10)

// Time an item shows on a given day: lodging on its check-out day uses the end time.
export const timeOn = (i: Item, date: string) => (date !== i.start_date ? i.end_time : i.start_time)

// Days of the itinerary, then "Sem data".
export function buildDays(trip: Trip, items: Item[]): DayGroup[] {
  const byDate = new Map<string, Item[]>()
  const add = (d: string, i: Item) => byDate.set(d, [...(byDate.get(d) ?? []), i])
  const undated: Item[] = []
  for (const i of items) {
    if (!i.start_date) {
      undated.push(i)
      continue
    }
    add(i.start_date, i)
    if (i.kind === 'lodging' && i.end_date && i.end_date !== i.start_date) add(i.end_date, i)
  }
  // Every day of the trip, plus a block for each item date outside it (a typo'd year must not create thousands of days).
  const dates = new Set(byDate.keys())
  for (let d = trip.start_date; d <= trip.end_date; d = addDay(d)) dates.add(d)
  const days: DayGroup[] = [...dates].sort().map((day) => {
    const t = (i: Item) => timeOn(i, day) || '99:99'
    return { date: day, items: (byDate.get(day) ?? []).sort((a, b) => t(a).localeCompare(t(b)) || a.id - b.id) }
  })
  if (undated.length) days.push({ date: null, items: undated.sort((a, b) => a.id - b.id) })
  return days
}

const minutes = (t: string) => Number(t.slice(0, 2)) * 60 + Number(t.slice(3, 5))

function duration(from: string, to: string) {
  const m = minutes(to) - minutes(from)
  if (m <= 0) return ''
  const h = Math.floor(m / 60)
  return h === 0 ? `${m}min` : `${h}h${m % 60 ? String(m % 60).padStart(2, '0') : ''}`
}

const dayDiff = (a: string, b: string) => Math.round((Date.parse(b) - Date.parse(a)) / 86400000)

// Text for an item's time on a given day: "~09:00–11:00 · 2h", "22:10–07:30 (+1)", lodging as single times.
export function timeLabel(i: Item, date: string): string {
  const start = timeOn(i, date)
  if (!start) return ''
  const label = (i.time_approx ? '~' : '') + start
  if (i.kind === 'lodging' || date !== i.start_date || !i.end_time) return label
  const later = i.end_date ? dayDiff(i.start_date, i.end_date) : 0
  if (later > 0) return `${label}–${i.end_time} (+${later})`
  const d = duration(i.start_time, i.end_time)
  return d ? `${label}–${i.end_time} · ${d}` : label
}

// Map pin numbers (1..n) for the pinned items, in the given order.
export function pinNumbers(items: Item[]): Map<number, number> {
  const out = new Map<number, number>()
  for (const i of items) if (i.lat != null) out.set(i.id, out.size + 1)
  return out
}
