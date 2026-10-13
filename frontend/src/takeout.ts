export type Place = { title: string; notes: string; url: string; lat: number | null; lng: number | null }

// RFC 4180 CSV: quoted fields may hold commas, newlines and "" for a quote.
export function parseCSV(text: string): string[][] {
  const rows: string[][] = []
  let row: string[] = []
  let field = ''
  let quoted = false
  for (let i = 0; i < text.length; i++) {
    const c = text[i]
    if (quoted) {
      if (c === '"' && text[i + 1] === '"') (field += '"'), i++
      else if (c === '"') quoted = false
      else field += c
    } else if (c === '"') quoted = true
    else if (c === ',') row.push(field), (field = '')
    else if (c === '\n' || c === '\r') {
      if (c === '\r' && text[i + 1] === '\n') i++
      row.push(field), rows.push(row), (row = []), (field = '')
    } else field += c
  }
  if (field || row.length) row.push(field), rows.push(row)
  return rows.filter((r) => r.some((f) => f.trim()))
}

type Feature = {
  geometry?: { coordinates?: [number, number] }
  properties?: { Title?: string; name?: string; Comment?: string; google_maps_url?: string; location?: { name?: string; address?: string } }
}

// Places from a Google Takeout file: starred places (GeoJSON) or a saved list (CSV: Title, Note, URL, ...).
export function parseTakeout(name: string, text: string): Place[] {
  if (/\.csv$/i.test(name)) {
    const [head, ...rows] = parseCSV(text)
    const col = (n: string) => head.findIndex((h) => h.trim().toLowerCase() === n)
    const [t, n, u] = [col('title'), col('note'), col('url')]
    return rows
      .map((r) => ({ title: (r[t] ?? '').trim(), notes: (r[n] ?? '').trim(), url: (r[u] ?? '').trim(), lat: null, lng: null }))
      .filter((p) => p.title)
  }
  const features: Feature[] = JSON.parse(text).features ?? []
  return features
    .map((f) => {
      const pr = f.properties ?? {}
      const [lng, lat] = f.geometry?.coordinates ?? [0, 0]
      const pinned = lat !== 0 || lng !== 0
      return {
        title: (pr.location?.name ?? pr.Title ?? pr.name ?? pr.location?.address ?? '').trim(),
        notes: (pr.Comment ?? '').trim(),
        url: pr.google_maps_url ?? '',
        lat: pinned ? lat : null,
        lng: pinned ? lng : null,
      }
    })
    .filter((p) => p.title)
}
