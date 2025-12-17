self.addEventListener('install', (ev) => { ev.waitUntil(self.skipWaiting()); });
self.addEventListener('activate', (ev) => { ev.waitUntil(self.clients.claim()); });
self.addEventListener('fetch', (ev) => { /* basic offline support could be added here */ });
