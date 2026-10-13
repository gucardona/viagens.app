// Offline support, written by hand: app files cache-first, API network-first with the last copy as fallback.
// shortcut: old hashed assets are never evicted from the shell cache; a few hundred KB per deploy. Clear it if it grows.
const SHELL = 'shell-v1'
const DATA = 'data-v1'

self.addEventListener('install', () => self.skipWaiting())
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()))

async function networkFirst(req, cacheName, key) {
  const cache = await caches.open(cacheName)
  try {
    const res = await fetch(req)
    if (res.ok) cache.put(key ?? req, res.clone())
    return res
  } catch {
    const hit = await cache.match(key ?? req)
    if (hit) return hit
    if (req.url.includes('/api/')) {
      return new Response(JSON.stringify({ error: 'Sem internet e sem cópia guardada' }), { status: 503, headers: { 'Content-Type': 'application/json' } })
    }
    throw new Error('offline')
  }
}

async function cacheFirst(req, cacheName) {
  const cache = await caches.open(cacheName)
  const hit = await cache.match(req)
  if (hit) return hit
  const res = await fetch(req)
  if (res.ok) cache.put(req, res.clone())
  return res
}

self.addEventListener('fetch', (e) => {
  const req = e.request
  const url = new URL(req.url)
  if (req.method !== 'GET' || url.origin !== location.origin) return
  if (req.mode === 'navigate') return e.respondWith(networkFirst(req, SHELL, '/'))
  if (url.pathname.startsWith('/api/attachments/')) return e.respondWith(cacheFirst(req, DATA))
  if (url.pathname.startsWith('/api/')) return e.respondWith(networkFirst(req, DATA))
  if (url.pathname.startsWith('/assets/')) return e.respondWith(cacheFirst(req, SHELL))
  e.respondWith(networkFirst(req, SHELL))
})
