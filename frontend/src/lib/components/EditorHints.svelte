<script lang="ts">
	import { Info, Lightbulb, ShieldAlert, TriangleAlert } from '@lucide/svelte';
	import { api, query } from '#lib/api/client.ts';
	import type { Result } from '#lib/api/knowledge.gen.ts';
	import type { RecipeInput } from '#lib/api/types.gen.ts';
	import { currentLocale, t } from '#lib/i18n/index.svelte.ts';

	// draft is the recipe as it is being edited. Hints come a moment after typing stops.
	let { draft }: { draft: RecipeInput | null } = $props();
	let result = $state<Result | null>(null);
	let timer: ReturnType<typeof setTimeout>;
	let request = 0;

	$effect(() => {
		const body = draft ? JSON.stringify(draft) : '';
		clearTimeout(timer);
		if (!body || !draft?.steps.length) {
			result = null;
			return;
		}
		timer = setTimeout(async () => {
			const mine = ++request;
			try {
				const r = await api.post<Result>(`/recipes/insights${query({ lang: currentLocale() })}`, JSON.parse(body));
				if (mine === request) result = r;
			} catch {
				// Hints are a help, not a requirement. Editing goes on without them.
			}
		}, 1000);
		return () => clearTimeout(timer);
	});

	const notes = $derived([...(result?.safety ?? []), ...(result?.problems ?? [])]);
	const sure = $derived(result?.allergens.filter((a) => !a.maybe) ?? []);
</script>

{#if result && (notes.length || sure.length || result.diet.vegetarian !== null)}
	<section class="card space-y-3 p-4" aria-labelledby="hints-title" aria-live="polite">
		<h2 id="hints-title" class="flex items-center gap-2 font-semibold"><Lightbulb size={17} class="text-muted" />{t('editor.hints')}</h2>
		{#if notes.length}
			<ul class="space-y-2">
				{#each notes as f (f.id)}
					<li class="rounded-xl p-3 text-sm {f.severity === 'warning' ? 'bg-warn-soft' : 'bg-surface-2'}">
						<p class="flex items-start gap-2 font-medium">
							{#if f.kind === 'safety'}<ShieldAlert size={16} class="mt-0.5 shrink-0" />{:else if f.severity === 'warning'}<TriangleAlert size={16} class="mt-0.5 shrink-0 text-warn" />{:else}<Info size={16} class="mt-0.5 shrink-0 text-muted" />{/if}
							<span>{f.title}{#if f.steps.length}<span class="font-normal text-muted"> · {t('insights.steps', { steps: f.steps.join(', ') })}</span>{/if}</span>
						</p>
						<p class="mt-1 pl-6 text-muted">{f.text}</p>
						{#each f.tips as tip, i (i)}<p class="mt-1 pl-6">{tip}</p>{/each}
					</li>
				{/each}
			</ul>
		{:else}
			<p class="text-sm text-muted">{t('editor.hints_none')}</p>
		{/if}
		<p class="text-xs text-muted">
			{#if result.diet.vegan}{t('insights.vegan')} · {:else if result.diet.vegetarian}{t('insights.vegetarian')} · {:else if result.diet.vegetarian === false}{t('insights.not_vegetarian')} · {/if}
			{#if sure.length}{t('insights.allergens_from')} {sure.map((a) => a.name).join(', ')}{:else}{t('editor.hints_no_allergens')}{/if}
		</p>
	</section>
{/if}
