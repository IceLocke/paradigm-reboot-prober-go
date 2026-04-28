const CACHE_NAME = 'prp-images-v1'
const COVER_PATTERN = /^\/cover\/.+/

self.addEventListener('fetch', (event) => {
  const { request } = event
  const url = new URL(request.url)

  // Only cache GET requests for cover images
  if (request.method !== 'GET' || !COVER_PATTERN.test(url.pathname)) {
    return
  }

  event.respondWith(
    caches.open(CACHE_NAME).then((cache) =>
      cache.match(request).then((cachedResponse) => {
        if (cachedResponse) {
          return cachedResponse
        }

        return fetch(request).then((networkResponse) => {
          if (!networkResponse || networkResponse.status !== 200) {
            return networkResponse
          }
          cache.put(request, networkResponse.clone())
          return networkResponse
        })
      })
    )
  )
})

// Clean up old caches when the service worker activates
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) =>
      Promise.all(
        cacheNames
          .filter((name) => name !== CACHE_NAME)
          .map((name) => caches.delete(name))
      )
    )
  )
})
