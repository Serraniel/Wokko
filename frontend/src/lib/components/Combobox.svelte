<script lang="ts" module>
	export interface Option {
		id: number;
		label: string;
		sub?: string;
	}
</script>

<script lang="ts">
	import { Plus } from '@lucide/svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let {
		value = $bindable(''),
		search,
		onpick,
		ontype,
		placeholder = '',
		label,
		allowCreate = true,
		id
	}: {
		value?: string;
		search: (q: string) => Promise<Option[]>;
		onpick?: (o: Option | null) => void;
		ontype?: () => void;
		placeholder?: string;
		label: string;
		allowCreate?: boolean;
		id?: string;
	} = $props();

	let open = $state(false);
	let options = $state<Option[]>([]);
	let active = $state(-1);
	let timer: ReturnType<typeof setTimeout>;
	let request = 0;
	const listId = `cb-${Math.random().toString(36).slice(2, 9)}`;

	const exact = $derived(options.some((o) => o.label.toLowerCase() === value.trim().toLowerCase()));
	const showCreate = $derived(allowCreate && value.trim() !== '' && !exact);
	const total = $derived(options.length + (showCreate ? 1 : 0));

	function run() {
		clearTimeout(timer);
		timer = setTimeout(async () => {
			const mine = ++request;
			try {
				const res = await search(value.trim());
				if (mine === request) options = res;
			} catch {
				if (mine === request) options = [];
			}
		}, 90);
	}

	function pick(o: Option) {
		value = o.label;
		open = false;
		onpick?.(o);
	}

	function create() {
		value = value.trim();
		open = false;
		onpick?.(null);
	}

	function onkeydown(e: KeyboardEvent) {
		if (!open && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
			open = true;
			run();
			return;
		}
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			active = total ? (active + 1) % total : -1;
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			active = total ? (active - 1 + total) % total : -1;
		} else if (e.key === 'Enter' && open) {
			e.preventDefault();
			if (active >= 0 && active < options.length) pick(options[active]);
			else if (active === options.length && showCreate) create();
			else if (options.length && exact) pick(options.find((o) => o.label.toLowerCase() === value.trim().toLowerCase())!);
			else if (showCreate) create();
		} else if (e.key === 'Escape') {
			open = false;
		}
	}
</script>

<div class="relative min-w-0">
	<input
		{id}
		class="input"
		bind:value
		{placeholder}
		aria-label={label}
		role="combobox"
		aria-expanded={open}
		aria-controls={listId}
		aria-autocomplete="list"
		autocomplete="off"
		autocapitalize="off"
		oninput={() => {
			open = true;
			active = -1;
			ontype?.();
			run();
		}}
		onfocus={() => {
			open = true;
			run();
		}}
		onblur={() => setTimeout(() => (open = false), 120)}
		{onkeydown}
	/>
	{#if open && total > 0}
		<ul id={listId} role="listbox" class="absolute top-full left-0 z-30 mt-1 max-h-64 w-full min-w-48 overflow-y-auto rounded-xl border border-line bg-surface py-1 shadow-card">
			{#each options as o, i (o.id)}
				<li role="option" aria-selected={i === active}>
					<button type="button" tabindex="-1" class="flex w-full items-baseline justify-between gap-2 px-3 py-2 text-left hover:bg-surface-2 {i === active ? 'bg-surface-2' : ''}" onmousedown={(e) => e.preventDefault()} onclick={() => pick(o)}>
						<span class="truncate">{o.label}</span>
						{#if o.sub}<span class="shrink-0 text-sm text-muted">{o.sub}</span>{/if}
					</button>
				</li>
			{/each}
			{#if showCreate}
				<li role="option" aria-selected={active === options.length}>
					<button type="button" tabindex="-1" class="flex w-full items-center gap-2 px-3 py-2 text-left font-medium text-brand hover:bg-surface-2 {active === options.length ? 'bg-surface-2' : ''}" onmousedown={(e) => e.preventDefault()} onclick={create}>
						<Plus size={16} class="shrink-0" /><span class="truncate">{t('common.create_named', { name: value.trim() })}</span>
					</button>
				</li>
			{/if}
		</ul>
	{/if}
</div>
