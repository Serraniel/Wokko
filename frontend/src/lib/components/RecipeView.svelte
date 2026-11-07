<script lang="ts">
	import { Clock, ExternalLink, Hourglass, Minus, Plus, Star, Users } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { Ingredient, Recipe, UnitRef } from '#lib/api/types.gen.ts';
	import { combineIngredients, roundAmount } from '#lib/format.ts';
	import { formatDate, formatMinutes, formatNumber, t } from '#lib/i18n/index.svelte.ts';

	let { recipe, actions, footer }: { recipe: Recipe; actions?: Snippet; footer?: Snippet } = $props();

	let servings = $state(0);
	$effect(() => {
		servings = recipe.servings;
	});
	const scale = $derived(recipe.servings > 0 ? servings / recipe.servings : 1);
	const overview = $derived(combineIngredients(recipe.steps));

	function amountText(amount: number | null | undefined): string {
		return amount === null || amount === undefined ? '' : formatNumber(roundAmount(amount * scale));
	}

	function unitText(unit: UnitRef | null | undefined, amount: number | null | undefined): string {
		if (!unit) return '';
		if (unit.abbreviation) return unit.abbreviation;
		return amount !== null && amount !== undefined && amount * scale !== 1 && unit.plural ? unit.plural : unit.name;
	}

	function foodText(name: string, plural: string, amount: number | null | undefined): string {
		return amount !== null && amount !== undefined && amount * scale > 1 && plural ? plural : name;
	}

	function line(i: Ingredient) {
		return [amountText(i.amount), unitText(i.unit, i.amount), foodText(i.food.name, i.food.plural, i.amount)].filter(Boolean).join(' ');
	}
</script>

<article class="mx-auto w-full max-w-3xl">
	{#if recipe.photo_url}
		<img src={recipe.photo_url} alt="" class="aspect-[16/10] w-full bg-surface-2 object-cover sm:mt-4 sm:rounded-2xl" />
	{/if}
	<div class="space-y-5 p-4">
		<header class="space-y-3">
			<h1 class="text-2xl leading-tight font-bold sm:text-3xl">{recipe.name}</h1>
			{#if recipe.description}<p class="whitespace-pre-line text-muted">{recipe.description}</p>{/if}
			<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted">
				{#if recipe.work_minutes}<span class="inline-flex items-center gap-1.5"><Clock size={16} />{t('recipe.work_short', { time: formatMinutes(recipe.work_minutes) })}</span>{/if}
				{#if recipe.wait_minutes}<span class="inline-flex items-center gap-1.5"><Hourglass size={16} />{t('recipe.wait_short', { time: formatMinutes(recipe.wait_minutes) })}</span>{/if}
				{#if recipe.rating}<span class="inline-flex items-center gap-1.5"><Star size={16} class="fill-flame text-flame" />{recipe.rating.toFixed(1)}</span>{/if}
				{#if recipe.last_cooked}<span>{t('recipe.last_cooked', { date: formatDate(recipe.last_cooked) })}</span>{/if}
			</div>
			{#if recipe.keywords.length}
				<div class="flex flex-wrap gap-1.5">
					{#each recipe.keywords as k (k.id)}<span class="chip">{k.name}</span>{/each}
				</div>
			{/if}
			{#if actions}<div class="flex flex-wrap gap-2">{@render actions()}</div>{/if}
		</header>

		{#if overview.length}
			<section class="card p-4">
				<div class="mb-3 flex items-center justify-between gap-3">
					<h2 class="text-lg font-semibold">{t('recipe.ingredients')}</h2>
					<div class="flex items-center gap-1" role="group" aria-label={t('recipe.servings')}>
						<button class="btn btn-soft btn-icon !min-h-9 !w-9" onclick={() => (servings = Math.max(1, servings - 1))} aria-label={t('recipe.fewer_servings')}><Minus size={16} /></button>
						<span class="inline-flex min-w-14 items-center justify-center gap-1 text-sm font-semibold tabular-nums" aria-live="polite"><Users size={15} />{servings}</span>
						<button class="btn btn-soft btn-icon !min-h-9 !w-9" onclick={() => servings++} aria-label={t('recipe.more_servings')}><Plus size={16} /></button>
					</div>
				</div>
				<ul class="divide-y divide-line">
					{#each overview as i (i.key)}
						<li class="flex items-baseline gap-3 py-2">
							<span class="w-24 shrink-0 text-right font-semibold tabular-nums">{amountText(i.amount)} {unitText(i.unit, i.amount)}</span>
							<span class="min-w-0">{foodText(i.name, i.plural, i.amount)}{#if i.notes.length}<span class="text-sm text-muted">, {i.notes.join('; ')}</span>{/if}</span>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		{#if recipe.steps.length}
			<section class="space-y-4">
				<h2 class="text-lg font-semibold">{t('recipe.steps')}</h2>
				<ol class="space-y-4">
					{#each recipe.steps as step, n (step.id)}
						<li class="flex gap-3">
							<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-soft text-sm font-bold text-brand-soft-ink">{n + 1}</span>
							<div class="min-w-0 flex-1 space-y-2 pt-0.5">
								{#if step.name}<h3 class="font-semibold">{step.name}</h3>{/if}
								{#if step.ingredients.length}
									<ul class="flex flex-wrap gap-1.5">
										{#each step.ingredients as ing (ing.id)}<li class="rounded-lg bg-surface-2 px-2 py-0.5 text-sm">{line(ing)}{#if ing.note}<span class="text-muted">, {ing.note}</span>{/if}</li>{/each}
									</ul>
								{/if}
								{#if step.instruction}<p class="whitespace-pre-line">{step.instruction}</p>{/if}
								{#if step.minutes}<p class="inline-flex items-center gap-1 text-sm text-muted"><Clock size={14} />{formatMinutes(step.minutes)}</p>{/if}
							</div>
						</li>
					{/each}
				</ol>
			</section>
		{/if}

		{#if recipe.source_url}
			<a href={recipe.source_url} rel="noopener noreferrer nofollow" target="_blank" class="inline-flex items-center gap-1.5 text-sm font-medium text-brand underline-offset-2 hover:underline"><ExternalLink size={15} />{t('recipe.source')}</a>
		{/if}
		{#if footer}{@render footer()}{/if}
	</div>
</article>
