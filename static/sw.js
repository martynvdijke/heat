const CACHE = "heat-cache-v3";
const PRECACHE_URLS = [
  "/",
  "/static/style.css",
  "/static/favicon.svg",
  "/static/manifest.json",
  "/static/vendor/bootstrap.7f1d37f0d90b.min.css",
  "/static/vendor/bootstrap.aa53d582f97e.bundle.min.js",
  "/static/vendor/fontawesome.7954fe83f51c.min.css",
  "/static/vendor/admin-nav.a01ce8d3267f.css"
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE).then((cache) => cache.addAll(PRECACHE_URLS))
  );
});

self.addEventListener("fetch", (event) => {
  if (event.request.url.startsWith(self.location.origin)) {
    event.respondWith(
      caches.match(event.request).then((cached) => {
        const fetched = fetch(event.request).then((response) => {
          if (response && response.status === 200) {
            const clone = response.clone();
            caches.open(CACHE).then((cache) => cache.put(event.request, clone));
          }
          return response;
        }).catch(() => cached);
        return cached || fetched;
      })
    );
  }
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.map((k) => { if (k !== CACHE) return caches.delete(k); }))
    )
  );
});
