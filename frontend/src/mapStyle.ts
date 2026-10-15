import type { AddProtocolAction, StyleSpecification } from 'maplibre-gl'

// Map toggle, chosen at build time: VITE_MAP_PROVIDER=google with VITE_GOOGLE_MAPS_KEY draws Google's map
// (Map Tiles API, 2D tiles). Anything else keeps OpenFreeMap (OpenStreetMap), the default. See deploy/README.md.
type Env = { VITE_MAP_PROVIDER?: string; VITE_GOOGLE_MAPS_KEY?: string }

export const mapProvider = (env: Env) => (env.VITE_MAP_PROVIDER === 'google' && env.VITE_GOOGLE_MAPS_KEY ? 'google' : 'osm')

const API = 'https://tile.googleapis.com/v1'

// Loader for gmaps://z/x/y tiles. One session per page load; a failed one is retried on the next tile.
export function googleTile(key: string, f: typeof fetch = fetch): (url: string, ac: AbortController) => Promise<{ data: ArrayBuffer }> {
  let session: Promise<string> | undefined
  const getSession = () =>
    (session ??= f(`${API}/createSession?key=${key}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mapType: 'roadmap', language: 'pt-BR', region: 'BR', scale: 'scaleFactor2x', highDpi: true }),
    })
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(`Google Maps: ${r.status}`))))
      .then((s: { session: string }) => s.session)
      .catch((err) => {
        session = undefined
        throw err
      }))
  return async (url, ac) => {
    const [z, x, y] = url.replace('gmaps://', '').split('/')
    const res = await f(`${API}/2dtiles/${z}/${x}/${y}?session=${await getSession()}&key=${key}`, { signal: ac.signal })
    if (!res.ok) throw new Error(`Google Maps: ${res.status}`)
    return { data: await res.arrayBuffer() }
  }
}

const env = import.meta.env as Env

export const PROVIDER = mapProvider(env)

const loadGoogle = PROVIDER === 'google' ? googleTile(env.VITE_GOOGLE_MAPS_KEY!) : null
export const GOOGLE_PROTOCOL: AddProtocolAction | null = loadGoogle && ((p, ac) => loadGoogle(p.url, ac))

// 512 px tiles (scaleFactor2x) shown at 256 px stay sharp on phone screens.
export const STYLE: string | StyleSpecification =
  PROVIDER === 'google'
    ? {
        version: 8,
        sources: { google: { type: 'raster', tiles: ['gmaps://{z}/{x}/{y}'], tileSize: 256, maxzoom: 22, attribution: 'Mapa © Google' } },
        layers: [{ id: 'google', type: 'raster', source: 'google' }],
      }
    : 'https://tiles.openfreemap.org/styles/liberty'
