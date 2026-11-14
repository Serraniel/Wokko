<script lang="ts">
	import { ChefHat, Clock, Download, ExternalLink, Hourglass, Minus, Plus, Printer, Star, Users } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { Ingredient, Recipe, UnitRef } from '#lib/api/types.gen.ts';
	import { asFraction, combineIngredients, roundAmount } from '#lib/format.ts';
	import { formatDate, formatMinutes, formatNumber, t } from '#lib/i18n/index.svelte.ts';
	import { renderMarkdown } from '#lib/markdown.ts';
	import { prefs } from '#lib/prefs.svelte.ts';
	import ExportSheet from './ExportSheet.svelte';
	import RecipeInsights from './RecipeInsights.svelte';

	// exportBase, on pages without an account, offers the recipe as a file to take elsewhere.
	// insightsUrl is where Wokko's kitchen knowledge about this recipe is read from.
	let { recipe, actions, footer, cookHref, exportBase, insightsUrl }: { recipe: Recipe; actions?: Snippet; footer?: Snippet; cookHref?: string; exportBase?: string; insightsUrl?: string } = $props();
	let exportOpen = $state(false);

	let servings = $state(0);
	$effect(() => {
		servings = recipe.servings;
	});
	const scale = $derived(recipe.servings > 0 ? servings / recipe.servings : 1);
	const overview = $derived(combineIngredients(recipe.steps));
	// When one step holds everything, as after an import, repeating the list there adds nothing.
	const perStep = $derived(recipe.steps.length === 1 || recipe.steps.filter((s) => s.ingredients.length).length > 1);

	function amountText(amount: number | null | undefined): string {
		if (amount === null || amount === undefined) return '';
		const scaled = roundAmount(amount * scale);
		return (prefs.amounts === 'fraction' ? asFraction(scaled) : null) ?? formatNumber(scaled);
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

<article class="mx-auto w-full max-w-6xl">
	<!-- On a phone the photo runs edge to edge. With room to spare it sits beside the title. -->
	<div class="lg:grid lg:items-center lg:gap-8 lg:p-6 {recipe.photo_url ? 'lg:grid-cols-[minmax(0,1fr)_26rem]' : ''}">
		{#if recipe.photo_url}
			<img src={recipe.photo_url} alt="" class="recipe-photo aspect-[16/10] w-full bg-surface-2 object-cover lg:order-2 lg:rounded-2xl" />
		{/if}
		<header class="space-y-3 p-4 lg:p-0">
			<h1 class="text-2xl leading-tight font-semibold lg:text-4xl">{recipe.name}</h1>
			{#if recipe.description}<p class="whitespace-pre-line text-muted">{recipe.description}</p>{/if}
			<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted">
				{#if recipe.work_minutes}<span class="inline-flex items-center gap-1.5"><Clock size={15} />{t('recipe.work_short', { time: formatMinutes(recipe.work_minutes) })}</span>{/if}
				{#if recipe.wait_minutes}<span class="inline-flex items-center gap-1.5"><Hourglass size={15} />{t('recipe.wait_short', { time: formatMinutes(recipe.wait_minutes) })}</span>{/if}
				{#if recipe.rating}<span class="inline-flex items-center gap-1.5"><Star size={15} class="fill-flame text-flame" />{recipe.rating.toFixed(1)}</span>{/if}
				{#if recipe.last_cooked}<span>{t('recipe.last_cooked', { date: formatDate(recipe.last_cooked) })}</span>{/if}
			</div>
			{#if recipe.keywords.length}
				<div class="flex flex-wrap gap-1.5">
					{#each recipe.keywords as k (k.id)}<span class="chip !min-h-7">{k.name}</span>{/each}
				</div>
			{/if}
			<div class="print-hide flex flex-wrap gap-2">
				{#if cookHref && recipe.steps.length}<a class="btn btn-primary flex-1 sm:flex-none" href="{cookHref}?servings={servings}"><ChefHat size={17} />{t('recipe.cook')}</a>{/if}
				{#if actions}{@render actions()}{:else}<button class="btn btn-soft" onclick={() => window.print()}><Printer size={17} />{t('recipe.print')}</button>{#if exportBase}<button class="btn btn-soft" onclick={() => (exportOpen = true)}><Download size={17} />{t('export.action')}</button>{/if}{/if}
			</div>
		</header>
	</div>

	<!-- Ingredients stay beside the steps while scrolling, for cooking at a laptop or tablet. -->
	<div class="space-y-6 px-4 pb-4 lg:grid lg:grid-cols-[22rem_minmax(0,1fr)] lg:items-start lg:gap-8 lg:space-y-0 lg:px-6 lg:pb-6">
		{#if overview.length}
			<section class="card p-4 lg:sticky lg:top-4">
				<div class="mb-2 flex items-center justify-between gap-3">
					<h2 class="font-semibold">{t('recipe.ingredients')}</h2>
					<div class="flex items-center gap-1" role="group" aria-label={t('recipe.servings')}>
						<button class="print-hide btn btn-soft btn-icon !min-h-9 !w-9" onclick={() => (servings = Math.max(1, servings - 1))} aria-label={t('recipe.fewer_servings')}><Minus size={15} /></button>
						<span class="inline-flex min-w-11 items-center justify-center gap-1 text-sm font-semibold tabular-nums" aria-live="polite"><Users size={14} />{servings}</span>
						<button class="print-hide btn btn-soft btn-icon !min-h-9 !w-9" onclick={() => servings++} aria-label={t('recipe.more_servings')}><Plus size={15} /></button>
					</div>
				</div>
				<ul class="divide-y divide-line text-sm">
					{#each overview as i (i.key)}
						<li class="grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 py-2">
							<span class="text-right font-semibold tabular-nums">{amountText(i.amount)} {unitText(i.unit, i.amount)}</span>
							<span>{foodText(i.name, i.plural, i.amount)}{#if i.notes.length}<span class="text-muted">, {i.notes.join('; ')}</span>{/if}</span>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<div class="space-y-6 {overview.length ? '' : 'lg:col-span-2'}">
			{#if recipe.steps.length}
				<section class="space-y-3">
					<h2 class="font-semibold">{t('recipe.steps')}</h2>
					<ol class="space-y-5">
						{#each recipe.steps as step, n (step.id)}
							<li class="flex gap-3">
								<span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-2 text-xs font-semibold tabular-nums">{n + 1}</span>
								<div class="min-w-0 flex-1 space-y-1.5">
									{#if step.name}<h3 class="font-semibold">{step.name}</h3>{/if}
									{#if step.instruction}<p class="whitespace-pre-line leading-relaxed">{@html renderMarkdown(step.instruction)}</p>{/if}
									{#if perStep && step.ingredients.length}
										<ul class="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted">
											{#each step.ingredients as ing (ing.id)}<li>{line(ing)}{#if ing.note}, {ing.note}{/if}</li>{/each}
										</ul>
									{/if}
									{#if step.minutes}<p class="inline-flex items-center gap-1 text-xs text-muted"><Clock size={12} />{formatMinutes(step.minutes)}</p>{/if}
								</div>
							</li>
						{/each}
					</ol>
				</section>
			{/if}

			{#if insightsUrl}<RecipeInsights url={insightsUrl} version={recipe.version} />{/if}

			{#if recipe.source_url}
				<a href={recipe.source_url} rel="noopener noreferrer nofollow" target="_blank" class="print-hide inline-flex min-h-9 items-center gap-1.5 text-sm font-medium text-muted underline-offset-2 hover:text-ink hover:underline"><ExternalLink size={14} />{t('recipe.source')}</a>
				<p class="print-only text-xs">{recipe.source_url}</p>
			{/if}
			{#if footer}{@render footer()}{/if}
		</div>
	</div>
</article>

{#if exportBase}<ExportSheet bind:open={exportOpen} base={exportBase} title={t('export.title')} />{/if}
