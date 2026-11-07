import { query } from '#lib/api/client.ts';
import type { MealPlan, Page, RecipeSummary } from '#lib/api/types.gen.ts';
import { addDays, isoDate, startOfWeek } from '#lib/format.ts';

/** Offline mode exists only with a service worker, which browsers allow on HTTPS and localhost. */
export function offlineAvailable(): boolean {
	return typeof navigator !== 'undefined' && 'serviceWorker' in navigator;
}

/** Deletes what's stored for offline use, when someone logs out or their session ends. */
export function clearOfflineData(): void {
	if (typeof caches === 'undefined') return;
	void caches.delete('wokko-data');
	void caches.delete('wokko-media');
}

async function get<T>(path: string): Promise<T | null> {
	try {
		const res = await fetch(`/api/v1${path}`);
		return res.ok ? ((await res.json()) as T) : null;
	} catch {
		return null;
	}
}

/**
 * Stores what's likely needed away from a connection: the shopping list, this week's plan and
 * its recipes, and the newest recipes. Fetching is enough, because the service worker keeps the
 * answers. It runs once per visit, with the same addresses the pages use.
 */
export async function prepareForOffline(): Promise<void> {
	if (!offlineAvailable() || !navigator.onLine || !navigator.serviceWorker.controller) return;
	try {
		if (sessionStorage.getItem('wokko-prepared')) return;
		sessionStorage.setItem('wokko-prepared', '1');
	} catch {
		// Only saves a little work.
	}
	const week = startOfWeek(new Date());
	// Status and account are how the app starts, so without them it couldn't open offline at all.
	const [plans, list] = await Promise.all([
		get<MealPlan[]>(`/meal-plans${query({ from: isoDate(week), to: isoDate(addDays(week, 6)) })}`),
		get<Page<RecipeSummary>>(`/recipes${query({ q: '', keyword: 0, sort: 'updated', limit: 48 })}`),
		get('/status'),
		get('/me'),
		get('/shopping'),
		get('/meal-types'),
		get('/keywords?limit=12'),
		get('/categories'),
		get('/supermarkets')
	]);
	const ids = new Set<number>();
	for (const p of plans ?? []) if (p.recipe) ids.add(p.recipe.id);
	for (const r of (list?.items ?? []).slice(0, 20)) ids.add(r.id);
	for (const id of ids) await get(`/recipes/${id}`);
}
