import type { Item, Trip } from './types'

export type DayGroup = { date: string | null; items: Item[] }

const addDay = (d: string) => new Date(Date.parse(d + 'T00:00:00Z') + 86400000).toISOString().slice(0, 10)

// Time an item shows on a given day: lodging on its check-out day uses the end time.
export const timeOn = (i: Item, date: string) => (date !== i.start_date ? i.end_time : i.start_time)

// Days of the itinerary: from the earliest to the latest date among the trip and its items, then "Sem data".
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
  const dates = [trip.start_date, trip.end_date, ...byDate.keys()].sort()
  const days: DayGroup[] = []
  for (let d = dates[0]; d <= dates[dates.length - 1]; d = addDay(d)) {
    const day = d
    const t = (i: Item) => timeOn(i, day) || '99:99'
    days.push({ date: d, items: (byDate.get(d) ?? []).sort((a, b) => t(a).localeCompare(t(b)) || a.id - b.id) })
  }
  if (undated.length) days.push({ date: null, items: undated.sort((a, b) => a.id - b.id) })
  return days
}
