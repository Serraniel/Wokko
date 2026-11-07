import { api } from '#lib/api/client.ts';
import type { Recipe } from '#lib/api/types.gen.ts';
import { cached, remember } from '#lib/cache.ts';

const pending = new Set<number>();

/** Loads a recipe the moment someone aims at it, so its page opens with content already there. */
export function warmRecipe(id: number): void {
	if (pending.has(id) || cached(`recipe:${id}`)) return;
	pending.add(id);
	api.get<Recipe>(`/recipes/${id}`)
		.then((r) => remember(`recipe:${id}`, r))
		.catch(() => {}) // Only a head start. The page itself reports any real problem.
		.finally(() => pending.delete(id));
}
