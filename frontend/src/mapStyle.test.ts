import { describe, expect, it, vi } from 'vitest'
import { googleTile, mapProvider } from './mapStyle'

describe('mapProvider', () => {
  it('keeps OpenStreetMap unless Google is asked for with a key', () => {
    expect(mapProvider({})).toBe('osm')
    expect(mapProvider({ VITE_MAP_PROVIDER: 'google' })).toBe('osm')
    expect(mapProvider({ VITE_MAP_PROVIDER: 'osm', VITE_GOOGLE_MAPS_KEY: 'k' })).toBe('osm')
    expect(mapProvider({ VITE_MAP_PROVIDER: 'google', VITE_GOOGLE_MAPS_KEY: 'k' })).toBe('google')
  })
})

describe('googleTile', () => {
  it('creates one session and fetches tiles with it', async () => {
    const f = vi.fn(async (url: string, _init?: RequestInit) => {
      if (url.includes('createSession')) return new Response(JSON.stringify({ session: 'S1' }))
      return new Response(new Uint8Array([1, 2, 3]))
    })
    const load = googleTile('KEY', f as typeof fetch)
    const a = await load('gmaps://14/9060/5740', new AbortController())
    await load('gmaps://14/9061/5740', new AbortController())

    const urls = f.mock.calls.map((c) => c[0])
    expect(urls.filter((u) => u.includes('createSession'))).toEqual(['https://tile.googleapis.com/v1/createSession?key=KEY'])
    expect(JSON.parse(f.mock.calls[0][1]!.body as string)).toMatchObject({ mapType: 'roadmap', language: 'pt-BR' })
    expect(urls).toContain('https://tile.googleapis.com/v1/2dtiles/14/9060/5740?session=S1&key=KEY')
    expect(new Uint8Array(a.data)).toEqual(new Uint8Array([1, 2, 3]))
  })

  it('tries a new session after a failed one', async () => {
    let fail = true
    const f = vi.fn(async (url: string) => {
      if (url.includes('createSession')) return fail ? new Response('nope', { status: 403 }) : new Response(JSON.stringify({ session: 'S2' }))
      return new Response(new Uint8Array([9]))
    })
    const load = googleTile('KEY', f as typeof fetch)
    await expect(load('gmaps://1/0/0', new AbortController())).rejects.toThrow()
    fail = false
    await load('gmaps://1/0/0', new AbortController())
    expect(f.mock.calls.at(-1)![0]).toBe('https://tile.googleapis.com/v1/2dtiles/1/0/0?session=S2&key=KEY')
  })
})
