<script lang="ts">
	import { Clock, Star, Users } from '@lucide/svelte';
	import type { RecipeSummary } from '#lib/api/types.gen.ts';
	import { formatMinutes, t } from '#lib/i18n/index.svelte.ts';
	import { warmRecipe } from '#lib/prefetch.ts';

	let { recipe, href }: { recipe: RecipeSummary; href: string } = $props();
	const minutes = $derived(recipe.work_minutes + recipe.wait_minutes);
	const hue = $derived([...recipe.name].reduce((h, c) => (h * 31 + c.codePointAt(0)!) % 360, 7));
	const initial = $derived([...recipe.name.trim()][0]?.toUpperCase() ?? '');
	const warm = () => {
		if (href.startsWith('/recipes/')) warmRecipe(recipe.id);
	};
</script>

<a {href} class="recipe-card group block" onpointerenter={warm} onpointerdown={warm} onfocus={warm}>
	<div class="relative aspect-[4/3] overflow-hidden rounded-xl bg-surface-2">
		{#if recipe.thumb_url}
			<img src={recipe.thumb_url} alt="" loading="lazy" decoding="async" class="h-full w-full object-cover" />
		{:else}
			<div class="tile flex h-full items-center justify-center text-4xl font-semibold" style="--hue: {hue}" aria-hidden="true">{initial}</div>
		{/if}
	</div>
	<div class="px-0.5 pt-2">
		<h2 class="line-clamp-2 text-sm leading-snug font-semibold group-hover:text-brand">{recipe.name}</h2>
		<div class="mt-0.5 flex flex-wrap items-center gap-x-2.5 text-xs text-muted">
			{#if minutes > 0}<span class="inline-flex items-center gap-1"><Clock size={12} />{formatMinutes(minutes)}</span>{/if}
			<span class="inline-flex items-center gap-1"><Users size={12} />{recipe.servings}<span class="sr-only">{t('recipe.servings_count', { count: recipe.servings })}</span></span>
			{#if recipe.rating}<span class="inline-flex items-center gap-1"><Star size={12} class="fill-flame text-flame" />{recipe.rating.toFixed(1)}</span>{/if}
		</div>
		{#if recipe.keywords.length}
			<ul class="mt-1 flex h-5 flex-wrap gap-1 overflow-hidden" aria-label={t('editor.tags')}>
				{#each recipe.keywords.slice(0, 3) as k (k.id)}<li class="max-w-full truncate rounded bg-surface-2 px-1.5 text-[0.6875rem] leading-5 text-muted">{k.name}</li>{/each}
			</ul>
		{/if}
	</div>
</a>
