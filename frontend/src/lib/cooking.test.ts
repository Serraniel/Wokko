import { describe, expect, it } from 'vitest';
import { durationsIn } from './cooking.ts';

describe('durationsIn', () => {
	it('finds times in several languages', () => {
		expect(durationsIn('30 Minuten köcheln')).toEqual([30]);
		expect(durationsIn('bake for 1 hour, then rest 10 min.')).toEqual([60, 10]);
		expect(durationsIn('1,5 Std. ziehen lassen')).toEqual([90]);
	});
	it('uses the upper end of a range', () => {
		expect(durationsIn('10-15 Minuten braten')).toEqual([15]);
		expect(durationsIn('5 bis 7 min')).toEqual([7]);
	});
	it('includes the step time without repeating it', () => {
		expect(durationsIn('Simmer for 20 minutes', 20)).toEqual([20]);
		expect(durationsIn('Simmer', 25)).toEqual([25]);
	});
	it('ignores words that only look like times', () => {
		expect(durationsIn('Add 2 mint leaves and 3 hazelnuts')).toEqual([]);
		expect(durationsIn('200 g Mehl')).toEqual([]);
	});
});
