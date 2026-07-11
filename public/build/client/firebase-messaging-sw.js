// MyPaas does not use Firebase Cloud Messaging.
// This no-op worker exists to quiet stale browser/extension probes for the
// default Firebase Messaging service worker path on reused localhost origins.
self.addEventListener('install', () => {
	self.skipWaiting();
});

self.addEventListener('activate', (event) => {
	event.waitUntil(self.registration.unregister());
});
