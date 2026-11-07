<script lang="ts">
	import { ArrowDown, ArrowUp, Check, Plus, Settings2, ShoppingCart, Trash2, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { ShoppingItem, Supermarket, SupermarketCategory } from '#lib/api/types.gen.ts';
	import { cached, remember } from '#lib/cache.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { parseAmount, roundAmount } from '#lib/format.ts';
	import { formatNumber, t } from '#lib/i18n/index.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let items = $state<ShoppingItem[] | null>(cached('shopping') ?? null);
	let categories = $state<SupermarketCategory[]>([]);
	let markets = $state<Supermarket[]>([]);
	let line = $state('');
	let error = $state('');
	let marketId = $state(readMarket());
	let manageOpen = $state(false);
	let editOpen = $state(false);
	let editing = $state<Row | null>(null);
	let editForm = $state({ amount: '', unit: '', note: '', category: 0 });
	let newAisle = $state('');
	let newShop = $state('');
	// While a change is on its way, a slower background refresh mustn't undo what's on screen.
	let pending = 0;

	function readMarket(): number {
		try {
			return Number(localStorage.getItem('wokko-shop')) || 0;
		} catch {
			return 0;
		}
	}

	$effect(() => {
		try {
			localStorage.setItem('wokko-shop', String(marketId));
		} catch {
			// Only a convenience.
		}
	});

	async function load() {
		if (pending) return;
		try {
			const res = await api.get<ShoppingItem[]>('/shopping');
			if (!pending) items = remember('shopping', res);
			error = '';
		} catch (e) {
			if (!items) error = errorText(e);
		}
	}

	async function loadShops() {
		try {
			[categories, markets] = await Promise.all([api.get<SupermarketCategory[]>('/categories'), api.get<Supermarket[]>('/supermarkets')]);
		} catch (e) {
			error = errorText(e);
		}
	}

	onMount(() => {
		load();
		loadShops();
		// Household members see each other's ticks while shopping together.
		const refresh = () => {
			if (document.visibilityState === 'visible' && !editOpen && !manageOpen) load();
		};
		const timer = setInterval(refresh, 5000);
		document.addEventListener('visibilitychange', refresh);
		return () => {
			clearInterval(timer);
			document.removeEventListener('visibilitychange', refresh);
		};
	});

	interface Row {
		key: string;
		ids: number[];
		name: string;
		unit: string;
		amount: number | null;
		note: string;
		checked: boolean;
		recipes: string[];
		item: ShoppingItem;
	}

	function rows(list: ShoppingItem[]): Row[] {
		const out = new Map<string, Row>();
		for (const i of list) {
			const k = `${i.food.id}:${i.unit?.id ?? ''}:${i.amount === null || i.amount === undefined ? 'x' : ''}:${i.checked}`;
			const cur = out.get(k);
			if (cur) {
				cur.ids.push(i.id);
				if (cur.amount !== null && i.amount !== null && i.amount !== undefined) cur.amount += i.amount;
				if (i.note && !cur.note.includes(i.note)) cur.note = cur.note ? `${cur.note}; ${i.note}` : i.note;
				if (i.recipe && !cur.recipes.includes(i.recipe.name)) cur.recipes.push(i.recipe.name);
			} else {
				out.set(k, { key: k, ids: [i.id], name: i.amount && i.amount > 1 && i.food.plural ? i.food.plural : i.food.name, unit: i.unit?.abbreviation || i.unit?.name || '', amount: i.amount ?? null, note: i.note, checked: i.checked, recipes: i.recipe ? [i.recipe.name] : [], item: i });
			}
		}
		return [...out.values()];
	}

	const market = $derived(markets.find((m) => m.id === marketId));
	const groups = $derived.by(() => {
		const open = rows((items ?? []).filter((i) => !i.checked));
		const byCat = new Map<number, Row[]>();
		for (const r of open) {
			const c = r.item.category?.id ?? 0;
			byCat.set(c, [...(byCat.get(c) ?? []), r]);
		}
		const rank = (id: number) => {
			if (id === 0) return 1e9;
			const i = market?.categories.findIndex((e) => e.category.id === id) ?? -1;
			return i >= 0 ? i : 1e8;
		};
		return [...byCat.entries()].sort((a, b) => rank(a[0]) - rank(b[0])).map(([id, list]) => ({ id, name: list[0].item.category?.name ?? t('shopping.other'), rows: list.sort((x, y) => x.name.localeCompare(y.name)) }));
	});
	const done = $derived(rows((items ?? []).filter((i) => i.checked)));

	async function change(fn: () => Promise<unknown>) {
		pending++;
		try {
			await fn();
		} catch (e) {
			toasts.error(errorText(e));
		} finally {
			pending--;
		}
		await load();
	}

	function toggle(r: Row) {
		const checked = !r.checked;
		items = (items ?? []).map((i) => (r.ids.includes(i.id) ? { ...i, checked } : i));
		change(() => api.post('/shopping/check', { ids: r.ids, checked }));
	}

	function removeRow(r: Row) {
		items = (items ?? []).filter((i) => !r.ids.includes(i.id));
		editOpen = false;
		change(async () => {
			for (const id of r.ids) await api.del(`/shopping/${id}`);
		});
	}

	function clearDone() {
		items = (items ?? []).filter((i) => !i.checked);
		change(() => api.del('/shopping/checked'));
	}

	async function add(e: SubmitEvent) {
		e.preventDefault();
		const text = line.trim();
		if (!text) return;
		line = '';
		await change(async () => {
			const made = await api.post<ShoppingItem>('/shopping', { line: text });
			items = [...(items ?? []), made];
		});
	}

	function startEdit(r: Row) {
		editing = r;
		editForm = { amount: r.amount === null ? '' : formatNumber(roundAmount(r.amount)), unit: r.item.unit?.name ?? '', note: r.note, category: r.item.category?.id ?? 0 };
		editOpen = true;
	}

	async function saveEdit(e: SubmitEvent) {
		e.preventDefault();
		const r = editing;
		if (!r) return;
		const form = $state.snapshot(editForm);
		editOpen = false;
		await change(async () => {
			if ((r.item.category?.id ?? 0) !== form.category) await api.put(`/foods/${r.item.food.id}/category`, { category_id: form.category });
			if (r.ids.length === 1) await api.put(`/shopping/${r.ids[0]}`, { food_id: r.item.food.id, unit_name: form.unit, amount: parseAmount(form.amount), note: form.note, checked: r.checked });
		});
	}

	async function addAisle(e: SubmitEvent) {
		e.preventDefault();
		if (!newAisle.trim()) return;
		await change(() => api.post('/categories', { name: newAisle }));
		newAisle = '';
		await loadShops();
	}

	async function addShop(e: SubmitEvent) {
		e.preventDefault();
		if (!newShop.trim()) return;
		await change(() => api.post('/supermarkets', { name: newShop }));
		newShop = '';
		await loadShops();
	}

	async function removeAisle(id: number) {
		await change(() => api.del(`/categories/${id}`));
		await loadShops();
	}

	async function removeShop(id: number) {
		if (marketId === id) marketId = 0;
		await change(() => api.del(`/supermarkets/${id}`));
		await loadShops();
	}

	const shopOrder = (m: Supermarket) => [...m.categories.map((e) => e.category), ...categories.filter((c) => !m.categories.some((e) => e.category.id === c.id))];

	async function moveAisle(m: Supermarket, index: number, by: number) {
		const order = shopOrder(m).map((c) => c.id);
		const j = index + by;
		if (j < 0 || j >= order.length) return;
		[order[index], order[j]] = [order[j], order[index]];
		await change(() => api.put(`/supermarkets/${m.id}/order`, { category_ids: order }));
		await loadShops();
	}

	const amountText = (r: Row) => [r.amount === null ? '' : formatNumber(roundAmount(r.amount)), r.unit].filter(Boolean).join(' ');
</script>

<svelte:head><title>{t('shopping.title')} · Wokko</title></svelte:head>

{#snippet itemRow(r: Row)}
	<li class="flex items-center gap-1">
		<button class="flex h-11 w-11 shrink-0 items-center justify-center" role="checkbox" aria-checked={r.checked} aria-label={r.name} onclick={() => toggle(r)}>
			<span class="flex h-5 w-5 items-center justify-center rounded-md {r.checked ? 'bg-ink text-bg' : 'ring-1 ring-line ring-inset'}">{#if r.checked}<Check size={14} strokeWidth={3} />{/if}</span>
		</button>
		<button class="min-h-11 min-w-0 flex-1 py-1.5 text-left" onclick={() => startEdit(r)} aria-label={t('shopping.edit')}>
			<span class="block text-sm {r.checked ? 'text-muted line-through' : ''}">{#if amountText(r)}<span class="font-semibold tabular-nums">{amountText(r)} </span>{/if}{r.name}{#if r.note}<span class="text-muted">, {r.note}</span>{/if}</span>
			{#if r.recipes.length}<span class="block text-xs text-muted">{t('shopping.for_recipe', { name: r.recipes.join(', ') })}</span>{/if}
		</button>
	</li>
{/snippet}

<div class="mx-auto max-w-2xl space-y-3 p-4 md:px-6">
	<header class="flex items-center justify-between gap-2">
		<h1 class="text-xl font-semibold">{t('shopping.title')}</h1>
		<div class="flex items-center gap-1">
			{#if markets.length}
				<select class="input !min-h-9 !w-auto !py-0 !text-sm" bind:value={marketId} aria-label={t('shopping.supermarket')}>
					<option value={0}>{t('shopping.any_shop')}</option>
					{#each markets as m (m.id)}<option value={m.id}>{m.name}</option>{/each}
				</select>
			{/if}
			<button class="btn btn-ghost btn-icon text-muted" aria-label={t('shopping.manage')} onclick={() => (manageOpen = true)}><Settings2 size={18} /></button>
		</div>
	</header>

	<form class="flex gap-2" onsubmit={add}>
		<input id="shopping-line" class="input min-w-0 flex-1" bind:value={line} placeholder={t('shopping.add_placeholder')} aria-label={t('shopping.add_placeholder')} autocomplete="off" />
		<button class="btn btn-primary btn-icon shrink-0" aria-label={t('shopping.add')} disabled={!line.trim()}><Plus size={18} /></button>
	</form>
	<ErrorNote text={error} />

	{#if items && items.length === 0}
		<div class="flex flex-col items-center gap-2 py-14 text-center">
			<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-surface-2 text-muted"><ShoppingCart size={22} /></div>
			<p class="font-semibold">{t('shopping.empty_title')}</p>
			<p class="max-w-xs text-sm text-muted">{t('shopping.empty_text')}</p>
		</div>
	{:else if items}
		{#each groups as g (g.id)}
			<section class="card px-1 py-1" aria-label={g.name}>
				<h2 class="px-3 pt-2 text-xs font-semibold text-muted">{g.name}</h2>
				<ul>{#each g.rows as r (r.key)}{@render itemRow(r)}{/each}</ul>
			</section>
		{/each}
		{#if done.length}
			<section class="px-1" aria-label={t('shopping.done')}>
				<div class="flex items-center justify-between px-2">
					<h2 class="text-xs font-semibold text-muted">{t('shopping.done')}</h2>
					<button class="inline-flex min-h-9 items-center gap-1 text-xs font-medium text-muted hover:text-ink" onclick={clearDone}><Trash2 size={13} />{t('shopping.clear_done')}</button>
				</div>
				<ul>{#each done as r (r.key)}{@render itemRow(r)}{/each}</ul>
			</section>
		{/if}
	{:else if !error}
		<div class="space-y-2" aria-hidden="true">{#each { length: 5 } as _}<div class="skeleton h-11 w-full"></div>{/each}</div>
	{/if}
</div>

<Sheet bind:open={editOpen} title={editing?.name ?? ''}>
	<form class="space-y-3" onsubmit={saveEdit}>
		{#if editing?.ids.length === 1}
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label class="label" for="item-amount">{t('editor.amount')}</label>
					<input id="item-amount" class="input" inputmode="decimal" bind:value={editForm.amount} />
				</div>
				<div>
					<label class="label" for="item-unit">{t('editor.unit')}</label>
					<input id="item-unit" class="input" bind:value={editForm.unit} />
				</div>
			</div>
			<div>
				<label class="label" for="item-note">{t('editor.note')}</label>
				<input id="item-note" class="input" bind:value={editForm.note} />
			</div>
		{/if}
		<div>
			<label class="label" for="item-category">{t('shopping.category')}</label>
			<select id="item-category" class="input" bind:value={editForm.category}>
				<option value={0}>{t('shopping.no_category')}</option>
				{#each categories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
			</select>
		</div>
		<div class="flex gap-2">
			<button type="button" class="btn btn-danger btn-icon" aria-label={t('shopping.remove', { name: editing?.name ?? '' })} onclick={() => editing && removeRow(editing)}><Trash2 size={17} /></button>
			<button class="btn btn-primary flex-1">{t('common.save')}</button>
		</div>
	</form>
</Sheet>

<Sheet bind:open={manageOpen} title={t('shopping.manage')}>
	<div class="space-y-5">
		<section class="space-y-2">
			<h3 class="font-semibold">{t('shopping.aisles')}</h3>
			<p class="hint !mt-0">{t('shopping.aisles_hint')}</p>
			<ul class="divide-y divide-line">
				{#each categories as c (c.id)}
					<li class="flex min-h-10 items-center justify-between text-sm">{c.name}<button class="btn btn-ghost btn-icon !min-h-9 !w-9 text-muted" aria-label={t('common.delete')} onclick={() => removeAisle(c.id)}><X size={15} /></button></li>
				{/each}
			</ul>
			<form class="flex gap-2" onsubmit={addAisle}>
				<input class="input min-w-0 flex-1" bind:value={newAisle} placeholder={t('shopping.new_aisle')} aria-label={t('shopping.new_aisle')} />
				<button class="btn btn-soft btn-icon shrink-0" aria-label={t('shopping.new_aisle')}><Plus size={16} /></button>
			</form>
		</section>
		<section class="space-y-2">
			<h3 class="font-semibold">{t('shopping.shops')}</h3>
			{#each markets as m (m.id)}
				<div class="rounded-xl p-2 ring-1 ring-line ring-inset">
					<div class="flex min-h-9 items-center justify-between text-sm font-medium">{m.name}<button class="btn btn-ghost btn-icon !min-h-9 !w-9 text-muted" aria-label={t('common.delete')} onclick={() => removeShop(m.id)}><X size={15} /></button></div>
					{#if categories.length}
						<p class="hint !mt-0 mb-1">{t('shopping.order_for', { name: m.name })}</p>
						<ol class="space-y-0.5">
							{#each shopOrder(m) as c, i (c.id)}
								<li class="flex items-center gap-1 text-sm">
									<span class="flex-1">{c.name}</span>
									<button class="btn btn-ghost btn-icon !min-h-8 !w-8" disabled={i === 0} aria-label={t('editor.move_up')} onclick={() => moveAisle(m, i, -1)}><ArrowUp size={14} /></button>
									<button class="btn btn-ghost btn-icon !min-h-8 !w-8" disabled={i === categories.length - 1} aria-label={t('editor.move_down')} onclick={() => moveAisle(m, i, 1)}><ArrowDown size={14} /></button>
								</li>
							{/each}
						</ol>
					{/if}
				</div>
			{/each}
			<form class="flex gap-2" onsubmit={addShop}>
				<input class="input min-w-0 flex-1" bind:value={newShop} placeholder={t('shopping.new_shop')} aria-label={t('shopping.new_shop')} />
				<button class="btn btn-soft btn-icon shrink-0" aria-label={t('shopping.new_shop')}><Plus size={16} /></button>
			</form>
		</section>
	</div>
</Sheet>
