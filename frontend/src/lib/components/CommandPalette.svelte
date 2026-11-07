<script lang="ts">
	import { BookMarked, BookOpen, CalendarDays, Carrot, Download, Moon, Plus, Search, Settings, Shield, ShoppingCart, Trash2, Users } from '@lucide/svelte';
	import { tick, type Component } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, query } from '#lib/api/client.ts';
	import type { Page, RecipeSummary } from '#lib/api/types.gen.ts';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { warmRecipe } from '#lib/prefetch.ts';
	import { session } from '#lib/session.svelte.ts';
	import { theme } from '#lib/theme.svelte.ts';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	interface Item {
		id: string;
		label: string;
		icon: Component<{ size?: number; class?: string }>;
		run: () => void;
		recipe?: number;
	}

	let dialog = $state<HTMLDialogElement>();
	let input = $state<HTMLInputElement>();
	let q = $state('');
	let recipes = $state<RecipeSummary[]>([]);
	let recipesFor = $state('');
	let selected = $state(0);
	let request = 0;
	let timer: ReturnType<typeof setTimeout>;

	const actions = $derived<Item[]>([
		{ id: 'new', label: t('palette.new'), icon: Plus, run: () => goto('/recipes/new') },
		{ id: 'import', label: t('palette.import'), icon: Download, run: () => goto('/recipes/new').then(async () => { await tick(); document.getElementById('import-url')?.focus(); }) },
		{ id: 'recipes', label: t('nav.recipes'), icon: BookOpen, run: () => goto('/recipes') },
		{ id: 'plan', label: t('nav.plan'), icon: CalendarDays, run: () => goto('/plan') },
		{ id: 'shopping', label: t('nav.shopping'), icon: ShoppingCart, run: () => goto('/shopping') },
		{ id: 'books', label: t('nav.books'), icon: BookMarked, run: () => goto('/books') },
		{ id: 'ingredients', label: t('nav.ingredients'), icon: Carrot, run: () => goto('/ingredients') },
		{ id: 'people', label: t('nav.people'), icon: Users, run: () => goto('/people') },
		{ id: 'settings', label: t('nav.settings'), icon: Settings, run: () => goto('/settings') },
		{ id: 'trash', label: t('palette.trash'), icon: Trash2, run: () => goto('/trash') },
		...(session.isAdmin ? [{ id: 'admin', label: t('nav.admin'), icon: Shield, run: () => goto('/admin') }] : []),
		{ id: 'theme', label: t('palette.theme'), icon: Moon, run: toggleTheme }
	]);

	function toggleTheme() {
		const dark = document.documentElement.dataset.theme === 'dark' || (!document.documentElement.dataset.theme && matchMedia('(prefers-color-scheme: dark)').matches);
		theme.set(dark ? 'light' : 'dark');
	}

	const needle = $derived(q.trim().toLowerCase());
	const matchingActions = $derived(actions.filter((a) => !needle || a.label.toLowerCase().includes(needle)));
	// Until the answer for what's typed arrives, older results only stay if they still match,
	// so a quick Enter can't open a recipe the person wasn't looking for.
	const shown = $derived(recipesFor === q.trim() ? recipes : recipes.filter((r) => r.name.toLowerCase().includes(needle)));
	const recipeItems = $derived<Item[]>(shown.map((r) => ({ id: `r${r.id}`, label: r.name, icon: BookOpen, recipe: r.id, run: () => goto(`/recipes/${r.id}`) })));
	const items = $derived([...recipeItems, ...matchingActions]);

	$effect(() => {
		if (open && dialog && !dialog.open) {
			q = '';
			recipes = [];
			recipesFor = '';
			selected = 0;
			dialog.showModal();
			input?.focus();
			search();
		} else if (!open && dialog?.open) dialog.close();
	});

	async function search() {
		const mine = ++request;
		const asked = q.trim();
		try {
			const res = await api.get<Page<RecipeSummary>>(`/recipes${query({ q: asked, limit: asked ? 8 : 5, sort: 'updated' })}`);
			if (mine === request) {
				const current = items[selected]?.id;
				recipes = res.items;
				recipesFor = asked;
				// Keep the highlight on what the person is about to choose.
				await tick();
				selected = Math.max(0, items.findIndex((i) => i.id === current));
			}
		} catch {
			// Moving around still works when the search can't.
		}
	}

	function typed() {
		selected = 0;
		clearTimeout(timer);
		timer = setTimeout(search, 60);
	}

	function choose(item: Item | undefined) {
		if (!item) return;
		open = false;
		item.run();
	}

	function keys(e: KeyboardEvent) {
		if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
			e.preventDefault();
			if (!items.length) return;
			selected = (selected + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length;
			const item = items[selected];
			if (item?.recipe) warmRecipe(item.recipe);
			document.getElementById(`palette-${selected}`)?.scrollIntoView({ block: 'nearest' });
		} else if (e.key === 'Enter') {
			e.preventDefault();
			choose(items[selected]);
		}
	}
</script>

<dialog
	bind:this={dialog}
	onclose={() => (open = false)}
	onclick={(e) => e.target === dialog && (open = false)}
	class="m-0 mx-auto mt-[max(1rem,10dvh)] max-h-[75dvh] w-[calc(100%-2rem)] max-w-xl overflow-hidden rounded-2xl border border-line bg-surface p-0 text-ink shadow-card backdrop:bg-black/40"
>
	{#if open}
		<div class="flex max-h-[75dvh] flex-col" onkeydown={keys} role="presentation">
			<div class="flex items-center gap-2.5 border-b border-line px-4">
				<Search size={17} class="shrink-0 text-muted" />
				<input
					bind:this={input}
					bind:value={q}
					oninput={typed}
					class="min-h-13 w-full bg-transparent text-base outline-none placeholder:text-muted"
					placeholder={t('palette.placeholder')}
					aria-label={t('palette.placeholder')}
					role="combobox"
					aria-expanded="true"
					aria-controls="palette-list"
					aria-activedescendant={items.length ? `palette-${selected}` : undefined}
					autocomplete="off"
				/>
				<span class="kbd hidden sm:inline-flex">esc</span>
			</div>
			<ul id="palette-list" role="listbox" aria-label={t('palette.actions')} class="overflow-y-auto p-2">
				{#each items as item, i (item.id)}
					{@const Icon = item.icon}
					{#if i === 0 && recipeItems.length}<li role="presentation" class="px-2.5 pt-1 pb-1 text-xs font-medium text-muted">{t('palette.recipes')}</li>{/if}
					{#if i === recipeItems.length && matchingActions.length}<li role="presentation" class="px-2.5 pt-2 pb-1 text-xs font-medium text-muted">{t('palette.actions')}</li>{/if}
					<!-- Arrow keys and Enter arrive through the search field, which keeps focus. -->
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<li id="palette-{i}" role="option" aria-selected={i === selected} class="flex min-h-10 cursor-pointer items-center gap-2.5 rounded-lg px-2.5 text-sm {i === selected ? 'bg-surface-2' : ''}" onpointermove={() => (selected = i)} onclick={() => choose(item)}>
						<Icon size={16} class="shrink-0 text-muted" /><span class="truncate">{item.label}</span>
					</li>
				{:else}
					<li class="px-2.5 py-8 text-center text-sm text-muted">{t('palette.empty')}</li>
				{/each}
			</ul>
		</div>
	{/if}
</dialog>
