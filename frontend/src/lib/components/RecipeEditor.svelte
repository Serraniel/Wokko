<script lang="ts">
	import EditorHints from './EditorHints.svelte';
	import { ArrowDown, ArrowUp, Camera, ChevronDown, Plus, Trash2, X } from '@lucide/svelte';
	import { onMount, untrack } from 'svelte';
	import { goto, replaceState } from '$app/navigation';
	import { page } from '$app/state';
	import { api, ApiError, errorText, query } from '#lib/api/client.ts';
	import type { Food, ImportedRecipe, Keyword, Page, Recipe, RecipeInput, Unit } from '#lib/api/types.gen.ts';
	import { forget, remember } from '#lib/cache.ts';
	import { clearDraft, draftKey, loadDraft, saveDraft } from '#lib/draft.ts';
	import { parseAmount } from '#lib/format.ts';
	import { formatDate, formatNumber, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { readShared } from '#lib/share.ts';
	import { toasts } from '#lib/toast.svelte.ts';
	import Combobox, { type Option } from './Combobox.svelte';
	import ErrorNote from './ErrorNote.svelte';
	import ImportPanel from './ImportPanel.svelte';

	let { recipe = null }: { recipe?: Recipe | null } = $props();

	interface IngredientRow {
		key: number;
		food_id: number;
		food_name: string;
		unit_id: number;
		unit_name: string;
		amount: string;
		note: string;
		showNote: boolean;
	}
	interface StepRow {
		key: number;
		name: string;
		instruction: string;
		minutes: string;
		ingredients: IngredientRow[];
	}
	interface Form {
		name: string;
		description: string;
		work: string;
		wait: string;
		servings: string;
		source: string;
		visibility: string;
		keywords: string[];
		steps: StepRow[];
	}

	let nextKey = 1;
	const newIngredient = (): IngredientRow => ({ key: nextKey++, food_id: 0, food_name: '', unit_id: 0, unit_name: '', amount: '', note: '', showNote: false });
	const newStep = (): StepRow => ({ key: nextKey++, name: '', instruction: '', minutes: '', ingredients: [newIngredient()] });

	function fromRecipe(r: Recipe | null): Form {
		if (!r) return { name: '', description: '', work: '', wait: '', servings: '4', source: '', visibility: 'household', keywords: [], steps: [newStep()] };
		return {
			name: r.name,
			description: r.description,
			work: r.work_minutes ? String(r.work_minutes) : '',
			wait: r.wait_minutes ? String(r.wait_minutes) : '',
			servings: String(r.servings),
			source: r.source_url,
			visibility: r.visibility,
			keywords: r.keywords.map((k) => k.name),
			steps: r.steps.map((s) => ({
				key: nextKey++,
				name: s.name,
				instruction: s.instruction,
				minutes: s.minutes ? String(s.minutes) : '',
				ingredients: s.ingredients.map((i) => ({
					key: nextKey++,
					food_id: i.food.id,
					food_name: i.food.name,
					unit_id: i.unit?.id ?? 0,
					unit_name: i.unit?.name ?? '',
					amount: i.amount === null || i.amount === undefined ? '' : formatNumber(i.amount),
					note: i.note,
					showNote: !!i.note
				}))
			}))
		};
	}

	const startRecipe = untrack(() => recipe);
	let form = $state<Form>(fromRecipe(startRecipe));
	let baseVersion = $state(startRecipe?.version ?? 0);
	let photoFile = $state<File | null>(null);
	let photoPreview = $state(startRecipe?.photo_url ?? '');
	let saving = $state(false);
	let error = $state('');
	let conflict = $state(false);
	let keywordInput = $state('');
	let foundDraft = $state<{ savedAt: number; value: Form } | null>(null);
	let showMore = $state(!!startRecipe?.source_url || !!startRecipe?.wait_minutes);
	let ready = false;

	const key = draftKey(startRecipe?.id ?? null);
	const startSnapshot = untrack(() => JSON.stringify(stripKeys($state.snapshot(form))));

	function stripKeys(f: Form) {
		return { ...f, steps: f.steps.map((s) => ({ ...s, key: 0, ingredients: s.ingredients.map((i) => ({ ...i, key: 0, showNote: false })) })) };
	}

	onMount(() => {
		const d = loadDraft<Form>(key);
		if (d && JSON.stringify(stripKeys(d.value)) !== startSnapshot) foundDraft = d;
		ready = true;
	});

	// What the kitchen knowledge looks at while editing. Nothing is saved for it.
	const hintsDraft = $derived(buildInput(true));

	// Every change is kept in the browser after a short pause, until the save succeeds.
	let draftTimer: ReturnType<typeof setTimeout>;
	$effect(() => {
		const snapshot = JSON.stringify(form);
		if (!ready) return;
		clearTimeout(draftTimer);
		if (JSON.stringify(stripKeys(JSON.parse(snapshot))) === startSnapshot) return;
		draftTimer = setTimeout(() => saveDraft(key, JSON.parse(snapshot)), 400);
	});

	function restoreDraft() {
		if (!foundDraft) return;
		const v = foundDraft.value;
		for (const s of v.steps) {
			s.key = nextKey++;
			for (const i of s.ingredients) i.key = nextKey++;
		}
		form = v;
		foundDraft = null;
	}

	function discardDraft() {
		clearDraft(key);
		foundDraft = null;
	}

	function move<T>(list: T[], i: number, by: number) {
		const j = i + by;
		if (j < 0 || j >= list.length) return;
		[list[i], list[j]] = [list[j], list[i]];
	}

	let foodCache = new Map<number, Food>();

	async function searchFoods(q: string): Promise<Option[]> {
		const p = await api.get<Page<Food>>(`/foods${query({ q, limit: 8 })}`);
		for (const f of p.items) foodCache.set(f.id, f);
		return p.items.map((f) => ({ id: f.id, label: f.name, sub: f.default_unit?.abbreviation || f.default_unit?.name }));
	}

	async function searchUnits(q: string): Promise<Option[]> {
		const p = await api.get<Page<Unit>>(`/units${query({ q, limit: 8 })}`);
		return p.items.map((u) => ({ id: u.id, label: u.name, sub: u.abbreviation }));
	}

	async function searchKeywords(q: string): Promise<Option[]> {
		const p = await api.get<Page<Keyword>>(`/keywords${query({ q, limit: 8 })}`);
		return p.items.filter((k) => !form.keywords.includes(k.name)).map((k) => ({ id: k.id, label: k.name }));
	}

	function pickFood(row: IngredientRow, o: Option | null) {
		row.food_id = o?.id ?? 0;
		// Suggest the unit this food is usually measured in, if none was chosen yet.
		if (o && !row.unit_name) {
			const u = foodCache.get(o.id)?.default_unit;
			if (u) {
				row.unit_id = u.id;
				row.unit_name = u.name;
			}
		}
	}

	function addKeyword(name: string) {
		const n = name.trim();
		if (n && !form.keywords.some((k) => k.toLowerCase() === n.toLowerCase())) form.keywords.push(n);
		keywordInput = '';
	}

	function int(s: string | number): number {
		const n = parseInt(String(s), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	}

	// quiet skips what can't be read yet instead of reporting it, for hints while typing.
	function buildInput(quiet = false): RecipeInput | null {
		const steps = [];
		for (const s of form.steps) {
			const rows = s.ingredients.filter((i) => i.food_name.trim() || i.amount.trim() || i.note.trim());
			if (!s.instruction.trim() && !s.name.trim() && rows.length === 0) continue;
			const ingredients = [];
			for (const i of rows) {
				if (!i.food_name.trim()) {
					if (quiet) continue;
					error = t('err.ingredient_required');
					return null;
				}
				const amount = parseAmount(i.amount);
				if (i.amount.trim() && amount === null && !quiet) {
					error = t('editor.bad_amount', { value: i.amount });
					return null;
				}
				ingredients.push({ food_id: i.food_id, food_name: i.food_name.trim(), unit_id: i.unit_id, unit_name: i.unit_name.trim(), amount, note: i.note.trim() });
			}
			steps.push({ name: s.name.trim(), instruction: s.instruction.trim(), minutes: int(s.minutes), ingredients });
		}
		return {
			name: form.name,
			description: form.description.trim(),
			work_minutes: int(form.work),
			wait_minutes: int(form.wait),
			servings: int(form.servings) || 1,
			servings_text: '',
			source_url: form.source.trim(),
			visibility: form.visibility,
			keywords: form.keywords,
			steps,
			base_version: baseVersion
		};
	}

	async function save() {
		error = '';
		conflict = false;
		const input = buildInput();
		if (!input) {
			window.scrollTo({ top: 0, behavior: 'smooth' });
			return;
		}
		saving = true;
		try {
			const saved = startRecipe ? await api.put<Recipe>(`/recipes/${startRecipe.id}`, input) : await api.post<Recipe>('/recipes', input);
			clearTimeout(draftTimer);
			clearDraft(key);
			ready = false;
			forget('recipes');
			let result = saved;
			if (photoFile) {
				try {
					result = await api.upload<Recipe>(`/recipes/${saved.id}/photo`, photoFile);
				} catch (e) {
					// The recipe itself is safe. Only the photo needs another try.
					toasts.error(t('editor.photo_failed', { reason: errorText(e) }));
				}
			}
			remember(`recipe:${result.id}`, result);
			await goto(`/recipes/${result.id}`, { replaceState: true });
		} catch (e) {
			if (e instanceof ApiError && e.code === 'edit_conflict') conflict = true;
			else error = errorText(e);
			window.scrollTo({ top: 0, behavior: 'smooth' });
		} finally {
			saving = false;
		}
	}

	// A deliberate choice after a conflict: this version replaces the newer one, which stays
	// in the recipe's version history.
	async function overwrite() {
		if (!startRecipe) return;
		try {
			const latest = await api.get<Recipe>(`/recipes/${startRecipe.id}`);
			baseVersion = latest.version;
			await save();
		} catch (e) {
			error = errorText(e);
		}
	}

	// The panel only shows while the form is untouched, so importing can't overwrite anyone's typing.
	const pristine = $derived(!startRecipe && JSON.stringify(stripKeys($state.snapshot(form))) === startSnapshot);
	let imported = $state<{ guessed: boolean } | null>(null);
	// A link from "Share to Wokko" or the bookmark.
	const shared = startRecipe ? { url: '', text: '' } : readShared(page.url.searchParams);

	function applyImport(d: ImportedRecipe) {
		const r = d.recipe;
		form = {
			name: r.name,
			description: r.description,
			work: r.work_minutes ? String(r.work_minutes) : '',
			wait: r.wait_minutes ? String(r.wait_minutes) : '',
			servings: String(r.servings),
			source: r.source_url,
			visibility: r.visibility,
			keywords: r.keywords,
			steps: r.steps.map((s) => ({
				key: nextKey++,
				name: s.name,
				instruction: s.instruction,
				minutes: s.minutes ? String(s.minutes) : '',
				ingredients: s.ingredients.map((i) => ({
					key: nextKey++,
					food_id: 0,
					food_name: i.food_name,
					unit_id: 0,
					unit_name: i.unit_name,
					amount: i.amount === null || i.amount === undefined ? '' : formatNumber(i.amount),
					note: i.note,
					showNote: !!i.note
				}))
			}))
		};
		showMore = true;
		imported = { guessed: d.guessed };
		if (shared.url || shared.text) replaceState('/recipes/new', {});
		if (d.image_data) {
			const bytes = Uint8Array.from(atob(d.image_data), (c) => c.charCodeAt(0));
			photoFile = new File([bytes], 'imported-photo', { type: d.image_type });
			photoPreview = URL.createObjectURL(photoFile);
		}
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}

	function choosePhoto(e: Event) {
		const file = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		photoFile = file;
		photoPreview = URL.createObjectURL(file);
	}

	const canPublic = $derived(session.status?.allow_public || startRecipe?.visibility === 'public');
	const isOwner = $derived(!startRecipe || startRecipe.can_manage);
</script>

<form class="mx-auto w-full max-w-3xl space-y-4 p-4 pb-28 md:pb-8" onsubmit={(e) => { e.preventDefault(); save(); }}>
	<h1 class="text-2xl font-bold">{startRecipe ? t('editor.edit_title') : t('editor.new_title')}</h1>

	{#if foundDraft}
		<div class="rise flex flex-wrap items-center justify-between gap-2 rounded-xl bg-warn-soft p-3 text-sm text-warn" role="status">
			<span class="font-medium">{t('editor.draft_found', { date: formatDate(new Date(foundDraft.savedAt).toISOString(), 'datetime') })}</span>
			<span class="flex gap-2">
				<button type="button" class="btn btn-soft !min-h-9" onclick={discardDraft}>{t('editor.draft_discard')}</button>
				<button type="button" class="btn btn-primary !min-h-9" onclick={restoreDraft}>{t('editor.draft_restore')}</button>
			</span>
		</div>
	{/if}
	{#if conflict}
		<div class="rise space-y-2 rounded-xl bg-warn-soft p-3 text-sm text-warn" role="alert">
			<p class="font-semibold">{t('err.edit_conflict')}</p>
			<p>{t('editor.conflict_hint')}</p>
			<div class="flex flex-wrap gap-2">
				<a class="btn btn-soft !min-h-9" href="/recipes/{startRecipe?.id}" target="_blank">{t('editor.conflict_view')}</a>
				<button type="button" class="btn btn-primary !min-h-9" onclick={overwrite}>{t('editor.conflict_overwrite')}</button>
			</div>
		</div>
	{/if}
	{#if imported}
		<div class="rise rounded-xl p-3 text-sm {imported.guessed ? 'bg-warn-soft text-warn' : 'bg-brand-soft text-brand-soft-ink'}" role="status">
			<p class="font-semibold">{t('import.done')}</p>
			{#if imported.guessed}<p>{t('import.guessed')}</p>{/if}
			<p>{t('import.split_hint')}</p>
		</div>
	{/if}
	<ErrorNote text={error} />

	{#if pristine && !imported}
		<ImportPanel onimport={applyImport} sharedUrl={shared.url} sharedText={shared.text} />
	{/if}

	<section class="card space-y-4 p-4">
		<label class="relative flex aspect-[16/9] cursor-pointer items-center justify-center overflow-hidden rounded-xl bg-surface-2 text-muted hover:bg-line">
			{#if photoPreview}
				<img src={photoPreview} alt="" class="h-full w-full object-cover" />
				<span class="absolute right-2 bottom-2 inline-flex items-center gap-1.5 rounded-lg bg-black/60 px-2 py-1 text-sm font-medium text-white"><Camera size={15} />{t('editor.change_photo')}</span>
			{:else}
				<span class="flex flex-col items-center gap-1 font-medium"><Camera size={28} />{t('editor.add_photo')}</span>
			{/if}
			<input type="file" accept="image/jpeg,image/png,image/webp" class="sr-only" onchange={choosePhoto} />
		</label>
		<div>
			<label class="label" for="name">{t('editor.name')}</label>
			<input id="name" class="input" bind:value={form.name} required maxlength="200" placeholder={t('editor.name_placeholder')} />
		</div>
		<div>
			<label class="label" for="description">{t('editor.description')}</label>
			<textarea id="description" class="input resize-y" rows="2" bind:value={form.description}></textarea>
		</div>
		<div class="grid grid-cols-2 gap-3">
			<div>
				<label class="label" for="servings">{t('recipe.servings')}</label>
				<input id="servings" class="input" type="text" inputmode="numeric" bind:value={form.servings} />
			</div>
			<div>
				<label class="label" for="work">{t('editor.work_time')}</label>
				<input id="work" class="input" type="text" inputmode="numeric" bind:value={form.work} placeholder="30" />
			</div>
		</div>
		<div>
			<label class="label" for="keyword">{t('editor.tags')}</label>
			<Combobox id="keyword" label={t('editor.tags')} bind:value={keywordInput} search={searchKeywords} placeholder={t('editor.tags_placeholder')} onpick={(o) => addKeyword(o?.label ?? keywordInput)} />
			{#if form.keywords.length}
				<div class="mt-2 flex flex-wrap gap-1.5">
					{#each form.keywords as k, i (k)}
						<span class="chip !pr-1">{k}<button type="button" class="rounded-full p-1 hover:bg-line" aria-label={t('editor.remove_tag', { name: k })} onclick={() => form.keywords.splice(i, 1)}><X size={14} /></button></span>
					{/each}
				</div>
			{/if}
		</div>
		<button type="button" class="flex min-h-9 items-center gap-1 text-sm font-medium text-muted" aria-expanded={showMore} onclick={() => (showMore = !showMore)}>
			<ChevronDown size={16} class="transition-transform {showMore ? 'rotate-180' : ''}" />{t('editor.more_options')}
		</button>
		{#if showMore}
			<div class="rise space-y-4">
				<div>
					<label class="label" for="wait">{t('editor.wait_time')}</label>
					<input id="wait" class="input" type="text" inputmode="numeric" bind:value={form.wait} />
				</div>
				<div>
					<label class="label" for="source">{t('editor.source')}</label>
					<input id="source" class="input" type="url" inputmode="url" bind:value={form.source} placeholder="https://" />
				</div>
				{#if isOwner}
					<div>
						<label class="label" for="visibility">{t('share.who_can_see')}</label>
						<select id="visibility" class="input" bind:value={form.visibility}>
							<option value="private">{t('visibility.private')}</option>
							<option value="household">{t('visibility.household')}</option>
							{#if canPublic}<option value="public">{t('visibility.public')}</option>{/if}
						</select>
					</div>
				{/if}
			</div>
		{/if}
	</section>

	<h2 class="pt-2 text-lg font-semibold">{t('recipe.steps')}</h2>
	{#each form.steps as step, s (step.key)}
		<section class="card space-y-3 p-4">
			<div class="flex items-center gap-1">
				<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-soft text-sm font-bold text-brand-soft-ink">{s + 1}</span>
				<span class="flex-1"></span>
				<button type="button" class="btn btn-ghost btn-icon !min-h-9 !w-9" disabled={s === 0} aria-label={t('editor.move_up')} onclick={() => move(form.steps, s, -1)}><ArrowUp size={17} /></button>
				<button type="button" class="btn btn-ghost btn-icon !min-h-9 !w-9" disabled={s === form.steps.length - 1} aria-label={t('editor.move_down')} onclick={() => move(form.steps, s, 1)}><ArrowDown size={17} /></button>
				<button type="button" class="btn btn-ghost btn-icon !min-h-9 !w-9 text-danger" aria-label={t('editor.remove_step')} onclick={() => form.steps.splice(s, 1)}><Trash2 size={17} /></button>
			</div>

			<div class="space-y-3">
				{#each step.ingredients as ing, i (ing.key)}
					<div class="space-y-2 border-b border-line pb-3 sm:border-0 sm:pb-0">
						<div class="grid grid-cols-[4.5rem_minmax(0,1fr)_2.75rem] gap-2 sm:grid-cols-[4.5rem_8rem_minmax(0,1fr)_2.75rem]">
							<input class="input" bind:value={ing.amount} inputmode="decimal" placeholder={t('editor.amount')} aria-label={t('editor.amount')} autocomplete="off" />
							<Combobox label={t('editor.unit')} placeholder={t('editor.unit')} bind:value={ing.unit_name} search={searchUnits} ontype={() => (ing.unit_id = 0)} onpick={(o) => (ing.unit_id = o?.id ?? 0)} />
							<div class="col-span-3 row-start-2 sm:col-span-1 sm:row-start-auto">
								<Combobox label={t('editor.ingredient')} placeholder={t('editor.ingredient')} bind:value={ing.food_name} search={searchFoods} ontype={() => (ing.food_id = 0)} onpick={(o) => pickFood(ing, o)} />
							</div>
							<button type="button" class="btn btn-ghost btn-icon col-start-3 row-start-1 text-muted sm:col-start-4" aria-label={t('editor.remove_ingredient')} onclick={() => step.ingredients.splice(i, 1)}><X size={18} /></button>
						</div>
						{#if ing.showNote}
							<input class="input" bind:value={ing.note} placeholder={t('editor.note_placeholder')} aria-label={t('editor.note')} />
						{:else}
							<button type="button" class="min-h-9 text-sm font-medium text-muted hover:text-ink" onclick={() => (ing.showNote = true)}>{t('editor.add_note')}</button>
						{/if}
					</div>
				{/each}
				<button type="button" class="btn btn-soft w-full" onclick={() => step.ingredients.push(newIngredient())}><Plus size={17} />{t('editor.add_ingredient')}</button>
			</div>

			<div>
				<label class="label" for="ins-{step.key}">{t('editor.instructions')}</label>
				<textarea id="ins-{step.key}" class="input resize-y" rows="3" bind:value={step.instruction} placeholder={t('editor.instructions_placeholder')}></textarea>
			</div>
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label class="label" for="st-{step.key}">{t('editor.step_time')}</label>
					<input id="st-{step.key}" class="input" type="text" inputmode="numeric" bind:value={step.minutes} />
				</div>
				<div>
					<label class="label" for="sn-{step.key}">{t('editor.step_name')}</label>
					<input id="sn-{step.key}" class="input" bind:value={step.name} />
				</div>
			</div>
		</section>
	{/each}
	<button type="button" class="btn btn-soft w-full" onclick={() => form.steps.push(newStep())}><Plus size={18} />{t('editor.add_step')}</button>

	<EditorHints draft={hintsDraft} />

	<!-- Always reachable: on phones it sits above the navigation bar. -->
	<div class="fixed inset-x-0 bottom-[calc(3.5rem+env(safe-area-inset-bottom))] z-20 border-t border-line bg-surface/95 p-3 backdrop-blur md:static md:border-0 md:bg-transparent md:p-0">
		<div class="mx-auto flex max-w-3xl gap-2">
			<a class="btn btn-soft flex-1 md:flex-none" href={startRecipe ? `/recipes/${startRecipe.id}` : '/recipes'}>{t('common.cancel')}</a>
			<button class="btn btn-primary flex-[2] md:flex-none" disabled={saving || !form.name.trim()}>{saving ? t('common.saving') : t('common.save')}</button>
		</div>
	</div>
</form>
