import { describe, expect, it } from 'vitest';
import type { ShoppingItem } from './api/types.gen.ts';
import { applyOps, enqueue, type ShoppingOp } from './outbox.ts';

const item = (id: number, checked = false): ShoppingItem => ({ id, food: { id: 1, name: `f${id}`, plural: '' }, unit: null, amount: null, note: '', checked, position: id, category: null, recipe: null, created_by: null, created_at: '', updated_at: '' });

describe('outbox', () => {
	it('shows waiting changes on top of the server list', () => {
		const ops: ShoppingOp[] = [{ kind: 'check', ids: [1], checked: true }, { kind: 'delete', id: 2 }, { kind: 'add', temp: -1, ref: 'r', line: '2 kg potatoes', checked: false }];
		const out = applyOps([item(1), item(2)], ops);
		expect(out.map((i) => [i.id, i.checked])).toEqual([[1, true], [-1, false]]);
		expect(out[1].food.name).toBe('2 kg potatoes');
	});
	it('clears only ticked items, in order', () => {
		expect(applyOps([item(1, true), item(2)], [{ kind: 'clear' }, { kind: 'check', ids: [2], checked: true }]).map((i) => i.id)).toEqual([2]);
	});
	it('folds changes to an unsent item into its add', () => {
		let ops = enqueue([], { kind: 'add', temp: -1, ref: 'r', line: 'milk', checked: false });
		ops = enqueue(ops, { kind: 'check', ids: [-1], checked: true });
		expect(ops).toEqual([{ kind: 'add', temp: -1, ref: 'r', line: 'milk', checked: true }]);
		expect(enqueue(ops, { kind: 'delete', id: -1 })).toEqual([]);
	});
	it('splits a tick covering sent and unsent items', () => {
		const ops = enqueue([{ kind: 'add', temp: -1, ref: 'r', line: 'milk', checked: false }], { kind: 'check', ids: [-1, 5], checked: true });
		expect(ops).toEqual([{ kind: 'add', temp: -1, ref: 'r', line: 'milk', checked: true }, { kind: 'check', ids: [5], checked: true }]);
	});
	it('drops unsent items that clearing would remove', () => {
		expect(enqueue([{ kind: 'add', temp: -1, ref: 'r', line: 'milk', checked: true }], { kind: 'clear' })).toEqual([{ kind: 'clear' }]);
	});
});
