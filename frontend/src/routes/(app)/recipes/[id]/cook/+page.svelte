<script lang="ts">
	import { ChevronLeft, ChevronRight, ListChecks, Timer, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import type { Ingredient, Recipe } from '#lib/api/types.gen.ts';
	import { cached, remember } from '#lib/cache.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { asFraction, combineIngredients, roundAmount } from '#lib/format.ts';
	import { formatMinutes, formatNumber, t } from '#lib/i18n/index.svelte.ts';
	import { renderMarkdown } from '#lib/markdown.ts';
	import { prefs } from '#lib/prefs.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	const id = page.params.id;
	let recipe = $state<Recipe | null>(cached(`recipe:${id}`) ?? null);
	let error = $state('');
	let index = $state(0);
	let showAll = $state(false);
	let left = $state(0);
	let timer: ReturnType<typeof setInterval> | undefined;
	let lock: { release: () => Promise<void> } | null = null;

	api.get<Recipe>(`/recipes/${id}`)
		.then((r) => (recipe = remember(`recipe:${id}`, r)))
		.catch((e) => {
			if (!recipe) error = errorText(e);
		});

	const steps = $derived(recipe?.steps ?? []);
	const step = $derived(steps[index]);
	const last = $derived(index >= steps.length - 1);
	const overview = $derived(recipe ? combineIngredients(recipe.steps) : []);

	const amount = (n: number | null | undefined) => {
		if (n === null || n === undefined) return '';
		const r = roundAmount(n);
		return (prefs.amounts === 'fraction' ? asFraction(r) : null) ?? formatNumber(r);
	};
	const line = (i: Ingredient) => [amount(i.amount), i.unit?.abbreviation || i.unit?.name || '', i.food.name].filter(Boolean).join(' ');

	async function keepAwake() {
		try {
			lock = await (navigator as Navigator & { wakeLock: { request(t: 'screen'): Promise<{ release: () => Promise<void> }> } }).wakeLock.request('screen');
		} catch {
			lock = null; // Not offered here, for example over plain HTTP. Cooking still works.
		}
	}

	function keys(e: KeyboardEvent) {
		if (e.key === 'ArrowRight' && !last) move(1);
		if (e.key === 'ArrowLeft' && index > 0) move(-1);
		if (e.key === 'Escape') exit();
	}

	onMount(() => {
		keepAwake();
		const again = () => {
			if (document.visibilityState === 'visible') keepAwake();
		};
		document.addEventListener('visibilitychange', again);
		return () => {
			document.removeEventListener('visibilitychange', again);
			clearInterval(timer);
			lock?.release().catch(() => {});
		};
	});

	function move(by: number) {
		index += by;
		showAll = false;
	}

	function startTimer(minutes: number) {
		clearInterval(timer);
		left = minutes * 60;
		timer = setInterval(() => {
			left -= 1;
			if (left <= 0) {
				clearInterval(timer);
				left = 0;
				toasts.show(t('cook.timer_done'));
				navigator.vibrate?.([300, 150, 300]);
			}
		}, 1000);
	}

	function stopTimer() {
		clearInterval(timer);
		left = 0;
	}

	const clock = $derived(`${Math.floor(left / 60)}:${String(left % 60).padStart(2, '0')}`);
	const exit = () => goto(`/recipes/${id}`);
</script>

<svelte:head><title>{recipe?.name ?? ''} · Wokko</title></svelte:head>
<svelte:window onkeydown={keys} />

<!-- Covers the whole screen: big text and big buttons, for floury hands at arm's length. -->
<div class="fixed inset-0 z-40 flex flex-col bg-bg pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)]">
	<header class="flex items-center gap-2 border-b border-line px-2 py-1">
		<button class="btn btn-ghost btn-icon" aria-label={t('cook.exit')} onclick={exit}><X size={20} /></button>
		<div class="min-w-0 flex-1">
			<p class="truncate text-sm font-semibold">{recipe?.name ?? ''}</p>
			{#if steps.length}<p class="text-xs text-muted tabular-nums">{t('cook.step', { current: index + 1, total: steps.length })}</p>{/if}
		</div>
		<button class="btn btn-ghost btn-icon {showAll ? 'bg-surface-2' : ''}" aria-pressed={showAll} aria-label={t('cook.ingredients')} onclick={() => (showAll = !showAll)}><ListChecks size={20} /></button>
	</header>
	<div class="h-1 bg-surface-2"><div class="h-full bg-brand" style="width: {steps.length ? ((index + 1) / steps.length) * 100 : 0}%"></div></div>

	<main class="min-h-0 flex-1 overflow-y-auto p-5">
		<ErrorNote text={error} />
		<div class="mx-auto max-w-2xl space-y-5">
			{#if showAll}
				<h1 class="text-xl font-semibold">{t('cook.ingredients')}</h1>
				<ul class="divide-y divide-line text-lg">
					{#each overview as i (i.key)}
						<li class="py-2.5"><span class="font-semibold tabular-nums">{amount(i.amount)} {i.unit?.abbreviation || i.unit?.name || ''}</span> {i.name}{#if i.notes.length}<span class="text-base text-muted">, {i.notes.join('; ')}</span>{/if}</li>
					{/each}
				</ul>
			{:else if step}
				{#if step.name}<h1 class="text-2xl font-semibold">{step.name}</h1>{/if}
				{#if step.ingredients.length}
					<ul class="flex flex-wrap gap-2">
						{#each step.ingredients as i (i.id)}<li class="rounded-lg bg-surface-2 px-3 py-1.5 font-medium">{line(i)}{#if i.note}<span class="font-normal text-muted">, {i.note}</span>{/if}</li>{/each}
					</ul>
				{/if}
				{#if step.instruction}<p class="whitespace-pre-line text-xl leading-relaxed sm:text-2xl">{@html renderMarkdown(step.instruction)}</p>{/if}
				{#if step.minutes}
					{#if left > 0}
						<button class="btn btn-soft !min-h-14 w-full !text-xl tabular-nums" aria-label={t('cook.timer_stop')} onclick={stopTimer}><Timer size={22} />{clock}</button>
					{:else}
						<button class="btn btn-soft w-full" onclick={() => startTimer(step.minutes)}><Timer size={18} />{t('cook.timer_start', { time: formatMinutes(step.minutes) })}</button>
					{/if}
				{/if}
			{/if}
		</div>
	</main>

	<footer class="grid grid-cols-[auto_1fr] gap-2 border-t border-line p-3">
		<button class="btn btn-soft !min-h-14 !w-16" disabled={index === 0} aria-label={t('cook.prev')} onclick={() => move(-1)}><ChevronLeft size={24} /></button>
		{#if last}
			<a class="btn btn-primary !min-h-14 !text-lg" href="/recipes/{id}?cooked=1">{t('cook.done')}</a>
		{:else}
			<button class="btn btn-primary !min-h-14 !text-lg" onclick={() => move(1)}>{t('cook.next')}<ChevronRight size={22} /></button>
		{/if}
	</footer>
</div>
