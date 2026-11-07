<script lang="ts">
	import { Download, Link2 } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { ImportedRecipe } from '#lib/api/types.gen.ts';
	import { t } from '#lib/i18n/index.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';

	let { onimport }: { onimport: (draft: ImportedRecipe) => void } = $props();

	let url = $state('');
	let text = $state('');
	let pasting = $state(false);
	let working = $state(false);
	let error = $state('');

	async function run(e: Event) {
		e.preventDefault();
		e.stopPropagation();
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
	{#if pasting}
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
	<button type="button" class="inline-flex min-h-9 items-center gap-1.5 text-sm font-medium text-muted hover:text-ink" onclick={() => { pasting = !pasting; error = ''; }}>
		<Link2 size={15} />{pasting ? t('import.link_instead') : t('import.paste_instead')}
	</button>
</section>
