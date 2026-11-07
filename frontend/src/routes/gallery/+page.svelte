<script lang="ts">
	import { Search } from '@lucide/svelte';
	import { api, errorText, query } from '#lib/api/client.ts';
	import type { Page, RecipeSummary } from '#lib/api/types.gen.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import PublicShell from '#lib/components/PublicShell.svelte';
	import RecipeCard from '#lib/components/RecipeCard.svelte';
	import RecipeGridSkeleton from '#lib/components/RecipeGridSkeleton.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let q = $state('');
	let items = $state<RecipeSummary[] | null>(null);
	let error = $state('');
	let timer: ReturnType<typeof setTimeout>;

	async function load() {
		try {
			const p = await api.get<Page<RecipeSummary>>(`/public/recipes${query({ q, limit: 60 })}`);
			items = p.items;
		} catch (e) {
			error = errorText(e);
		}
	}
	load();
</script>

<svelte:head><title>{t('gallery.title')} · Wokko</title></svelte:head>

<PublicShell>
	<div class="mx-auto max-w-6xl space-y-4 p-4">
		<h1 class="text-2xl font-bold">{t('gallery.title')}</h1>
		<div class="relative">
			<Search size={18} class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" />
			<input class="input !pl-10" type="search" bind:value={q} placeholder={t('recipes.search')} aria-label={t('recipes.search')} oninput={() => { clearTimeout(timer); timer = setTimeout(load, 150); }} />
		</div>
		<ErrorNote text={error} />
		{#if !items}
			<RecipeGridSkeleton />
		{:else if items.length === 0}
			<p class="py-12 text-center text-muted">{t('gallery.empty')}</p>
		{:else}
			<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
				{#each items as r (r.id)}<RecipeCard recipe={r} href="/gallery/{r.id}" />{/each}
			</div>
		{/if}
	</div>
</PublicShell>
