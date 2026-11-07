export interface Toast {
	id: number;
	text: string;
	kind: 'info' | 'error';
	action?: { label: string; run: () => void };
}

let next = 1;

class Toasts {
	items = $state<Toast[]>([]);

	show(text: string, opts: { kind?: Toast['kind']; action?: Toast['action'] } = {}) {
		const id = next++;
		this.items.push({ id, text, kind: opts.kind ?? 'info', action: opts.action });
		setTimeout(() => this.dismiss(id), opts.action ? 7000 : 4000);
	}

	error(text: string) {
		this.show(text, { kind: 'error' });
	}

	dismiss(id: number) {
		this.items = this.items.filter((t) => t.id !== id);
	}
}

export const toasts = new Toasts();
