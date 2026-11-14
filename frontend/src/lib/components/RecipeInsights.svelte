<script lang="ts">
	import { Info, Leaf, Lightbulb, ShieldAlert, TriangleAlert } from '@lucide/svelte';
	import { api } from '#lib/api/client.ts';
	import type { Finding, Result } from '#lib/api/knowledge.gen.ts';
	import { currentLocale, t } from '#lib/i18n/index.svelte.ts';

	// url is the insights address without /api/v1. version makes it load again after an edit.
	let { url, version = 0 }: { url: string; version?: number } = $props();
	let result = $state<Result | null>(null);

	$effect(() => {
		const address = `${url}?lang=${encodeURIComponent(currentLocale())}&v=${version}`;
		result = null;
		// Insights are extra. If they can't load, the recipe is still complete without them.
		api.get<Result>(address).then((r) => (result = r)).catch(() => {});
	});

	const sure = $derived(result?.allergens.filter((a) => !a.maybe) ?? []);
	const maybe = $derived(result?.allergens.filter((a) => a.maybe) ?? []);
	const notes = $derived([...(result?.safety ?? []), ...(result?.problems ?? [])]);
	const knowledge = $derived([...(result?.sauces ?? []), ...(result?.techniques ?? [])]);
	const anything = $derived(!!result && (result.allergens.length > 0 || notes.length > 0 || knowledge.length > 0 || result.diet.vegetarian !== null));

	function stepsText(f: Finding): string {
		return f.steps.length ? t('insights.steps', { steps: f.steps.join(', ') }) : '';
	}
</script>

{#if anything && result}
	<section class="print-hide card space-y-4 p-4" aria-labelledby="insights-title">
		<div>
			<h2 id="insights-title" class="flex items-center gap-2 font-semibold"><Lightbulb size={17} class="text-muted" />{t('insights.title')}</h2>
			<p class="hint !mt-0.5">{t('insights.hint')}</p>
		</div>

		{#if result.diet.vegetarian !== null || result.allergens.length}
			<div class="space-y-2">
				<div class="flex flex-wrap gap-2">
					{#if result.diet.vegan}
						<span class="chip bg-ok-soft text-ok"><Leaf size={14} />{t('insights.vegan')}</span>
					{:else if result.diet.vegetarian}
						<span class="chip bg-ok-soft text-ok"><Leaf size={14} />{t('insights.vegetarian')}</span>
					{:else if result.diet.vegetarian === false}
						<span class="chip bg-surface-2" title={result.diet.because.join(', ')}>{t('insights.not_vegetarian')}</span>
					{/if}
					{#each sure as a (a.id)}
						<span class="chip bg-warn-soft text-warn" title={a.foods.join(', ')}>{a.name}</span>
					{/each}
				</div>
				{#if sure.length}
					<p class="text-xs text-muted">{t('insights.allergens_from')} {sure.map((a) => `${a.name}: ${a.foods.join(', ')}`).join(' · ')}</p>
				{/if}
				{#if maybe.length}
					<p class="text-xs text-muted">{t('insights.may_contain')} {maybe.map((a) => `${a.name} (${a.foods.join(', ')})`).join(' · ')}</p>
				{/if}
				{#if result.unclassified.length}
					<p class="text-xs text-muted">{t('insights.unknown', { foods: result.unclassified.join(', ') })}</p>
				{/if}
			</div>
		{/if}

		{#if notes.length}
			<ul class="space-y-2">
				{#each notes as f (f.id)}
					<li class="rounded-xl p-3 text-sm {f.severity === 'warning' ? 'bg-warn-soft' : 'bg-surface-2'}">
						<p class="flex items-start gap-2 font-medium">
							{#if f.kind === 'safety'}<ShieldAlert size={16} class="mt-0.5 shrink-0 {f.severity === 'warning' ? 'text-warn' : 'text-muted'}" />{:else if f.severity === 'warning'}<TriangleAlert size={16} class="mt-0.5 shrink-0 text-warn" />{:else}<Info size={16} class="mt-0.5 shrink-0 text-muted" />{/if}
							<span>{f.title}{#if f.steps.length}<span class="font-normal text-muted"> · {stepsText(f)}</span>{/if}</span>
						</p>
						<p class="mt-1 pl-6 text-muted">{f.text}</p>
						{#each f.tips as tip, i (i)}<p class="mt-1 pl-6">{tip}</p>{/each}
					</li>
				{/each}
			</ul>
		{/if}

		{#if knowledge.length}
			<div class="divide-y divide-line">
				{#each knowledge as f (f.id)}
					<details class="group py-1">
						<summary class="flex min-h-10 cursor-pointer list-none items-center gap-2 text-sm">
							<span class="rounded-md bg-surface-2 px-1.5 py-0.5 text-xs text-muted">{t(f.kind === 'sauce' ? 'insights.sauce' : 'insights.technique')}</span>
							<span class="font-medium">{f.title}</span>
							{#if f.steps.length}<span class="text-xs text-muted">{stepsText(f)}</span>{/if}
						</summary>
						<div class="space-y-1.5 pb-2 pl-1 text-sm">
							<p class="text-muted">{f.text}</p>
							{#each f.tips as tip, i (i)}<p class="flex gap-2"><Lightbulb size={14} class="mt-1 shrink-0 text-muted" /><span>{tip}</span></p>{/each}
							{#if f.sources.length}
								<p class="text-xs text-muted">
									{t('insights.source')}
									{#each f.sources as s, i (i)}{i ? '; ' : ''}{#if s.url}<a class="underline underline-offset-2 hover:text-ink" href={s.url} target="_blank" rel="noopener noreferrer">{s.title}</a>{:else}{s.title}{/if}{/each}
								</p>
							{/if}
						</div>
					</details>
				{/each}
			</div>
		{/if}
	</section>
{/if}
