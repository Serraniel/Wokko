<script lang="ts">
	import { ArrowLeft, Pencil, Trash2, X } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import type { RecipeBookDetail } from '#lib/api/types.gen.ts';
	import { forget } from '#lib/cache.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import RecipeCard from '#lib/components/RecipeCard.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let book = $state<RecipeBookDetail | null>(null);
	let error = $state('');
	let open = $state(false);
	let name = $state('');
	let description = $state('');
	let formError = $state('');

	async function load() {
		try {
			book = await api.get<RecipeBookDetail>(`/books/${page.params.id}`);
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	function edit() {
		name = book?.name ?? '';
		description = book?.description ?? '';
		formError = '';
		open = true;
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api.put(`/books/${page.params.id}`, { name, description });
			open = false;
			forget('books');
			await load();
		} catch (err) {
			formError = errorText(err);
		}
	}

	async function remove() {
		try {
			await api.del(`/books/${page.params.id}`);
			forget('books');
			await goto('/books');
		} catch (err) {
			formError = errorText(err);
		}
	}

	async function takeOut(id: number) {
		try {
			await api.del(`/books/${page.params.id}/recipes/${id}`);
			await load();
		} catch (err) {
			error = errorText(err);
		}
	}
</script>

<svelte:head><title>{book?.name ?? t('books.title')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-[96rem] space-y-3 p-4 md:px-6">
	<header class="flex items-center gap-2">
		<a class="btn btn-ghost btn-icon" href="/books" aria-label={t('common.back')}><ArrowLeft size={18} /></a>
		<div class="min-w-0 flex-1">
			<h1 class="text-xl font-semibold">{book?.name ?? ''}</h1>
			{#if book?.description}<p class="text-sm text-muted">{book.description}</p>{/if}
		</div>
		<button class="btn btn-ghost btn-icon" aria-label={t('common.edit')} onclick={edit}><Pencil size={17} /></button>
	</header>
	<ErrorNote text={error} />
	{#if book?.recipes.length === 0}
		<p class="py-12 text-center text-sm text-muted">{t('books.empty_book')}</p>
	{:else if book}
		<div class="grid grid-cols-2 gap-x-3 gap-y-5 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
			{#each book.recipes as r (r.id)}
				<div class="relative">
					<RecipeCard recipe={r} href="/recipes/{r.id}" />
					<button class="btn btn-soft btn-icon absolute top-1.5 right-1.5 !min-h-8 !w-8 shadow-card" aria-label={t('common.delete')} onclick={() => takeOut(r.id)}><X size={14} /></button>
				</div>
			{/each}
		</div>
	{/if}
</div>

<Sheet bind:open title={t('common.edit')}>
	<form class="space-y-3" onsubmit={save}>
		<ErrorNote text={formError} />
		<div>
			<label class="label" for="book-name">{t('books.name')}</label>
			<input id="book-name" class="input" bind:value={name} required maxlength="200" />
		</div>
		<div>
			<label class="label" for="book-description">{t('books.description')}</label>
			<input id="book-description" class="input" bind:value={description} />
		</div>
		<div class="flex gap-2">
			<button type="button" class="btn btn-danger btn-icon" aria-label={t('common.delete')} onclick={remove}><Trash2 size={17} /></button>
			<button class="btn btn-primary flex-1">{t('common.save')}</button>
		</div>
		<p class="hint">{t('books.delete_hint')}</p>
	</form>
</Sheet>
