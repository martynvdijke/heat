// Service worker registration + "always pick up the newest release" logic.
//
// The build pipeline (build.mjs) content-hashes the cache version inside sw.js,
// so every frontend release ships a byte-different service worker. Browsers only
// re-check /sw.js on navigation (or every ~24h), though, so a page left open
// would keep running the previous release forever. This module closes that gap:
//  1. register() with updateViaCache:'none' so the SW script is never served
//     from the HTTP cache.
//  2. controllerchange -> reload: as soon as a new SW takes control (sw.js uses
//     skipWaiting + clients.claim), reload the page so the user lands on the
//     new release instead of stale markup.
//  3. periodic update() polling + visibilitychange so long-lived tabs still
//     detect a release they would otherwise never see.
const SW_PATH = '/sw.js';
const UPDATE_INTERVAL_MS = 60 * 60 * 1000; // 1 hour

if ('serviceWorker' in navigator) {
    navigator.serviceWorker
        .register(SW_PATH, { updateViaCache: 'none' })
        .then((registration) => {
            // Only auto-reload when a NEW service worker replaces one that was
            // already controlling this page. Skip the first-ever visit (where
            // the controller goes null -> set); reloading then is pointless.
            if (navigator.serviceWorker.controller) {
                let refreshing = false;
                navigator.serviceWorker.addEventListener('controllerchange', () => {
                    if (refreshing) return;
                    refreshing = true;
                    window.location.reload();
                });
            }

            // Idle tabs never navigate, so the browser's update check never
            // fires. Poll periodically and re-check whenever the tab becomes
            // visible again.
            const checkForUpdate = () => {
                registration.update().catch(() => {
                    /* offline or transient failure — retry on next tick */
                });
            };
            window.setInterval(checkForUpdate, UPDATE_INTERVAL_MS);
            document.addEventListener('visibilitychange', () => {
                if (document.visibilityState === 'visible') {
                    checkForUpdate();
                }
            });
        })
        .catch((err) => {
            console.log('SW registration failed:', err);
        });
}
