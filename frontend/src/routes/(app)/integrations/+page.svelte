<script lang="ts">
	import { Bookmark, Check, Download, FileUp, Smartphone, TriangleAlert } from '@lucide/svelte';
	import { ApiError, errorText } from '#lib/api/client.ts';
	import { api } from '#lib/api/client.ts';
	import type { FileImportResult } from '#lib/api/types.gen.ts';
	import { forget } from '#lib/cache.ts';
	import AssistantsCard from '#lib/components/AssistantsCard.svelte';
	import CopyField from '#lib/components/CopyField.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { bookmarklet } from '#lib/share.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let file = $state<File | null>(null);
	let preview = $state<FileImportResult | null>(null);
	let result = $state<FileImportResult | null>(null);
	let busy = $state(false);
	let error = $state('');
	let undone = $state(false);

	const secure = typeof window !== 'undefined' && window.isSecureContext;
	const mark = $derived(bookmarklet(location.origin));
	const toImport = $derived((preview?.items ?? []).filter((i) => !i.duplicate).length);

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

<svelte:head><title>{t('nav.integrations')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-3 p-4 md:px-6">
	<header>
		<h1 class="text-xl font-semibold">{t('nav.integrations')}</h1>
		<p class="hint !mt-1">{t('integrations.hint')}</p>
	</header>

	<h2 class="label !mt-5">{t('integrations.in_out')}</h2>

	<section class="card space-y-3 p-4" aria-labelledby="file-title">
		<h3 id="file-title" class="flex items-center gap-2 font-semibold"><FileUp size={17} class="text-muted" />{t('fileimport.from_file')}</h3>
		<p class="hint !mt-0">{t('fileimport.from_file_hint')}</p>
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
			<button class="btn btn-primary w-full" disabled={busy || toImport === 0} onclick={run}>{busy ? t('import.working') : t('fileimport.import', { count: toImport })}</button>
			<p class="hint !mt-0">{t('fileimport.private')}</p>
		{/if}

		{#if result}
			<p class="flex items-center gap-2 font-medium text-ok" role="status"><Check size={17} />{t('fileimport.done', { count: result.imported })}</p>
			{#if result.skipped}<p class="text-sm text-muted">{t('fileimport.skipped', { count: result.skipped })}</p>{/if}
			{#each result.items.filter((i) => i.error || i.photo_failed) as item, n (n)}
				<p class="flex items-start gap-2 text-sm text-warn"><TriangleAlert size={15} class="mt-0.5 shrink-0" />{item.error ? t('fileimport.failed', { name: item.name }) : t('fileimport.no_photo', { name: item.name })}</p>
			{/each}
			<div class="flex flex-wrap gap-2">
				<a class="btn btn-primary flex-1" href="/recipes">{t('nav.recipes')}</a>
				{#if result.imported && !undone}<button class="btn btn-soft flex-1" disabled={busy} onclick={undo}>{t('fileimport.undo')}</button>{/if}
			</div>
		{/if}
	</section>

	<section class="card space-y-2 p-4" aria-labelledby="bookmark-title">
		<h3 id="bookmark-title" class="flex items-center gap-2 font-semibold"><Bookmark size={17} class="text-muted" />{t('fileimport.bookmark')}</h3>
		<p class="hint !mt-0">{t('fileimport.bookmark_hint')}</p>
		<CopyField value={mark} label={t('fileimport.bookmark')} />
	</section>

	<section class="card space-y-2 p-4" aria-labelledby="phone-title">
		<h3 id="phone-title" class="flex items-center gap-2 font-semibold"><Smartphone size={17} class="text-muted" />{t('fileimport.phone')}</h3>
		<p class="hint !mt-0">{secure ? t('fileimport.phone_hint') : t('fileimport.phone_needs_https')}</p>
	</section>

	<a class="card flex items-center gap-3 p-4 font-medium hover:bg-surface-2" href="/api/v1/me/export" download data-sveltekit-reload><Download size={18} class="shrink-0 text-muted" /><span><span class="block">{t('settings.export')}</span><span class="block text-sm font-normal text-muted">{t('settings.export_hint')}</span></span></a>

	<div class="pt-3"><AssistantsCard /></div>
</div>
