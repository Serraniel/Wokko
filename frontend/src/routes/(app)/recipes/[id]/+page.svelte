<script lang="ts">
	import { ArrowLeft, BookMarked, CalendarPlus, CookingPot, Download, History, MoreHorizontal, Pencil, Printer, Share2, ShoppingCart, Star, Trash2 } from '@lucide/svelte';
	import { tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, ApiError, errorText } from '#lib/api/client.ts';
	import type { Recipe, RecipeBook, RecipeNutrition, RecipeVersion } from '#lib/api/types.gen.ts';
	import { cached, forget, remember } from '#lib/cache.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import RecipeView from '#lib/components/RecipeView.svelte';
	import ShareSheet from '#lib/components/ShareSheet.svelte';
	import ExportSheet from '#lib/components/ExportSheet.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { formatDate, formatNumber, t } from '#lib/i18n/index.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	const id = $derived(page.params.id);
	let recipe = $state<Recipe | null>(null);
	let missing = $state(false);
	let error = $state('');
	let shareOpen = $state(false);
	let cookOpen = $state(false);
	let rating = $state(0);
	let comment = $state('');
	let logging = $state(false);
	let moreOpen = $state(false);
	let exportOpen = $state(false);
	let historyOpen = $state(false);
	let booksOpen = $state(false);
	let versions = $state<RecipeVersion[]>([]);
	let books = $state<RecipeBook[]>([]);
	let inBooks = $state<number[]>([]);
	let nutrition = $state<RecipeNutrition | null>(null);

	$effect(() => {
		if (page.url.searchParams.get('cooked')) cookOpen = true;
	});

	$effect(() => {
		const current = id;
		nutrition = null;
		api.get<RecipeNutrition>(`/recipes/${current}/nutrition`)
			.then((n) => {
				if (current === id) nutrition = n;
			})
			.catch(() => {}); // Nutrition is extra. The recipe matters here.
	});

	async function openHistory() {
		moreOpen = false;
		historyOpen = true;
		try {
			versions = await api.get<RecipeVersion[]>(`/recipes/${id}/versions`);
		} catch (e) {
			toasts.error(errorText(e));
		}
	}

	async function restore(v: RecipeVersion) {
		try {
			recipe = remember(`recipe:${id}`, await api.post<Recipe>(`/recipes/${id}/versions/${v.id}/restore`));
			forget('recipes');
			historyOpen = false;
			toasts.show(t('history.restored'));
		} catch (e) {
			toasts.error(errorText(e));
		}
	}

	async function openBooks() {
		moreOpen = false;
		booksOpen = true;
		try {
			[books, inBooks] = await Promise.all([api.get<RecipeBook[]>('/books'), api.get<number[]>(`/recipes/${id}/books`)]);
		} catch (e) {
			toasts.error(errorText(e));
		}
	}

	async function toggleBook(b: RecipeBook) {
		const had = inBooks.includes(b.id);
		inBooks = had ? inBooks.filter((x) => x !== b.id) : [...inBooks, b.id];
		try {
			if (had) await api.del(`/books/${b.id}/recipes/${id}`);
			else await api.put(`/books/${b.id}/recipes/${id}`, {});
			forget('books');
		} catch (e) {
			toasts.error(errorText(e));
		}
	}

	async function addToShopping() {
		moreOpen = false;
		if (!recipe) return;
		try {
			await api.post('/shopping/recipes', { recipe_id: recipe.id, servings: recipe.servings });
			forget('shopping');
			toasts.show(t('shopping.added'));
		} catch (e) {
			toasts.error(errorText(e));
		}
	}

	const planLink = $derived(recipe ? `/plan?recipe=${recipe.id}&name=${encodeURIComponent(recipe.name)}&servings=${recipe.servings}` : '/plan');

	$effect(() => {
		const current = id;
		recipe = cached<Recipe>(`recipe:${current}`) ?? null;
		missing = false;
		api.get<Recipe>(`/recipes/${current}`)
			.then((r) => {
				if (current === id) recipe = remember(`recipe:${current}`, r);
			})
			.catch((e) => {
				if (e instanceof ApiError && e.status === 404) {
					recipe = null;
					missing = true;
				} else if (!recipe) error = errorText(e);
			});
	});

	async function remove() {
		if (!recipe) return;
		const r = recipe;
		try {
			await api.del(`/recipes/${r.id}`);
			forget('recipe');
			toasts.show(t('recipe.moved_to_trash'), {
				action: {
					label: t('common.undo'),
					run: async () => {
						try {
							await api.post(`/recipes/${r.id}/restore`);
							forget('recipe');
							await goto(`/recipes/${r.id}`);
						} catch (e) {
							toasts.error(errorText(e));
						}
					}
				}
			});
			await goto('/recipes');
		} catch (e) {
			toasts.error(errorText(e));
		}
	}

	async function logCooked(e: SubmitEvent) {
		e.preventDefault();
		if (!recipe) return;
		logging = true;
		try {
			await api.post(`/recipes/${recipe.id}/cooklog`, { rating: rating || null, comment, servings: recipe.servings });
			recipe = remember(`recipe:${recipe.id}`, await api.get<Recipe>(`/recipes/${recipe.id}`));
			forget('recipes');
			cookOpen = false;
			rating = 0;
			comment = '';
			toasts.show(t('recipe.cook_logged'));
		} catch (err) {
			toasts.error(errorText(err));
		} finally {
			logging = false;
		}
	}
</script>

<svelte:head><title>{recipe?.name ?? t('nav.recipes')} · Wokko</title></svelte:head>

<div class="relative">
	<a href="/recipes" class="print-hide btn btn-soft btn-icon absolute top-3 left-3 z-10 shadow-card md:hidden" aria-label={t('common.back')}><ArrowLeft size={20} /></a>
	<ErrorNote text={error} />
	{#if recipe}
		<div class={recipe.photo_url ? '' : 'pt-12 md:pt-0'}>
			<RecipeView {recipe} cookHref="/recipes/{recipe.id}/cook" insightsUrl="/recipes/{recipe.id}/insights">
				{#snippet actions()}
										<button class="btn btn-soft !px-3 sm:!px-3.5" aria-label={t('recipe.cooked_it')} onclick={() => (cookOpen = true)}><CookingPot size={17} /><span class="hidden sm:inline">{t('recipe.cooked_it')}</span></button>
					{#if recipe?.can_edit}<a class="btn btn-soft" href="/recipes/{recipe.id}/edit" aria-label={t('common.edit')}><Pencil size={18} /><span class="hidden sm:inline">{t('common.edit')}</span></a>{/if}
					{#if recipe?.can_manage}
						<button class="btn btn-soft" onclick={() => (shareOpen = true)} aria-label={t('share.button')}><Share2 size={18} /><span class="hidden sm:inline">{t('share.button')}</span></button>
					{/if}
					<button class="btn btn-soft btn-icon" aria-label={t('recipe.more')} onclick={() => (moreOpen = true)}><MoreHorizontal size={18} /></button>
				{/snippet}
				{#snippet footer()}
					{#if nutrition && nutrition.counted > 0}
						<section class="card p-4" aria-labelledby="nutrition-title">
							<h2 id="nutrition-title" class="mb-2 font-semibold">{t('recipe.nutrition')}</h2>
							<table class="w-full text-sm">
								<thead><tr class="text-left text-xs text-muted"><th class="font-medium"><span class="sr-only">{t('recipe.nutrition')}</span></th><th class="text-right font-medium">{t('recipe.per_serving')}</th><th class="text-right font-medium">{t('recipe.total')}</th></tr></thead>
								<tbody class="divide-y divide-line">
									{#each nutrition.nutrients as n (n.nutrient.id)}
										<tr><td class="py-1.5">{n.nutrient.name}</td><td class="py-1.5 text-right font-medium tabular-nums">{formatNumber(Math.round(n.per_serving * 10) / 10)} {n.nutrient.unit}</td><td class="py-1.5 text-right text-muted tabular-nums">{formatNumber(Math.round(n.total))} {n.nutrient.unit}</td></tr>
									{/each}
								</tbody>
							</table>
							<p class="hint">{nutrition.counted < nutrition.total ? t('recipe.nutrition_partial', { counted: nutrition.counted, total: nutrition.total, names: nutrition.missing.join(', ') }) : t('recipe.nutrition_estimate')}</p>
						</section>
					{/if}
				{/snippet}
			</RecipeView>
		</div>
		{#if recipe.can_manage}
			<ShareSheet bind:open={shareOpen} {recipe} onchange={(r) => (recipe = remember(`recipe:${r.id}`, r))} />
		{/if}
		<Sheet bind:open={moreOpen} title={recipe.name}>
			<div class="divide-y divide-line">
				<button class="flex min-h-12 w-full items-center gap-3 text-left font-medium" onclick={async () => { moreOpen = false; await tick(); window.print(); }}><Printer size={18} class="text-muted" />{t('recipe.print')}</button>
				<a class="flex min-h-12 items-center gap-3 font-medium" href={planLink}><CalendarPlus size={18} class="text-muted" />{t('recipe.add_to_plan')}</a>
				<button class="flex min-h-12 w-full items-center gap-3 text-left font-medium" onclick={addToShopping}><ShoppingCart size={18} class="text-muted" />{t('recipe.add_to_shopping')}</button>
				<button class="flex min-h-12 w-full items-center gap-3 text-left font-medium" onclick={openBooks}><BookMarked size={18} class="text-muted" />{t('books.add_to')}</button>
				<button class="flex min-h-12 w-full items-center gap-3 text-left font-medium" onclick={async () => { moreOpen = false; await tick(); exportOpen = true; }}><Download size={18} class="text-muted" />{t('export.action')}</button>
				{#if recipe.can_edit}<button class="flex min-h-12 w-full items-center gap-3 text-left font-medium" onclick={openHistory}><History size={18} class="text-muted" />{t('recipe.history')}</button>{/if}
				{#if recipe.can_manage}<button class="flex min-h-12 w-full items-center gap-3 text-left font-medium text-danger" onclick={() => { moreOpen = false; remove(); }}><Trash2 size={18} />{t('common.delete')}</button>{/if}
			</div>
		</Sheet>
		<ExportSheet bind:open={exportOpen} base="/api/v1/recipes/{recipe.id}/export" title={t('export.title')} />
		<Sheet bind:open={historyOpen} title={t('history.title')}>
			{#if versions.length === 0}
				<p class="py-4 text-sm text-muted">{t('history.empty')}</p>
			{:else}
				<ul class="divide-y divide-line">
					{#each versions as v (v.id)}
						<li class="flex min-h-12 items-center justify-between gap-2">
							<span class="text-sm"><span class="font-medium">{t('history.version', { version: v.version })}</span><span class="block text-xs text-muted">{formatDate(v.created_at, 'datetime')}</span></span>
							<button class="btn btn-soft" onclick={() => restore(v)}>{t('history.restore')}</button>
						</li>
					{/each}
				</ul>
			{/if}
		</Sheet>
		<Sheet bind:open={booksOpen} title={t('books.add_to')}>
			{#if books.length === 0}
				<p class="py-4 text-sm text-muted">{t('books.none')} <a class="underline" href="/books">{t('books.title')}</a></p>
			{:else}
				<ul class="divide-y divide-line">
					{#each books as b (b.id)}
						<li><label class="flex min-h-12 items-center gap-3 font-medium"><input type="checkbox" class="h-4 w-4 accent-[var(--brand)]" checked={inBooks.includes(b.id)} onchange={() => toggleBook(b)} />{b.name}</label></li>
					{/each}
				</ul>
			{/if}
		</Sheet>
		<Sheet bind:open={cookOpen} title={t('recipe.cooked_it')}>
			<form class="space-y-4" onsubmit={logCooked}>
				<div>
					<p class="label">{t('recipe.your_rating')}</p>
					<div class="flex gap-1" role="group" aria-label={t('recipe.your_rating')}>
						{#each [1, 2, 3, 4, 5] as n}
							<button type="button" class="btn btn-ghost btn-icon" aria-label={t('recipe.stars', { count: n })} aria-pressed={rating === n} onclick={() => (rating = rating === n ? 0 : n)}>
								<Star size={26} class={n <= rating ? 'fill-flame text-flame' : 'text-muted'} />
							</button>
						{/each}
					</div>
				</div>
				<div>
					<label class="label" for="comment">{t('recipe.comment')}</label>
					<textarea id="comment" class="input" rows="2" bind:value={comment}></textarea>
				</div>
				<button class="btn btn-primary w-full" disabled={logging}>{t('common.save')}</button>
			</form>
		</Sheet>
	{:else if missing}
		<div class="flex flex-col items-center gap-3 p-10 text-center">
			<Mascot />
			<h1 class="text-xl font-bold">{t('notfound.title')}</h1>
			<a class="btn btn-primary" href="/recipes">{t('notfound.home')}</a>
		</div>
	{:else if !error}
		<div class="mx-auto max-w-3xl space-y-4 p-4 pt-16 md:pt-4" aria-hidden="true">
			<div class="skeleton h-8 w-2/3"></div>
			<div class="skeleton h-4 w-1/2"></div>
			<div class="skeleton h-40 w-full"></div>
		</div>
	{/if}
</div>
