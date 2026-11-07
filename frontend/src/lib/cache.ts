// Remembers the last result per key, so returning to a page shows content immediately while
// fresh data loads in the background. It lasts until the tab is reloaded.
const store = new Map<string, unknown>();

export function cached<T>(key: string): T | undefined {
	return store.get(key) as T | undefined;
}

export function remember<T>(key: string, value: T): T {
	store.set(key, value);
	return value;
}

export function forget(prefix = ''): void {
	for (const k of store.keys()) if (k.startsWith(prefix)) store.delete(k);
}
