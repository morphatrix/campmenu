// Minimal service worker: enables PWA install + an offline app-shell fallback.
// API requests (and SSE) are never intercepted; static GETs are network-first
// with a cache fallback so a stale shell never hides fresh data when online.
const CACHE = 'campmenu-v1'

self.addEventListener('install', () => self.skipWaiting())

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  )
})

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url)
  // Let API / non-GET / cross-origin requests hit the network untouched.
  if (event.request.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/api')) {
    return
  }
  event.respondWith(
    fetch(event.request)
      .then((res) => {
        const copy = res.clone()
        caches.open(CACHE).then((c) => c.put(event.request, copy)).catch(() => {})
        return res
      })
      .catch(() => caches.match(event.request).then((r) => r || caches.match('/index.html'))),
  )
})
