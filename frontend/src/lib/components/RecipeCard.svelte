<script lang="ts">
	import { Clock, Star, Users } from '@lucide/svelte';
	import type { RecipeSummary } from '#lib/api/types.gen.ts';
	import { formatMinutes, t } from '#lib/i18n/index.svelte.ts';
	import Logo from './Logo.svelte';

	let { recipe, href }: { recipe: RecipeSummary; href: string } = $props();
	const minutes = $derived(recipe.work_minutes + recipe.wait_minutes);
</script>

<a {href} class="card group flex flex-col overflow-hidden transition-shadow hover:shadow-card">
	<div class="relative aspect-[4/3] overflow-hidden bg-surface-2">
		{#if recipe.thumb_url}
			<img src={recipe.thumb_url} alt="" loading="lazy" decoding="async" class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105" />
		{:else}
			<div class="flex h-full items-center justify-center opacity-40 grayscale"><Logo size={48} /></div>
		{/if}
	</div>
	<div class="flex flex-1 flex-col gap-1 p-3">
		<h3 class="line-clamp-2 leading-snug font-semibold">{recipe.name}</h3>
		<div class="mt-auto flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted">
			{#if minutes > 0}<span class="inline-flex items-center gap-1"><Clock size={14} />{formatMinutes(minutes)}</span>{/if}
			<span class="inline-flex items-center gap-1"><Users size={14} />{t('recipe.servings_count', { count: recipe.servings })}</span>
			{#if recipe.rating}<span class="inline-flex items-center gap-1"><Star size={14} class="fill-flame text-flame" />{recipe.rating.toFixed(1)}</span>{/if}
		</div>
	</div>
</a>
