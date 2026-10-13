import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import TripMap from './TripMap'
import type { Item } from '../types'

// Mini map of one day. Browsers allow only ~16 WebGL maps at once, so it mounts only near the screen.
export default function DayMap({ items, numbers, to }: { items: Item[]; numbers: Map<number, number>; to: string }) {
  const box = useRef<HTMLAnchorElement>(null)
  const [near, setNear] = useState(false)
  useEffect(() => {
    const io = new IntersectionObserver(([e]) => setNear(e.isIntersecting), { rootMargin: '300px' })
    io.observe(box.current!)
    return () => io.disconnect()
  }, [])
  return (
    <Link ref={box} to={to} aria-label="Ver o dia no mapa" className="mt-3 block h-40 overflow-hidden rounded-lg bg-line">
      {near && <TripMap items={items} numbers={numbers} line interactive={false} className="pointer-events-none h-40" />}
    </Link>
  )
}
