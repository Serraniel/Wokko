<script lang="ts">
	import { Search } from '@lucide/svelte';
	import { page } from '$app/state';
	import { api, ApiError, errorText, query } from '#lib/api/client.ts';
	import type { Page, RecipeSummary } from '#lib/api/types.gen.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import PublicShell from '#lib/components/PublicShell.svelte';
	import RecipeCard from '#lib/components/RecipeCard.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let q = $state('');
	let items = $state<RecipeSummary[] | null>(null);
	let missing = $state(false);
	let error = $state('');
	let timer: ReturnType<typeof setTimeout>;

	async function load() {
		try {
			items = (await api.get<Page<RecipeSummary>>(`/public/collections/${page.params.token}${query({ q, limit: 100 })}`)).items;
		} catch (e) {
			if (e instanceof ApiError && e.status === 404) missing = true;
			else error = errorText(e);
		}
	}
	load();
</script>

<svelte:head>
	<title>{t('shared.title')} · Wokko</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<PublicShell>
	<div class="mx-auto max-w-[96rem] space-y-3 p-4">
		{#if missing}
			<div class="flex flex-col items-center gap-3 p-10 text-center">
				<Mascot />
				<h1 class="text-xl font-semibold">{t('share.link_gone')}</h1>
				<p class="text-muted">{t('share.link_gone_hint')}</p>
			</div>
		{:else}
			<h1 class="text-xl font-semibold">{t('shared.title')}</h1>
			<div class="relative">
				<Search size={16} class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" />
				<input class="input !pl-9" type="search" bind:value={q} oninput={() => { clearTimeout(timer); timer = setTimeout(load, 80); }} placeholder={t('recipes.search')} aria-label={t('recipes.search')} />
			</div>
			<ErrorNote text={error} />
			{#if items}
				<div class="grid grid-cols-2 gap-x-3 gap-y-5 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
					{#each items as r (r.id)}<RecipeCard recipe={r} href="/shared/{page.params.token}/{r.id}" />{/each}
				</div>
			{/if}
		{/if}
	</div>
</PublicShell>
