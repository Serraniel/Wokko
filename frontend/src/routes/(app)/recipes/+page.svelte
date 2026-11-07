<script lang="ts">
	import { BookOpen, Plus, Search, Trash2, X } from '@lucide/svelte';
	import { api, errorText, query } from '#lib/api/client.ts';
	import type { Keyword, Page, RecipeSummary } from '#lib/api/types.gen.ts';
	import { cached, remember } from '#lib/cache.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import RecipeCard from '#lib/components/RecipeCard.svelte';
	import RecipeGridSkeleton from '#lib/components/RecipeGridSkeleton.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	const PAGE = 48;
	let q = $state('');
	let keyword = $state(0);
	let sort = $state('updated');
	let page = $state<Page<RecipeSummary> | null>(cached('recipes:||updated') ?? null);
	let keywords = $state<Keyword[]>(cached('keywords:top') ?? []);
	let error = $state('');
	let loadingMore = $state(false);
	let timer: ReturnType<typeof setTimeout>;
	let request = 0;

	async function load() {
		const key = `recipes:${q}|${keyword || ''}|${sort}`;
		const mine = ++request;
		const known = cached<Page<RecipeSummary>>(key);
		if (known) page = known;
		try {
			const res = await api.get<Page<RecipeSummary>>(`/recipes${query({ q, keyword, sort, limit: PAGE })}`);
			if (mine !== request) return;
			page = remember(key, res);
			error = '';
		} catch (e) {
			if (mine === request) error = errorText(e);
		}
	}

	async function more() {
		if (!page) return;
		loadingMore = true;
		try {
			const res = await api.get<Page<RecipeSummary>>(`/recipes${query({ q, keyword, sort, limit: PAGE, offset: page.items.length })}`);
			page = { total: res.total, items: [...page.items, ...res.items] };
		} catch (e) {
			error = errorText(e);
		} finally {
			loadingMore = false;
		}
	}

	load();
	api.get<Page<Keyword>>('/keywords?limit=12')
		.then((p) => (keywords = remember('keywords:top', p.items)))
		.catch(() => {}); // The filter row is optional. The recipes matter here.

	function typed() {
		clearTimeout(timer);
		timer = setTimeout(load, 60);
	}

	function toggleKeyword(id: number) {
		keyword = keyword === id ? 0 : id;
		load();
	}

	// More recipes load before you reach the end of the list.
	let sentinel = $state<HTMLElement>();
	$effect(() => {
		if (!sentinel) return;
		const watcher = new IntersectionObserver((entries) => {
			if (entries[0].isIntersecting && !loadingMore && page && page.items.length < page.total) more();
		}, { rootMargin: '800px' });
		watcher.observe(sentinel);
		return () => watcher.disconnect();
	});

	const filtered = $derived(q.trim() !== '' || keyword !== 0);
</script>

<svelte:head><title>{t('nav.recipes')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-[96rem] px-4 pb-4 md:px-6">
	<!-- Search and filters stay in reach while scrolling. -->
	<div class="sticky top-0 z-20 -mx-4 space-y-2 bg-bg/90 px-4 pt-3 pb-2 backdrop-blur-lg md:-mx-6 md:px-6 md:pt-4">
		<header class="flex items-center justify-between gap-2">
			<h1 class="flex items-baseline gap-2 text-xl font-semibold">{t('nav.recipes')}{#if page && !filtered && page.total}<span class="text-sm font-normal text-muted tabular-nums">{page.total}</span>{/if}</h1>
			<div class="flex items-center gap-1">
				<a class="btn btn-ghost btn-icon text-muted" href="/trash" aria-label={t('trash.title')}><Trash2 size={18} /></a>
				<select class="input !min-h-9 !w-auto !py-0 !text-sm" bind:value={sort} onchange={load} aria-label={t('recipes.sort')}>
					<option value="updated">{t('recipes.sort_updated')}</option>
					<option value="name">{t('recipes.sort_name')}</option>
					<option value="created">{t('recipes.sort_created')}</option>
				</select>
			</div>
		</header>

		<div class="relative">
			<Search size={16} class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" />
			<input class="input !pr-10 !pl-9" type="search" bind:value={q} oninput={typed} placeholder={t('recipes.search')} aria-label={t('recipes.search')} />
			{#if q}
				<button class="absolute top-1/2 right-0.5 flex h-9 w-9 -translate-y-1/2 items-center justify-center text-muted" aria-label={t('common.clear')} onclick={() => { q = ''; load(); }}><X size={16} /></button>
			{/if}
		</div>

		{#if keywords.length}
			<div class="no-scrollbar -mx-4 flex gap-1.5 overflow-x-auto px-4 md:-mx-6 md:px-6">
				{#each keywords as k (k.id)}
					<button class="chip {keyword === k.id ? 'chip-active' : 'hover:bg-surface-2'}" aria-pressed={keyword === k.id} onclick={() => toggleKeyword(k.id)}>{k.name}</button>
				{/each}
			</div>
		{/if}
	</div>

	<ErrorNote text={error} />

	{#if !page}
		{#if !error}<div class="mt-2"><RecipeGridSkeleton /></div>{/if}
	{:else if page.items.length === 0}
		<div class="flex flex-col items-center gap-3 py-14 text-center">
			<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-surface-2 text-muted"><BookOpen size={22} /></div>
			{#if filtered}
				<p class="font-semibold">{t('recipes.no_results')}</p>
			{:else}
				<p class="text-lg font-semibold">{t('recipes.empty_title')}</p>
				<p class="max-w-xs text-muted">{t('recipes.empty_text')}</p>
				<a class="btn btn-primary" href="/recipes/new"><Plus size={18} />{t('recipes.new')}</a>
			{/if}
		</div>
	{:else}
		<div class="mt-2 grid grid-cols-2 gap-x-3 gap-y-5 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
			{#each page.items as r (r.id)}<RecipeCard recipe={r} href="/recipes/{r.id}" />{/each}
		</div>
		{#if page.items.length < page.total}
			<div bind:this={sentinel} class="pt-4">
				<button class="btn btn-soft w-full" disabled={loadingMore} onclick={more}>{loadingMore ? t('common.loading') : t('recipes.load_more')}</button>
			</div>
		{/if}
	{/if}
</div>
