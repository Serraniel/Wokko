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

const FRACTIONS: [number, string][] = [[1 / 8, '⅛'], [1 / 4, '¼'], [1 / 3, '⅓'], [1 / 2, '½'], [2 / 3, '⅔'], [3 / 4, '¾']];

/** Writes 1.5 as "1½" when a common kitchen fraction is close enough. Otherwise it returns null. */
export function asFraction(n: number): string | null {
	if (n <= 0 || n >= 100) return null;
	const whole = Math.floor(n);
	const rest = n - whole;
	if (rest < 0.01) return String(whole);
	if (rest > 0.99) return String(whole + 1);
	for (const [value, sign] of FRACTIONS) {
		if (Math.abs(rest - value) < 0.02) return (whole || '') + sign;
	}
	return null;
}

/** Monday of the week containing a date, in local time, as YYYY-MM-DD parts are awkward otherwise. */
export function startOfWeek(d: Date): Date {
	const out = new Date(d.getFullYear(), d.getMonth(), d.getDate());
	out.setDate(out.getDate() - ((out.getDay() + 6) % 7));
	return out;
}

export function isoDate(d: Date): string {
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

export function addDays(d: Date, days: number): Date {
	const out = new Date(d);
	out.setDate(out.getDate() + days);
	return out;
}
