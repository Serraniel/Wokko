import { version } from '$app/env';
import { assets, immutable } from '$app/manifest';
import { self as sw } from '$app/service-worker';

// Keeps Wokko usable with a bad connection, mainly in a shop or at the stove.
//
// App code is precached per version. Everything else prefers the network and falls back to the
// last answer, so an update is never held back by an old copy. Only the read-only addresses
// below are stored, and logging out deletes them.
const toUrl = ({ path }: { path: string }) => `/${path.replace(/^\/+/, '')}`;
const build = immutable.map(toUrl);
const files = assets.map(toUrl);
const SHELL = `wokko-shell-${version}`;
const DATA = 'wokko-data';
const MEDIA = 'wokko-media';
const MAX_DATA = 500;
const MAX_MEDIA = 300;
const NETWORK_WAIT = 4000;

const STORED_DATA = /^\/api\/v1\/(status|me|shopping|meal-plans|meal-types|books(\/\d+)?|categories|supermarkets|keywords|foods|units|nutrients|recipes(\/\d+(\/nutrition|\/books)?)?|public\/.+)$/;
const NOT_PAGES = /^\/(api|media|mcp|oauth\/(authorize|token|register)|_allauth|\.well-known)(\/|$)/;

sw.addEventListener('install', (event) => {
	event.waitUntil(
		caches.open(SHELL).then(async (cache) => {
			// One missing file shouldn't cost the whole app its offline mode.
			await Promise.allSettled([...build, ...files, '/'].map((url) => cache.add(url)));
			await sw.skipWaiting();
		})
	);
});

sw.addEventListener('activate', (event) => {
	event.waitUntil(
		caches.keys().then(async (keys) => {
			await Promise.all(keys.filter((k) => k.startsWith('wokko-shell-') && k !== SHELL).map((k) => caches.delete(k)));
			await sw.clients.claim();
		})
	);
});

async function trim(name: string, max: number) {
	const cache = await caches.open(name);
	const keys = await cache.keys();
	await Promise.all(keys.slice(0, Math.max(keys.length - max, 0)).map((k) => cache.delete(k)));
}

function clearStored() {
	return Promise.all([caches.delete(DATA), caches.delete(MEDIA)]);
}

/** Tries the network, and doesn't wait long for it when there's something stored to show. */
async function networkFirst(request: Request, cacheName: string, store: (res: Response) => Promise<void>): Promise<Response> {
	const fresh = fetch(request).then(async (res) => {
		await store(res);
		return res;
	});
	const cache = await caches.open(cacheName);
	const stored = await cache.match(request);
	if (!stored) return fresh;
	const timeout = new Promise<undefined>((resolve) => setTimeout(resolve, NETWORK_WAIT));
	try {
		return (await Promise.race([fresh.catch(() => undefined), timeout])) ?? stored;
	} catch {
		return stored;
	}
}

async function page(request: Request): Promise<Response> {
	try {
		return await fetch(request);
	} catch {
		return (await caches.match('/')) ?? Response.error();
	}
}

async function data(request: Request, path: string): Promise<Response> {
	return networkFirst(request, DATA, async (res) => {
		const cache = await caches.open(DATA);
		if (res.status === 401 && !path.startsWith('/api/v1/public/')) {
			await clearStored(); // The session is over, so what it could see goes too.
		} else if (res.ok) {
			await cache.put(request, res.clone());
			await trim(DATA, MAX_DATA);
		} else if (res.status === 404 || res.status === 403) {
			await cache.delete(request);
		}
	});
}

async function media(request: Request): Promise<Response> {
	const cache = await caches.open(MEDIA);
	const stored = await cache.match(request);
	if (stored) return stored;
	const res = await fetch(request);
	if (res.ok) {
		await cache.put(request, res.clone());
		await trim(MEDIA, MAX_MEDIA);
	}
	return res;
}

sw.addEventListener('fetch', (event) => {
	const { request } = event;
	if (request.method !== 'GET') return;
	const url = new URL(request.url);
	if (url.origin !== sw.location.origin) return;
	if (request.mode === 'navigate') {
		if (!NOT_PAGES.test(url.pathname)) event.respondWith(page(request));
		return;
	}
	if (STORED_DATA.test(url.pathname)) {
		event.respondWith(data(request, url.pathname));
	} else if (url.pathname.startsWith('/media/')) {
		event.respondWith(media(request));
	} else if (build.includes(url.pathname) || files.includes(url.pathname)) {
		event.respondWith(caches.match(request).then((hit) => hit ?? fetch(request)));
	}
});
