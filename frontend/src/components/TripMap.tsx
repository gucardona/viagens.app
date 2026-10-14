import { useEffect, useRef } from 'react'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import { KINDS } from '../kinds'
import { groupNear } from '../cluster'
import type { Item } from '../types'

const STYLE = 'https://tiles.openfreemap.org/styles/liberty'

type Props = {
  items: Item[]
  line?: boolean
  onSelect?: (item: Item) => void
  onPick?: (p: { lat: number; lng: number }) => void
  numbers?: Map<number, number>
  colors?: Map<number, string>
  extra?: Extra[]
  onMove?: (c: { lat: number; lng: number }) => void
  geolocate?: boolean
  cluster?: boolean
  interactive?: boolean
  className?: string
}

export type Extra = { lat: number; lng: number; emoji: string; label: string }

function extraPin(x: Extra) {
  const el = document.createElement('div')
  el.title = x.label
  el.textContent = x.emoji
  el.style.cssText = 'width:26px;height:26px;border-radius:50%;background:white;display:grid;place-items:center;font-size:14px;box-shadow:0 1px 4px rgba(18,53,91,.5)'
  return el
}

function bubble(count: number, onClick: () => void) {
  const el = document.createElement('button')
  el.type = 'button'
  el.setAttribute('aria-label', `${count} lugares, toque para aproximar`)
  el.textContent = String(count)
  el.style.cssText = 'width:38px;height:38px;border-radius:50%;border:3px solid white;background:#12355B;color:white;font:700 16px var(--font-board);box-shadow:0 0 0 4px rgba(18,53,91,.25);cursor:pointer'
  el.addEventListener('click', (e) => (e.stopPropagation(), onClick()))
  return el
}

function pin(item: Item, n?: number, onSelect?: (item: Item) => void, color?: string) {
  const el = document.createElement('button')
  el.type = 'button'
  el.setAttribute('aria-label', item.title)
  el.textContent = n ? String(n) : KINDS[item.kind].emoji
  const size = n ? 26 : 34
  el.style.cssText = `width:${size}px;height:${size}px;border-radius:50%;border:${n ? 2 : 3}px solid white;background:${color ?? KINDS[item.kind].color};color:white;font:700 ${n ? 14 : 16}px var(--font-board);box-shadow:0 1px 4px rgba(18,53,91,.4);cursor:pointer`
  if (onSelect) el.addEventListener('click', (e) => (e.stopPropagation(), onSelect(item)))
  return el
}

export default function TripMap({ items, line, onSelect, onPick, numbers, colors, extra, onMove, geolocate, cluster, interactive = true, className = 'h-72' }: Props) {
  const box = useRef<HTMLDivElement>(null)
  const map = useRef<maplibregl.Map>()
  const pickRef = useRef(onPick)
  pickRef.current = onPick
  const moveRef = useRef(onMove)
  moveRef.current = onMove

  useEffect(() => {
    const m = new maplibregl.Map({ container: box.current!, style: STYLE, center: [-30, 20], zoom: 1.5, attributionControl: { compact: true }, interactive })
    if (interactive) m.addControl(new maplibregl.NavigationControl({ showCompass: false }))
    if (geolocate) m.addControl(new maplibregl.GeolocateControl({ trackUserLocation: false }))
    m.on('click', (e) => pickRef.current?.({ lat: e.lngLat.lat, lng: e.lngLat.lng }))
    m.on('moveend', () => moveRef.current?.({ lat: m.getCenter().lat, lng: m.getCenter().lng }))
    map.current = m
    return () => m.remove()
  }, [])

  useEffect(() => {
    const m = map.current!
    const pinned = items.filter((i) => i.lat != null && i.lng != null)
    const coords = pinned.map((i) => [i.lng!, i.lat!] as [number, number])
    const bounds = (cs: [number, number][]) => cs.reduce((acc, c) => acc.extend(c), new maplibregl.LngLatBounds(cs[0], cs[0]))

    // With `cluster`, pins closer than 44 px on screen become one bubble with the count, redrawn on every move.
    let markers: maplibregl.Marker[] = []
    const draw = () => {
      markers.forEach((mk) => mk.remove())
      const groups = cluster && m.getZoom() < 17 ? groupNear(coords.map((c) => m.project(c)), 44) : coords.map((_, k) => [k])
      markers = groups.map((g) => {
        if (g.length === 1) {
          const i = pinned[g[0]]
          return new maplibregl.Marker({ element: pin(i, numbers?.get(i.id), onSelect, colors?.get(i.id)) }).setLngLat(coords[g[0]]).addTo(m)
        }
        const cs = g.map((k) => coords[k])
        const center = bounds(cs).getCenter()
        return new maplibregl.Marker({ element: bubble(g.length, () => m.fitBounds(bounds(cs), { padding: 60, maxZoom: 18 })) }).setLngLat(center).addTo(m)
      })
    }
    if (cluster) m.on('moveend', draw)

    const drawLine = () => {
      if (m.getSource('route')) (m.getSource('route') as maplibregl.GeoJSONSource).setData(lineData(line ? coords : []))
      else {
        m.addSource('route', { type: 'geojson', data: lineData(line ? coords : []) })
        m.addLayer({ id: 'route', type: 'line', source: 'route', paint: { 'line-color': '#12355B', 'line-width': 3, 'line-dasharray': [2, 1.5] } })
      }
    }
    if (m.isStyleLoaded()) drawLine()
    else m.once('load', drawLine)

    if (coords.length === 1) m.jumpTo({ center: coords[0], zoom: 14 })
    else if (coords.length > 1) m.fitBounds(bounds(coords), { padding: 48, maxZoom: 15, duration: 0 })
    draw()
    return () => {
      m.off('moveend', draw)
      markers.forEach((mk) => mk.remove())
    }
  }, [items, line, onSelect, numbers, colors, cluster])

  useEffect(() => {
    const markers = (extra ?? []).map((x) => new maplibregl.Marker({ element: extraPin(x) }).setLngLat([x.lng, x.lat]).addTo(map.current!))
    return () => markers.forEach((mk) => mk.remove())
  }, [extra])

  return <div ref={box} className={`w-full overflow-hidden rounded-lg ${className}`} />
}

const lineData = (coords: [number, number][]) => ({
  type: "Feature" as const,
  properties: {},
  geometry: { type: "LineString" as const, coordinates: coords },
})
