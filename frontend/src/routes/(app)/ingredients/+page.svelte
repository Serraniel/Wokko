<script lang="ts">
	import { Plus, Search } from '@lucide/svelte';
	import { api, errorText, query } from '#lib/api/client.ts';
	import type { Food, FoodNutrition, Nutrient, Page, Translation, Unit } from '#lib/api/types.gen.ts';
	import Combobox, { type Option } from '#lib/components/Combobox.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { parseAmount } from '#lib/format.ts';
	import { currentLocale, formatNumber, languages, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	type Kind = 'foods' | 'units';
	type Entry = Food | Unit;

	let kind = $state<Kind>('foods');
	let q = $state('');
	let page = $state<Page<Entry> | null>(null);
	let error = $state('');
	let timer: ReturnType<typeof setTimeout>;
	let request = 0;

	let editOpen = $state(false);
	let editing = $state<Entry | null>(null);
	let rows = $state<Translation[]>([]);
	let aliases = $state('');
	let formError = $state('');
	let busy = $state(false);
	let mergeTarget = $state('');
	let mergeId = $state(0);

	async function load() {
		const mine = ++request;
		try {
			const p = await api.get<Page<Entry>>(`/${kind}${query({ q, limit: 100 })}`);
			if (mine === request) {
				page = p;
				error = '';
			}
		} catch (e) {
			if (mine === request) error = errorText(e);
		}
	}
	load();

	function switchKind(k: Kind) {
		kind = k;
		q = '';
		page = null;
		load();
	}

	/** One row per language the app has, plus any other language this entry already has. */
	function buildRows(existing: Translation[]): Translation[] {
		const codes = [...new Set([currentLocale(), ...languages.map((l) => l.code), ...existing.map((x) => x.locale)])];
		return codes.map((locale) => existing.find((x) => x.locale === locale) ?? { locale, name: '', plural: '', abbreviation: '' });
	}

	function openNew() {
		editing = null;
		rows = buildRows([]);
		aliases = '';
		formError = '';
		mergeTarget = '';
		mergeId = 0;
		editOpen = true;
	}

	let nutrients = $state<Nutrient[]>([]);
	let nutrition = $state<Record<number, string>>({});
	let pieceGrams = $state('');
	let nutritionLoaded = $state(false);

	async function loadNutrition(id: number) {
		nutritionLoaded = false;
		nutrition = {};
		pieceGrams = '';
		try {
			if (!nutrients.length) nutrients = await api.get<Nutrient[]>('/nutrients');
			const n = await api.get<FoodNutrition>(`/foods/${id}/nutrition`);
			nutrition = Object.fromEntries(Object.entries(n.values).map(([k, v]) => [Number(k), formatNumber(v)]));
			pieceGrams = n.piece_grams ? formatNumber(n.piece_grams) : '';
			nutritionLoaded = true;
		} catch (err) {
			formError = errorText(err);
		}
	}

	async function saveNutrition() {
		if (!editing) return;
		const values: Record<number, number> = {};
		for (const [k, v] of Object.entries(nutrition)) {
			const parsed = parseAmount(v);
			if (parsed !== null) values[Number(k)] = parsed;
		}
		try {
			await api.put(`/foods/${editing.id}/nutrition`, { piece_grams: parseAmount(pieceGrams), values });
			toasts.show(t('catalog.nutrition_saved'));
		} catch (err) {
			formError = errorText(err);
		}
	}

	function openEntry(e: Entry) {
		editing = e;
		if (kind === 'foods') loadNutrition(e.id);
		rows = buildRows(e.translations);
		aliases = e.aliases.join(', ');
		formError = '';
		mergeTarget = '';
		mergeId = 0;
		editOpen = true;
	}

	const languageName = (code: string) => languages.find((l) => l.code === code)?.name ?? code;
	const readOnly = $derived(editing !== null && !editing.can_edit);

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		formError = '';
		const body = {
			translations: rows.filter((r) => r.name.trim()),
			aliases: aliases.split(',').map((a) => a.trim()).filter(Boolean),
			default_unit_id: editing && 'default_unit' in editing ? (editing.default_unit?.id ?? null) : null
		};
		try {
			if (editing) await api.put(`/${kind}/${editing.id}`, body);
			else await api.post(`/${kind}`, body);
			editOpen = false;
			await load();
		} catch (err) {
			formError = errorText(err);
		} finally {
			busy = false;
		}
	}

	async function remove() {
		if (!editing) return;
		try {
			await api.del(`/${kind}/${editing.id}`);
			editOpen = false;
			await load();
		} catch (err) {
			formError = errorText(err);
		}
	}

	async function mergeOptions(text: string): Promise<Option[]> {
		const p = await api.get<Page<Entry>>(`/${kind}${query({ q: text, limit: 8 })}`);
		return p.items.filter((x) => x.id !== editing?.id).map((x) => ({ id: x.id, label: x.name, sub: t('catalog.uses', { count: x.uses }) }));
	}

	async function merge() {
		if (!editing || !mergeId) return;
		try {
			await api.post(`/${kind}/merge`, { source_id: editing.id, target_id: mergeId });
			toasts.show(t('catalog.merged', { source: editing.name, target: mergeTarget }));
			editOpen = false;
			await load();
		} catch (err) {
			formError = errorText(err);
		}
	}
</script>

<svelte:head><title>{t('nav.ingredients')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-3 p-4">
	<header class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">{t('nav.ingredients')}</h1>
		<button class="btn btn-soft" onclick={openNew}><Plus size={18} />{kind === 'foods' ? t('catalog.new_food') : t('catalog.new_unit')}</button>
	</header>

	<div class="grid grid-cols-2 gap-1 rounded-xl bg-surface-2 p-1" role="tablist">
		<button role="tab" aria-selected={kind === 'foods'} class="rounded-lg py-2 text-sm font-semibold transition-colors {kind === 'foods' ? 'bg-surface shadow-card' : 'text-muted'}" onclick={() => switchKind('foods')}>{t('catalog.foods')}</button>
		<button role="tab" aria-selected={kind === 'units'} class="rounded-lg py-2 text-sm font-semibold transition-colors {kind === 'units' ? 'bg-surface shadow-card' : 'text-muted'}" onclick={() => switchKind('units')}>{t('catalog.units')}</button>
	</div>

	<div class="relative">
		<Search size={18} class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" />
		<input class="input !pl-10" type="search" bind:value={q} placeholder={t('catalog.search')} aria-label={t('catalog.search')} oninput={() => { clearTimeout(timer); timer = setTimeout(load, 110); }} />
	</div>
	<ErrorNote text={error} />

	{#if page}
		<p class="text-sm text-muted">{t('catalog.count', { count: page.total })}</p>
		<ul class="card divide-y divide-line overflow-hidden">
			{#each page.items as item (item.id)}
				<li>
					<button class="flex w-full items-center justify-between gap-3 px-4 py-3 text-left hover:bg-surface-2" onclick={() => openEntry(item)}>
						<span class="min-w-0 truncate font-medium">{item.name}</span>
						{#if item.uses}<span class="shrink-0 text-sm text-muted">{t('catalog.uses', { count: item.uses })}</span>{/if}
					</button>
				</li>
			{:else}
				<li class="p-8 text-center text-muted">{t('recipes.no_results')}</li>
			{/each}
		</ul>
	{:else if !error}
		<div class="skeleton h-64 w-full"></div>
	{/if}
</div>

<Sheet bind:open={editOpen} title={editing ? editing.name : kind === 'foods' ? t('catalog.new_food') : t('catalog.new_unit')}>
	<form class="space-y-4" onsubmit={save}>
		<ErrorNote text={formError} />
		{#if readOnly}<p class="rounded-xl bg-surface-2 px-3 py-2 text-sm text-muted">{t('catalog.read_only')}</p>{/if}
		<fieldset class="space-y-3" disabled={readOnly}>
			{#each rows as row (row.locale)}
				<div class="space-y-1.5">
					<p class="label !mb-0">{languageName(row.locale)}</p>
					<div class="grid gap-2 {kind === 'units' ? 'grid-cols-3' : 'grid-cols-2'}">
						<input class="input" bind:value={row.name} placeholder={t('catalog.name')} aria-label={`${languageName(row.locale)}: ${t('catalog.name')}`} />
						<input class="input" bind:value={row.plural} placeholder={t('catalog.plural')} aria-label={`${languageName(row.locale)}: ${t('catalog.plural')}`} />
						{#if kind === 'units'}<input class="input" bind:value={row.abbreviation} placeholder={t('catalog.abbreviation')} aria-label={`${languageName(row.locale)}: ${t('catalog.abbreviation')}`} />{/if}
					</div>
				</div>
			{/each}
			<div>
				<label class="label" for="aliases">{t('catalog.aliases')}</label>
				<input id="aliases" class="input" bind:value={aliases} />
				<p class="hint">{t('catalog.aliases_hint')}</p>
			</div>
		</fieldset>
		{#if !readOnly}<button class="btn btn-primary w-full" disabled={busy}>{t('common.save')}</button>{/if}
	</form>

	{#if editing && kind === 'foods' && nutritionLoaded}
		<details class="mt-4 border-t border-line pt-3">
			<summary class="min-h-9 cursor-pointer font-medium">{t('catalog.nutrition')}</summary>
			<fieldset class="mt-2 space-y-2" disabled={readOnly}>
				<div class="grid grid-cols-2 gap-2">
					{#each nutrients as n (n.id)}
						<div>
							<label class="label" for="nutrient-{n.id}">{n.name} ({n.unit})</label>
							<input id="nutrient-{n.id}" class="input" inputmode="decimal" bind:value={nutrition[n.id]} />
						</div>
					{/each}
				</div>
				<div>
					<label class="label" for="piece-grams">{t('catalog.piece_grams')}</label>
					<input id="piece-grams" class="input" inputmode="decimal" bind:value={pieceGrams} />
				</div>
				{#if !readOnly}<button type="button" class="btn btn-soft w-full" onclick={saveNutrition}>{t('common.save')}</button>{/if}
			</fieldset>
		</details>
	{/if}

	{#if editing && session.isAdmin}
		<div class="mt-6 space-y-2 border-t border-line pt-4">
			<p class="label !mb-0">{t('catalog.merge')}</p>
			<p class="hint !mt-0">{t('catalog.merge_hint', { name: editing.name })}</p>
			<Combobox label={t('catalog.merge_target')} placeholder={t('catalog.merge_target')} bind:value={mergeTarget} search={mergeOptions} allowCreate={false} ontype={() => (mergeId = 0)} onpick={(o) => (mergeId = o?.id ?? 0)} />
			<button class="btn btn-soft w-full" disabled={!mergeId} onclick={merge}>{t('catalog.merge_button')}</button>
			{#if editing.uses === 0}
				<button class="btn btn-danger w-full" onclick={remove}>{t('common.delete')}</button>
			{/if}
		</div>
	{/if}
</Sheet>
