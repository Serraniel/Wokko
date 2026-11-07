<script lang="ts">
	import { ChevronLeft, ChevronRight, ListChecks, Minus, Plus, Timer, Users, X } from '@lucide/svelte';
	import { onMount, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Ingredient, Recipe } from '#lib/api/types.gen.ts';
	import { durationsIn } from '#lib/cooking.ts';
	import { asFraction, combineIngredients, roundAmount } from '#lib/format.ts';
	import { formatMinutes, formatNumber, t } from '#lib/i18n/index.svelte.ts';
	import { renderMarkdown } from '#lib/markdown.ts';
	import { prefs } from '#lib/prefs.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let { recipe, exitHref, doneHref, servings: startServings }: { recipe: Recipe; exitHref: string; doneHref: string; servings?: number } = $props();

	interface RunningTimer {
		id: number;
		label: string;
		endsAt: number;
	}
	interface Saved {
		index: number;
		ticked: string[];
		servings: number;
		timers: RunningTimer[];
	}

	// Fixed for the life of this page: each recipe has its own route.
	const storageKey = untrack(() => `wokko-cook:${exitHref}`);

	function load(): Partial<Saved> {
		try {
			return JSON.parse(localStorage.getItem(storageKey) ?? '{}');
		} catch {
			return {};
		}
	}

	// Coming back to a recipe picks up where you were, even after the phone restarted the page.
	const saved = load();
	let index = $state(untrack(() => Math.min(saved.index ?? 0, Math.max(recipe.steps.length - 1, 0))));
	let ticked = $state<string[]>(saved.ticked ?? []);
	let servings = $state(untrack(() => startServings || saved.servings || recipe.servings || 1));
	let timers = $state<RunningTimer[]>(saved.timers ?? []);
	let showAll = $state(false);
	let now = $state(Date.now());
	let lock: { release: () => Promise<void> } | null = null;
	let swipeFrom = 0;

	$effect(() => {
		const value = JSON.stringify({ index, ticked, servings, timers });
		try {
			localStorage.setItem(storageKey, value);
		} catch {
			// Only a convenience. Cooking still works.
		}
	});

	const steps = $derived(recipe.steps);
	const step = $derived(steps[index]);
	const last = $derived(index >= steps.length - 1);
	const overview = $derived(combineIngredients(recipe.steps));
	const scale = $derived(recipe.servings > 0 ? servings / recipe.servings : 1);
	const found = $derived(step ? durationsIn(step.instruction, step.minutes) : []);

	const amount = (n: number | null | undefined) => {
		if (n === null || n === undefined) return '';
		const r = roundAmount(n * scale);
		return (prefs.amounts === 'fraction' ? asFraction(r) : null) ?? formatNumber(r);
	};
	const line = (i: Ingredient) => [amount(i.amount), i.unit?.abbreviation || i.unit?.name || '', i.food.name].filter(Boolean).join(' ');

	function tick(key: string) {
		ticked = ticked.includes(key) ? ticked.filter((k) => k !== key) : [...ticked, key];
	}

	async function keepAwake() {
		try {
			lock = await (navigator as Navigator & { wakeLock: { request(t: 'screen'): Promise<{ release: () => Promise<void> }> } }).wakeLock.request('screen');
		} catch {
			lock = null; // Not offered here, for example over plain HTTP. Cooking still works.
		}
	}

	function beep() {
		try {
			const audio = new AudioContext();
			const tone = audio.createOscillator();
			tone.connect(audio.destination);
			tone.frequency.value = 880;
			tone.start();
			tone.stop(audio.currentTime + 0.6);
		} catch {
			// The toast and vibration still say it.
		}
		navigator.vibrate?.([300, 150, 300, 150, 300]);
	}

	onMount(() => {
		keepAwake();
		const again = () => {
			if (document.visibilityState === 'visible') keepAwake();
		};
		document.addEventListener('visibilitychange', again);
		const clock = setInterval(() => {
			now = Date.now();
			const finished = timers.filter((x) => x.endsAt <= now);
			if (finished.length) {
				for (const x of finished) toasts.show(t('cook.timer_done_named', { name: x.label }));
				timers = timers.filter((x) => x.endsAt > now);
				beep();
			}
		}, 1000);
		return () => {
			document.removeEventListener('visibilitychange', again);
			clearInterval(clock);
			lock?.release().catch(() => {});
		};
	});

	function move(by: number) {
		index = Math.min(Math.max(index + by, 0), Math.max(steps.length - 1, 0));
		showAll = false;
	}

	function startTimer(minutes: number) {
		const n = index + 1;
		timers = [...timers, { id: Date.now(), label: t('cook.timer_label', { step: n, time: formatMinutes(minutes) }), endsAt: Date.now() + minutes * 60000 }];
	}

	function clock(x: RunningTimer) {
		const left = Math.max(Math.ceil((x.endsAt - now) / 1000), 0);
		const h = Math.floor(left / 3600);
		const m = Math.floor((left % 3600) / 60);
		return `${h ? `${h}:${String(m).padStart(2, '0')}` : m}:${String(left % 60).padStart(2, '0')}`;
	}

	function finish() {
		try {
			localStorage.removeItem(storageKey);
		} catch {
			// Nothing to clean up, then.
		}
		goto(doneHref);
	}

	function keys(e: KeyboardEvent) {
		if (e.key === 'ArrowRight' && !last) move(1);
		if (e.key === 'ArrowLeft' && index > 0) move(-1);
		if (e.key === 'Escape') goto(exitHref);
	}

	function swipeEnd(e: TouchEvent) {
		const dist = e.changedTouches[0].clientX - swipeFrom;
		if (Math.abs(dist) < 70) return;
		if (dist < 0 && !last) move(1);
		if (dist > 0 && index > 0) move(-1);
	}
</script>

<svelte:head><title>{recipe.name} · Wokko</title></svelte:head>
<svelte:window onkeydown={keys} />

<!-- Covers the whole screen: big text and big buttons, for floury hands at arm's length. -->
<div class="fixed inset-0 z-40 flex flex-col bg-bg pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)]">
	<header class="flex items-center gap-1 border-b border-line px-2 py-1">
		<a class="btn btn-ghost btn-icon" href={exitHref} aria-label={t('cook.exit')}><X size={20} /></a>
		<div class="min-w-0 flex-1">
			<p class="truncate text-sm font-semibold">{recipe.name}</p>
			{#if steps.length}<p class="text-xs text-muted tabular-nums">{t('cook.step', { current: index + 1, total: steps.length })}</p>{/if}
		</div>
		<button class="btn btn-ghost btn-icon {showAll ? 'bg-surface-2' : ''}" aria-pressed={showAll} aria-label={t('cook.ingredients')} onclick={() => (showAll = !showAll)}><ListChecks size={20} /></button>
	</header>
	<div class="h-1 bg-surface-2"><div class="h-full bg-brand" style="width: {steps.length ? ((index + 1) / steps.length) * 100 : 0}%"></div></div>

	{#if timers.length}
		<ul class="flex gap-1.5 overflow-x-auto border-b border-line px-3 py-1.5" aria-label={t('cook.timers')}>
			{#each timers as x (x.id)}
				<li class="flex shrink-0 items-center gap-1 rounded-full bg-brand-soft py-0.5 pr-0.5 pl-2.5 text-sm font-semibold text-brand-soft-ink">
					<Timer size={14} /><span class="tabular-nums">{clock(x)}</span>
					<span class="max-w-32 truncate text-xs font-normal">{x.label}</span>
					<button class="flex h-7 w-7 items-center justify-center" aria-label={t('cook.timer_stop')} onclick={() => (timers = timers.filter((y) => y.id !== x.id))}><X size={14} /></button>
				</li>
			{/each}
		</ul>
	{/if}

	<div role="presentation" class="min-h-0 flex-1 overflow-y-auto p-5" ontouchstart={(e) => (swipeFrom = e.touches[0].clientX)} ontouchend={swipeEnd}>
		<div class="mx-auto max-w-2xl space-y-5">
			{#if showAll}
				<div class="flex items-center justify-between gap-2">
					<h1 class="text-xl font-semibold">{t('cook.ingredients')}</h1>
					<div class="flex items-center gap-1" role="group" aria-label={t('recipe.servings')}>
						<button class="btn btn-soft btn-icon" aria-label={t('recipe.fewer_servings')} onclick={() => (servings = Math.max(1, servings - 1))}><Minus size={16} /></button>
						<span class="inline-flex min-w-11 items-center justify-center gap-1 font-semibold tabular-nums" aria-live="polite"><Users size={15} />{servings}</span>
						<button class="btn btn-soft btn-icon" aria-label={t('recipe.more_servings')} onclick={() => servings++}><Plus size={16} /></button>
					</div>
				</div>
				<ul class="divide-y divide-line text-lg">
					{#each overview as i (i.key)}
						<li>
							<label class="flex min-h-12 items-center gap-3 py-2 {ticked.includes(i.key) ? 'text-muted line-through' : ''}">
								<input type="checkbox" class="h-5 w-5 shrink-0 accent-[var(--brand)]" checked={ticked.includes(i.key)} onchange={() => tick(i.key)} />
								<span><span class="font-semibold tabular-nums">{amount(i.amount)} {i.unit?.abbreviation || i.unit?.name || ''}</span> {i.name}{#if i.notes.length}<span class="text-base text-muted">, {i.notes.join('; ')}</span>{/if}</span>
							</label>
						</li>
					{/each}
				</ul>
			{:else if step}
				{#if step.name}<h1 class="text-2xl font-semibold">{step.name}</h1>{/if}
				{#if step.ingredients.length}
					<ul class="flex flex-wrap gap-2">
						{#each step.ingredients as i (i.id)}
							{@const key = `step:${i.id}`}
							<li>
								<button class="min-h-10 rounded-lg px-3 py-1.5 text-left font-medium {ticked.includes(key) ? 'bg-surface-2 text-muted line-through' : 'bg-surface-2'}" aria-pressed={ticked.includes(key)} onclick={() => tick(key)}>{line(i)}{#if i.note}<span class="font-normal text-muted">, {i.note}</span>{/if}</button>
							</li>
						{/each}
					</ul>
				{/if}
				{#if step.instruction}<p class="whitespace-pre-line text-xl leading-relaxed sm:text-2xl">{@html renderMarkdown(step.instruction)}</p>{/if}
				{#if found.length}
					<div class="flex flex-wrap gap-2">
						{#each found as minutes (minutes)}
							<button class="btn btn-soft" onclick={() => startTimer(minutes)}><Timer size={18} />{t('cook.timer_start', { time: formatMinutes(minutes) })}</button>
						{/each}
					</div>
				{/if}
			{/if}
		</div>
	</div>

	<footer class="grid grid-cols-[auto_1fr] gap-2 border-t border-line p-3">
		<button class="btn btn-soft !min-h-14 !w-16" disabled={index === 0} aria-label={t('cook.prev')} onclick={() => move(-1)}><ChevronLeft size={24} /></button>
		{#if last}
			<button class="btn btn-primary !min-h-14 !text-lg" onclick={finish}>{t('cook.done')}</button>
		{:else}
			<button class="btn btn-primary !min-h-14 !text-lg" onclick={() => move(1)}>{t('cook.next')}<ChevronRight size={22} /></button>
		{/if}
	</footer>
</div>
