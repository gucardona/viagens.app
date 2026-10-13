import { useEffect, useRef } from 'react'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import { KINDS } from '../kinds'
import type { Item } from '../types'

const STYLE = 'https://tiles.openfreemap.org/styles/liberty'

type Props = {
  items: Item[]
  line?: boolean
  onSelect?: (item: Item) => void
  onPick?: (p: { lat: number; lng: number }) => void
  className?: string
}

function pin(item: Item, onSelect?: (item: Item) => void) {
  const el = document.createElement('button')
  el.type = 'button'
  el.setAttribute('aria-label', item.title)
  el.textContent = KINDS[item.kind].emoji
  el.style.cssText = `width:34px;height:34px;border-radius:50%;border:3px solid white;background:${KINDS[item.kind].color};font-size:16px;box-shadow:0 1px 4px rgba(18,53,91,.4);cursor:pointer`
  if (onSelect) el.addEventListener('click', (e) => (e.stopPropagation(), onSelect(item)))
  return el
}

export default function TripMap({ items, line, onSelect, onPick, className = 'h-72' }: Props) {
  const box = useRef<HTMLDivElement>(null)
  const map = useRef<maplibregl.Map>()
  const pickRef = useRef(onPick)
  pickRef.current = onPick

  useEffect(() => {
    const m = new maplibregl.Map({ container: box.current!, style: STYLE, center: [-30, 20], zoom: 1.5, attributionControl: { compact: true } })
    m.addControl(new maplibregl.NavigationControl({ showCompass: false }))
    m.on('click', (e) => pickRef.current?.({ lat: e.lngLat.lat, lng: e.lngLat.lng }))
    map.current = m
    return () => m.remove()
  }, [])

  useEffect(() => {
    const m = map.current!
    const pinned = items.filter((i) => i.lat != null && i.lng != null)
    const markers = pinned.map((i) => new maplibregl.Marker({ element: pin(i, onSelect) }).setLngLat([i.lng!, i.lat!]).addTo(m))
    const coords = pinned.map((i) => [i.lng!, i.lat!] as [number, number])

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
    else if (coords.length > 1) {
      const b = coords.reduce((acc, c) => acc.extend(c), new maplibregl.LngLatBounds(coords[0], coords[0]))
      m.fitBounds(b, { padding: 48, maxZoom: 15, duration: 0 })
    }
    return () => markers.forEach((mk) => mk.remove())
  }, [items, line, onSelect])

  return <div ref={box} className={`w-full overflow-hidden rounded-lg ${className}`} />
}

const lineData = (coords: [number, number][]) => ({
  type: "Feature" as const,
  properties: {},
  geometry: { type: "LineString" as const, coordinates: coords },
})
