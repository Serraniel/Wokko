import { describe, expect, it } from 'vitest';
import { combineIngredients, parseAmount, roundAmount } from './format.ts';
import type { Step } from './api/types.gen.ts';

describe('parseAmount', () => {
	it.each([
		['1.5', 1.5],
		['1,5', 1.5],
		['1/2', 0.5],
		['1 1/2', 1.5],
		['2', 2],
		['', null],
		['abc', null],
		['-1', null],
		['1/0', null]
	])('%s', (input, want) => expect(parseAmount(input)).toBe(want));
});

describe('roundAmount', () => {
	it('keeps measurable precision', () => {
		expect(roundAmount(0.333333)).toBe(0.33);
		expect(roundAmount(12.34)).toBe(12.3);
		expect(roundAmount(333.33)).toBe(333);
		expect(roundAmount(0)).toBe(0);
	});
});

describe('combineIngredients', () => {
	const ing = (id: number, food: number, unit: number | null, amount: number | null, note = '') => ({
		id,
		food: { id: food, name: `food${food}`, plural: '' },
		unit: unit === null ? null : { id: unit, name: `unit${unit}`, plural: '', abbreviation: '' },
		amount,
		note
	});
	const steps = (lists: ReturnType<typeof ing>[][]) => lists.map((ingredients, id) => ({ id, name: '', instruction: '', minutes: 0, ingredients })) as Step[];

	it('adds up the same food and unit', () => {
		const got = combineIngredients(steps([[ing(1, 1, 1, 100, 'chopped')], [ing(2, 1, 1, 50, 'whole')]]));
		expect(got).toHaveLength(1);
		expect(got[0].amount).toBe(150);
		expect(got[0].notes).toEqual(['chopped', 'whole']);
	});

	it('keeps different units and missing amounts apart', () => {
		const got = combineIngredients(steps([[ing(1, 1, 1, 100), ing(2, 1, 2, 1), ing(3, 1, 1, null)]]));
		expect(got).toHaveLength(3);
	});
});
