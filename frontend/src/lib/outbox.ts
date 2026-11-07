import type { ShoppingItem } from '#lib/api/types.gen.ts';

// Shopping changes made without a connection. They wait here in order, and survive closing the
// app, until they can be sent.
export type ShoppingOp = { kind: 'check'; ids: number[]; checked: boolean } | { kind: 'delete'; id: number } | { kind: 'clear' } | { kind: 'add'; temp: number; ref: string; line: string; checked: boolean };

/** Applies a change to the waiting list. Changes to an item not sent yet just change its add. */
export function enqueue(ops: ShoppingOp[], op: ShoppingOp): ShoppingOp[] {
	if (op.kind === 'delete' && op.id < 0) return ops.filter((o) => !(o.kind === 'add' && o.temp === op.id));
	if (op.kind === 'check' && op.ids.some((id) => id < 0)) {
		const updated = ops.map((o) => (o.kind === 'add' && op.ids.includes(o.temp) ? { ...o, checked: op.checked } : o));
		const real = op.ids.filter((id) => id > 0);
		return real.length ? [...updated, { ...op, ids: real }] : updated;
	}
	if (op.kind === 'clear') return [...ops.filter((o) => !(o.kind === 'add' && o.checked)), op];
	return [...ops, op];
}

/** Shows the list as it will be once everything waiting has been sent. */
export function applyOps(items: ShoppingItem[], ops: ShoppingOp[]): ShoppingItem[] {
	let out = items;
	for (const op of ops) {
		switch (op.kind) {
			case 'check':
				out = out.map((i) => (op.ids.includes(i.id) ? { ...i, checked: op.checked } : i));
				break;
			case 'delete':
				out = out.filter((i) => i.id !== op.id);
				break;
			case 'clear':
				out = out.filter((i) => !i.checked);
				break;
			case 'add':
				out = [...out, { id: op.temp, food: { id: 0, name: op.line, plural: '' }, unit: null, amount: null, note: '', checked: op.checked, position: Number.MAX_SAFE_INTEGER, category: null, recipe: null, created_by: null, created_at: '', updated_at: '' }];
				break;
		}
	}
	return out;
}
