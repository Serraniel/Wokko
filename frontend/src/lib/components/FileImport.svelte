<script lang="ts">
	import { Check, TriangleAlert } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { api, ApiError, errorText } from '#lib/api/client.ts';
	import type { FileImportResult } from '#lib/api/types.gen.ts';
	import { forget } from '#lib/cache.ts';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';

	// Importing recipe files: choose, see what's inside, import, and undo if it was wrong. With
	// openAfter, it goes straight to the imported recipe, or the list when there were several.
	let { openAfter = false }: { openAfter?: boolean } = $props();

	let file = $state<File | null>(null);
	let preview = $state<FileImportResult | null>(null);
	let result = $state<FileImportResult | null>(null);
	let busy = $state(false);
	let error = $state('');
	let undone = $state(false);

	const toImport = $derived((preview?.items ?? []).filter((i) => !i.duplicate).length);
	const imported = $derived((result?.items ?? []).filter((i) => i.recipe_id));

	async function send(dryRun: boolean): Promise<FileImportResult> {
		const body = new FormData();
		body.append('file', file as File);
		const res = await fetch(`/api/v1/import/file${dryRun ? '?dry_run=1' : ''}`, { method: 'POST', body });
		const json = await res.json().catch(() => null);
		if (!res.ok) throw new ApiError(res.status, json?.error?.code ?? 'server_error', json?.error?.message ?? '');
		return json;
	}

	async function choose(e: Event) {
		file = (e.currentTarget as HTMLInputElement).files?.[0] ?? null;
		preview = null;
		result = null;
		undone = false;
		error = '';
		if (!file) return;
		busy = true;
		try {
			preview = await send(true);
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}

	async function run() {
		busy = true;
		error = '';
		try {
			result = await send(false);
			preview = null;
			forget('recipes');
			const ids = result.items.filter((i) => i.recipe_id).map((i) => i.recipe_id);
			if (openAfter && ids.length && !result.items.some((i) => i.error || i.photo_failed)) {
				await goto(ids.length === 1 ? `/recipes/${ids[0]}` : '/recipes');
			}
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}

	// Everything just imported goes to the trash, where it can still be restored.
	async function undo() {
		busy = true;
		try {
			for (const item of result?.items ?? []) if (item.recipe_id) await api.del(`/recipes/${item.recipe_id}`);
			undone = true;
			forget('recipes');
			toasts.show(t('recipe.moved_to_trash'));
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}
</script>

<div class="space-y-3">
	<ErrorNote text={error} />
	<div>
		<label class="label" for="import-file">{t('fileimport.choose')}</label>
		<input id="import-file" class="input !py-2" type="file" accept=".zip,.json,.paprikarecipes,.paprikarecipe,application/zip,application/json" disabled={busy} onchange={choose} />
	</div>
	{#if busy && !preview && !result}<p class="text-sm text-muted" role="status">{t('common.loading')}</p>{/if}

	{#if preview}
		<p class="font-medium" role="status">{t('fileimport.found', { count: preview.items.length })}</p>
		<ul class="max-h-80 divide-y divide-line overflow-y-auto text-sm">
			{#each preview.items as item, n (n)}
				<li class="flex min-h-10 items-center justify-between gap-2 py-1">
					<span class="min-w-0"><span class="font-medium {item.duplicate ? 'text-muted' : ''}">{item.name}</span><span class="block text-xs text-muted">{item.format} · {t('fileimport.counts', { steps: item.steps, ingredients: item.ingredients })}</span></span>
					{#if item.duplicate}<span class="chip !min-h-6 shrink-0 text-xs text-muted">{t('fileimport.already')}</span>{/if}
				</li>
			{/each}
		</ul>
		<button type="button" class="btn btn-primary w-full" disabled={busy || toImport === 0} onclick={run}>{busy ? t('import.working') : t('fileimport.import', { count: toImport })}</button>
		<p class="hint !mt-0">{t('fileimport.private')}</p>
	{/if}

	{#if result}
		<p class="flex items-center gap-2 font-medium text-ok" role="status"><Check size={17} />{t('fileimport.done', { count: result.imported })}</p>
		{#if result.skipped}<p class="text-sm text-muted">{t('fileimport.skipped', { count: result.skipped })}</p>{/if}
		{#each result.items.filter((i) => i.error || i.photo_failed) as item, n (n)}
			<p class="flex items-start gap-2 text-sm text-warn"><TriangleAlert size={15} class="mt-0.5 shrink-0" />{item.error ? t('fileimport.failed', { name: item.name }) : t('fileimport.no_photo', { name: item.name })}</p>
		{/each}
		<div class="flex flex-wrap gap-2">
			{#if imported.length === 1 && !undone}
				<a class="btn btn-primary flex-1" href="/recipes/{imported[0].recipe_id}">{t('fileimport.open')}</a>
			{:else}
				<a class="btn btn-primary flex-1" href="/recipes">{t('nav.recipes')}</a>
			{/if}
			{#if result.imported && !undone}<button type="button" class="btn btn-soft flex-1" disabled={busy} onclick={undo}>{t('fileimport.undo')}</button>{/if}
		</div>
	{/if}
</div>
