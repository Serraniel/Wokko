<script lang="ts">
	import { ArrowLeft, CookingPot, Pencil, Share2, Star, Trash2 } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, ApiError, errorText } from '#lib/api/client.ts';
	import type { Recipe } from '#lib/api/types.gen.ts';
	import { cached, forget, remember } from '#lib/cache.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import RecipeView from '#lib/components/RecipeView.svelte';
	import ShareSheet from '#lib/components/ShareSheet.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
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
	<a href="/recipes" class="btn btn-soft btn-icon absolute top-3 left-3 z-10 shadow-card md:hidden" aria-label={t('common.back')}><ArrowLeft size={20} /></a>
	<ErrorNote text={error} />
	{#if recipe}
		<div class={recipe.photo_url ? '' : 'pt-12 md:pt-0'}>
			<RecipeView {recipe}>
				{#snippet actions()}
					<button class="btn btn-primary flex-1 sm:flex-none" onclick={() => (cookOpen = true)}><CookingPot size={18} />{t('recipe.cooked_it')}</button>
					{#if recipe?.can_edit}<a class="btn btn-soft" href="/recipes/{recipe.id}/edit" aria-label={t('common.edit')}><Pencil size={18} /><span class="hidden sm:inline">{t('common.edit')}</span></a>{/if}
					{#if recipe?.can_manage}
						<button class="btn btn-soft" onclick={() => (shareOpen = true)} aria-label={t('share.button')}><Share2 size={18} /><span class="hidden sm:inline">{t('share.button')}</span></button>
						<button class="btn btn-ghost btn-icon text-danger" aria-label={t('common.delete')} onclick={remove}><Trash2 size={18} /></button>
					{/if}
				{/snippet}
			</RecipeView>
		</div>
		{#if recipe.can_manage}
			<ShareSheet bind:open={shareOpen} {recipe} onchange={(r) => (recipe = remember(`recipe:${r.id}`, r))} />
		{/if}
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
