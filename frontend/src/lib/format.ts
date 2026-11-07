import type { Ingredient, Step } from './api/types.gen.ts';

/** Rounds a scaled amount to something a cook would actually measure. */
export function roundAmount(n: number): number {
	if (n === 0) return 0;
	if (n >= 100) return Math.round(n);
	if (n >= 10) return Math.round(n * 10) / 10;
	return Math.round(n * 100) / 100;
}

export interface CombinedIngredient {
	key: string;
	name: string;
	plural: string;
	unit: Ingredient['unit'];
	amount: number | null;
	notes: string[];
}

/** Adds up the same food in the same unit across steps, for the shopping-style overview. */
export function combineIngredients(steps: Step[]): CombinedIngredient[] {
	const out = new Map<string, CombinedIngredient>();
	for (const step of steps) {
		for (const ing of step.ingredients) {
			const key = `${ing.food.id}:${ing.unit?.id ?? ''}:${ing.amount === null ? 'x' : ''}`;
			const cur = out.get(key);
			if (cur) {
				if (cur.amount !== null && ing.amount !== null) cur.amount += ing.amount;
				if (ing.note && !cur.notes.includes(ing.note)) cur.notes.push(ing.note);
			} else {
				out.set(key, { key, name: ing.food.name, plural: ing.food.plural, unit: ing.unit, amount: ing.amount, notes: ing.note ? [ing.note] : [] });
			}
		}
	}
	return [...out.values()];
}

/** Parses what people type as an amount: 1.5, 1,5, 1/2 or 1 1/2. */
export function parseAmount(input: string): number | null {
	const s = input.trim().replace(',', '.');
	if (!s) return null;
	const mixed = s.match(/^(\d+)\s+(\d+)\/(\d+)$/);
	if (mixed && Number(mixed[3])) return Number(mixed[1]) + Number(mixed[2]) / Number(mixed[3]);
	const frac = s.match(/^(\d+)\/(\d+)$/);
	if (frac && Number(frac[2])) return Number(frac[1]) / Number(frac[2]);
	const n = Number(s);
	return Number.isFinite(n) && n >= 0 ? n : null;
}
