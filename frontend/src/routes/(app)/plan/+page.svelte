<script lang="ts">
	import { ChevronLeft, ChevronRight, Plus, ShoppingCart, Trash2 } from '@lucide/svelte';
	import { page } from '$app/state';
	import { api, errorText, query } from '#lib/api/client.ts';
	import type { MealPlan, MealPlanInput, MealType, Page, RecipeSummary } from '#lib/api/types.gen.ts';
	import { cached, forget, remember } from '#lib/cache.ts';
	import Combobox, { type Option } from '#lib/components/Combobox.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { addDays, isoDate, startOfWeek } from '#lib/format.ts';
	import { currentLocale, t } from '#lib/i18n/index.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let week = $state(startOfWeek(new Date()));
	const days = $derived(Array.from({ length: 7 }, (_, i) => addDays(week, i)));
	const today = isoDate(new Date());
	const key = $derived(`plans:${isoDate(week)}`);

	let plans = $state<MealPlan[] | null>(null);
	let types = $state<MealType[]>(cached('meal-types') ?? []);
	let error = $state('');
	let open = $state(false);
	let editing = $state<MealPlan | null>(null);
	let formError = $state('');
	let saving = $state(false);
	let form = $state({ recipeName: '', recipeId: 0, title: '', date: today, toDate: '', mealType: 0, servings: '4', note: '', shopping: false });

	async function load() {
		const asked = key;
		plans = cached<MealPlan[]>(asked) ?? plans;
		try {
			const res = await api.get<MealPlan[]>(`/meal-plans${query({ from: isoDate(week), to: isoDate(addDays(week, 6)) })}`);
			if (asked === key) plans = remember(asked, res);
			error = '';
		} catch (e) {
			error = errorText(e);
		}
	}

	$effect(() => {
		void week;
		load();
	});
	api.get<MealType[]>('/meal-types').then((r) => (types = remember('meal-types', r))).catch(() => {});

	// Arriving from a recipe's "Add to meal plan".
	const arriving = page.url.searchParams.get('recipe');
	if (arriving) startNew(today, { id: Number(arriving), name: page.url.searchParams.get('name') ?? '', servings: page.url.searchParams.get('servings') ?? '4' });

	function startNew(date: string, recipe?: { id: number; name: string; servings: string }) {
		editing = null;
		form = { recipeName: recipe?.name ?? '', recipeId: recipe?.id ?? 0, title: '', date, toDate: '', mealType: types[2]?.id ?? types[0]?.id ?? 0, servings: recipe?.servings ?? '4', note: '', shopping: false };
		formError = '';
		open = true;
	}

	function startEdit(p: MealPlan) {
		editing = p;
		form = { recipeName: p.recipe?.name ?? '', recipeId: p.recipe?.id ?? 0, title: p.title, date: p.from_date, toDate: p.to_date === p.from_date ? '' : p.to_date, mealType: p.meal_type.id, servings: String(p.servings), note: p.note, shopping: false };
		formError = '';
		open = true;
	}

	async function searchRecipes(q: string): Promise<Option[]> {
		const res = await api.get<Page<RecipeSummary>>(`/recipes${query({ q, limit: 8 })}`);
		return res.items.map((r) => ({ id: r.id, label: r.name }));
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		formError = '';
		const body: MealPlanInput = { title: form.title, note: form.note, servings: Number(form.servings.replace(',', '.')) || 1, from_date: form.date, to_date: form.toDate, meal_type_id: form.mealType || types[0]?.id, recipe_id: form.recipeId, add_to_shopping: form.shopping };
		try {
			if (editing) await api.put(`/meal-plans/${editing.id}`, body);
			else await api.post('/meal-plans', body);
			open = false;
			forget('plans:');
			if (form.date < isoDate(week) || form.date > isoDate(addDays(week, 6))) week = startOfWeek(new Date(`${form.date}T12:00:00`));
			await load();
		} catch (err) {
			formError = errorText(err);
		} finally {
			saving = false;
		}
	}

	async function remove() {
		if (!editing) return;
		try {
			await api.del(`/meal-plans/${editing.id}`);
			open = false;
			forget('plans:');
			await load();
		} catch (err) {
			formError = errorText(err);
		}
	}

	async function weekToShopping() {
		const withRecipes = (plans ?? []).filter((p) => p.recipe);
		try {
			for (const p of withRecipes) await api.post('/shopping/recipes', { recipe_id: p.recipe!.id, servings: p.servings, meal_plan_id: p.id });
			forget('shopping');
			toasts.show(t('plan.week_added', { count: withRecipes.length }));
		} catch (err) {
			toasts.error(errorText(err));
		}
	}

	const dayLabel = (d: Date) => new Intl.DateTimeFormat(currentLocale(), { weekday: 'short', day: 'numeric', month: 'short' }).format(d);
	const onDay = (d: Date) => (plans ?? []).filter((p) => p.from_date <= isoDate(d) && p.to_date >= isoDate(d)).sort((a, b) => a.meal_type.position - b.meal_type.position);
	const planName = (p: MealPlan) => p.recipe?.name || p.title || (p.recipe_hidden ? t('plan.hidden') : '');
	const rangeLabel = $derived(`${new Intl.DateTimeFormat(currentLocale(), { day: 'numeric', month: 'short' }).format(week)} – ${new Intl.DateTimeFormat(currentLocale(), { day: 'numeric', month: 'short' }).format(addDays(week, 6))}`);
</script>

<svelte:head><title>{t('plan.title')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-[96rem] space-y-3 p-4 md:px-6">
	<header class="flex flex-wrap items-center justify-between gap-2">
		<h1 class="text-xl font-semibold">{t('plan.title')}</h1>
		<div class="flex items-center gap-1">
			<button class="btn btn-ghost btn-icon" aria-label={t('plan.prev')} onclick={() => (week = addDays(week, -7))}><ChevronLeft size={18} /></button>
			<button class="btn btn-soft !px-2.5 text-sm tabular-nums" onclick={() => (week = startOfWeek(new Date()))} title={t('plan.this_week')}>{rangeLabel}</button>
			<button class="btn btn-ghost btn-icon" aria-label={t('plan.next')} onclick={() => (week = addDays(week, 7))}><ChevronRight size={18} /></button>
		</div>
	</header>
	<ErrorNote text={error} />

	<div class="grid gap-2 lg:grid-cols-7">
		{#each days as d (isoDate(d))}
			{@const iso = isoDate(d)}
			<section class="card flex flex-col p-3 lg:min-h-64 {iso === today ? 'ring-1 ring-brand' : ''}" aria-label={dayLabel(d)}>
				<div class="mb-1 flex items-center justify-between">
					<h2 class="text-sm font-semibold {iso === today ? 'text-brand' : ''}">{dayLabel(d)}</h2>
					<button class="btn btn-ghost btn-icon !min-h-8 !w-8 text-muted" aria-label={t('plan.add')} onclick={() => startNew(iso)}><Plus size={16} /></button>
				</div>
				<ul class="space-y-1">
					{#each onDay(d) as p (p.id)}
						<li>
							<button class="w-full rounded-lg bg-surface-2 px-2.5 py-1.5 text-left hover:bg-line" onclick={() => startEdit(p)}>
								<span class="block text-xs text-muted">{p.meal_type.name}{p.servings ? ` · ${p.servings}` : ''}</span>
								<span class="block text-sm font-medium">{planName(p)}</span>
								{#if p.note}<span class="block text-xs text-muted">{p.note}</span>{/if}
							</button>
						</li>
					{:else}
						{#if plans}<li class="text-xs text-muted">{t('plan.empty_day')}</li>{/if}
					{/each}
				</ul>
			</section>
		{/each}
	</div>

	{#if plans?.some((p) => p.recipe)}
		<button class="btn btn-soft" onclick={weekToShopping}><ShoppingCart size={16} />{t('plan.add_week_to_shopping')}</button>
	{/if}
</div>

<Sheet bind:open title={editing ? t('plan.edit') : t('plan.add')}>
	<form class="space-y-3" onsubmit={save}>
		<ErrorNote text={formError} />
		{#if editing?.recipe_hidden}
			<p class="rounded-lg bg-surface-2 px-3 py-2 text-sm text-muted">{t('plan.hidden')}</p>
		{:else}
			<div>
				<label class="label" for="plan-recipe">{t('plan.recipe')}</label>
				<Combobox id="plan-recipe" label={t('plan.recipe')} placeholder={t('plan.recipe_placeholder')} allowCreate={false} bind:value={form.recipeName} search={searchRecipes} ontype={() => (form.recipeId = 0)} onpick={(o) => (form.recipeId = o?.id ?? 0)} />
			</div>
		{/if}
		<div>
			<label class="label" for="plan-title">{t('plan.title_field')}</label>
			<input id="plan-title" class="input" bind:value={form.title} placeholder={t('plan.title_placeholder')} />
		</div>
		<div class="grid grid-cols-2 gap-3">
			<div>
				<label class="label" for="plan-date">{t('plan.date')}</label>
				<input id="plan-date" class="input" type="date" bind:value={form.date} required />
			</div>
			<div>
				<label class="label" for="plan-to">{t('plan.to_date')}</label>
				<input id="plan-to" class="input" type="date" bind:value={form.toDate} min={form.date} />
			</div>
			<div>
				<label class="label" for="plan-type">{t('plan.meal_type')}</label>
				<select id="plan-type" class="input" bind:value={form.mealType}>
					{#each types as mt (mt.id)}<option value={mt.id}>{mt.name}</option>{/each}
				</select>
			</div>
			<div>
				<label class="label" for="plan-servings">{t('plan.servings')}</label>
				<input id="plan-servings" class="input" inputmode="decimal" bind:value={form.servings} />
			</div>
		</div>
		<div>
			<label class="label" for="plan-note">{t('plan.note')}</label>
			<input id="plan-note" class="input" bind:value={form.note} />
		</div>
		{#if !editing && form.recipeId}
			<label class="flex min-h-9 items-center gap-2 text-sm"><input type="checkbox" class="h-4 w-4 accent-[var(--brand)]" bind:checked={form.shopping} />{t('plan.add_to_shopping')}</label>
		{/if}
		<div class="flex gap-2">
			{#if editing}<button type="button" class="btn btn-danger btn-icon" aria-label={t('common.delete')} onclick={remove}><Trash2 size={17} /></button>{/if}
			<button class="btn btn-primary flex-1" disabled={saving}>{saving ? t('common.saving') : t('common.save')}</button>
		</div>
	</form>
</Sheet>
