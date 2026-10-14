import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'

// useLive refetches what is on screen whenever the other person changes something.
// After a dropped connection (tunnel, plane) the browser reconnects and we refetch everything.
export function useLive() {
  const qc = useQueryClient()
  const off = useLocation().pathname === '/login'
  useEffect(() => {
    if (off || typeof EventSource === 'undefined') return
    const es = new EventSource('/api/events')
    let reconnect = false
    es.onopen = () => { if (reconnect) qc.invalidateQueries(); reconnect = true }
    es.onmessage = () => qc.invalidateQueries()
    return () => es.close()
  }, [off, qc])
}

export function Live() {
  useLive()
  return null
}
