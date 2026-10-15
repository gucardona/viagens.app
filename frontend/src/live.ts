import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { send } from './outbox'

// useLive refetches what is on screen whenever the other person changes something.
// After a dropped connection (tunnel, plane) the browser reconnects and we refetch everything.
export function useLive() {
  const qc = useQueryClient()
  const off = useLocation().pathname === '/login'
  useEffect(() => {
    if (off) return
    // Send what was saved without internet, then show the server's version.
    const sync = () => send().then((sent) => { if (sent) qc.invalidateQueries() })
    const es = new EventSource('/api/events')
    let reconnect = false
    es.onopen = () => { sync(); if (reconnect) qc.invalidateQueries(); reconnect = true }
    es.onmessage = () => qc.invalidateQueries()
    addEventListener('online', sync)
    return () => (es.close(), removeEventListener('online', sync))
  }, [off, qc])
}

export function Live() {
  useLive()
  return null
}
