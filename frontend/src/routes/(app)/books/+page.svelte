<script lang="ts">
	import { BookMarked, Plus } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { RecipeBook } from '#lib/api/types.gen.ts';
	import { cached, remember } from '#lib/cache.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let books = $state<RecipeBook[] | null>(cached('books') ?? null);
	let error = $state('');
	let open = $state(false);
	let name = $state('');
	let description = $state('');
	let formError = $state('');

	async function load() {
		try {
			books = remember('books', await api.get<RecipeBook[]>('/books'));
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		formError = '';
		try {
			await api.post('/books', { name, description });
			open = false;
			name = '';
			description = '';
			await load();
		} catch (err) {
			formError = errorText(err);
		}
	}
</script>

<svelte:head><title>{t('books.title')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-4xl space-y-3 p-4 md:px-6">
	<header class="flex items-center justify-between gap-2">
		<h1 class="text-xl font-semibold">{t('books.title')}</h1>
		<button class="btn btn-soft" onclick={() => (open = true)}><Plus size={16} />{t('books.new')}</button>
	</header>
	<ErrorNote text={error} />
	{#if books?.length === 0}
		<div class="flex flex-col items-center gap-2 py-14 text-center">
			<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-surface-2 text-muted"><BookMarked size={22} /></div>
			<p class="font-semibold">{t('books.empty_title')}</p>
			<p class="max-w-xs text-sm text-muted">{t('books.empty_text')}</p>
		</div>
	{:else if books}
		<ul class="grid gap-2 sm:grid-cols-2">
			{#each books as b (b.id)}
				<li>
					<a class="card block p-3 hover:bg-surface-2" href="/books/{b.id}">
						<span class="block font-semibold">{b.name}</span>
						{#if b.description}<span class="block text-sm text-muted">{b.description}</span>{/if}
						<span class="block text-xs text-muted">{t('books.count', { count: b.recipe_count })}</span>
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<Sheet bind:open title={t('books.new')}>
	<form class="space-y-3" onsubmit={create}>
		<ErrorNote text={formError} />
		<div>
			<label class="label" for="book-name">{t('books.name')}</label>
			<input id="book-name" class="input" bind:value={name} required maxlength="200" />
		</div>
		<div>
			<label class="label" for="book-description">{t('books.description')}</label>
			<input id="book-description" class="input" bind:value={description} />
		</div>
		<button class="btn btn-primary w-full">{t('common.save')}</button>
	</form>
</Sheet>
