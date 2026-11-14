<script lang="ts">
	import { ClipboardPaste, Download, FileUp, Link2 } from '@lucide/svelte';
	import { onMount, untrack } from 'svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { ImportedRecipe } from '#lib/api/types.gen.ts';
	import { t } from '#lib/i18n/index.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';
	import FileImport from './FileImport.svelte';

	let { onimport, sharedUrl = '', sharedText = '' }: { onimport: (draft: ImportedRecipe) => void; sharedUrl?: string; sharedText?: string } = $props();

	let url = $state(untrack(() => sharedUrl));
	let text = $state(untrack(() => sharedText));
	// How the recipe comes in: from a link, pasted text, or a recipe file.
	let mode = $state<'link' | 'text' | 'file'>(untrack(() => (!sharedUrl && sharedText ? 'text' : 'link')));
	const pasting = $derived(mode === 'text');
	let working = $state(false);
	let error = $state('');

	// Something shared from another app starts right away.
	onMount(() => {
		if (sharedUrl || sharedText) run();
	});

	async function run(e?: Event) {
		e?.preventDefault();
		e?.stopPropagation();
		if (working) return;
		working = true;
		error = '';
		try {
			onimport(await api.post<ImportedRecipe>('/import', pasting ? { url: '', text } : { url: url.trim(), text: '' }));
		} catch (err) {
			error = errorText(err);
		} finally {
			working = false;
		}
	}
</script>

<!-- A form of its own would nest inside the editor's form, which HTML doesn't allow. -->
<section class="card space-y-3 p-4" aria-labelledby="import-title">
	<div>
		<h2 id="import-title" class="flex items-center gap-2 font-semibold"><Download size={18} class="text-brand" />{t('import.title')}</h2>
		<p class="text-sm text-muted">{t('import.hint')}</p>
	</div>
	{#if mode === 'file'}
		<p class="hint !mt-0">{t('fileimport.from_file_hint')}</p>
		<FileImport openAfter />
	{:else if pasting}
		<div>
			<label class="label" for="import-text">{t('import.text_label')}</label>
			<textarea id="import-text" class="input resize-y" rows="7" bind:value={text} placeholder={t('import.text_placeholder')}></textarea>
		</div>
		<button type="button" class="btn btn-primary w-full" disabled={working || !text.trim()} onclick={run}>{working ? t('import.working') : t('import.button')}</button>
	{:else}
		<div class="flex gap-2">
			<input
				id="import-url"
				class="input min-w-0 flex-1"
				type="url"
				inputmode="url"
				autocomplete="off"
				aria-label={t('import.title')}
				placeholder={t('import.url_placeholder')}
				bind:value={url}
				onkeydown={(e) => { if (e.key === 'Enter') run(e); }}
			/>
			<button type="button" class="btn btn-primary shrink-0" disabled={working || !url.trim()} onclick={run}>{working ? t('import.working') : t('import.button')}</button>
		</div>
	{/if}
	<ErrorNote text={error} />
	<div class="flex flex-wrap gap-x-4">
		{#if mode !== 'link'}<button type="button" class="inline-flex min-h-9 items-center gap-1.5 text-sm font-medium text-muted hover:text-ink" onclick={() => { mode = 'link'; error = ''; }}><Link2 size={15} />{t('import.link_instead')}</button>{/if}
		{#if mode !== 'text'}<button type="button" class="inline-flex min-h-9 items-center gap-1.5 text-sm font-medium text-muted hover:text-ink" onclick={() => { mode = 'text'; error = ''; }}><ClipboardPaste size={15} />{t('import.paste_instead')}</button>{/if}
		{#if mode !== 'file'}<button type="button" class="inline-flex min-h-9 items-center gap-1.5 text-sm font-medium text-muted hover:text-ink" onclick={() => { mode = 'file'; error = ''; }}><FileUp size={15} />{t('import.file_instead')}</button>{/if}
	</div>
</section>
