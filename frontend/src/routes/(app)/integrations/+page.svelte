<script lang="ts">
	import { Bookmark, Download, FileUp, Smartphone } from '@lucide/svelte';
	import AssistantsCard from '#lib/components/AssistantsCard.svelte';
	import CopyField from '#lib/components/CopyField.svelte';
	import FileImport from '#lib/components/FileImport.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { bookmarklet } from '#lib/share.ts';

	let exportScope = $state('mine');

	const secure = typeof window !== 'undefined' && window.isSecureContext;
	const mark = $derived(bookmarklet(location.origin));
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
		<FileImport />
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

	<section class="card space-y-3 p-4" aria-labelledby="export-title">
		<h3 id="export-title" class="flex items-center gap-2 font-semibold"><Download size={17} class="text-muted" />{t('export.all_title')}</h3>
		<p class="hint !mt-0">{t('settings.export_hint')}</p>
		<fieldset class="flex flex-wrap gap-2">
			<legend class="label">{t('export.which')}</legend>
			{#each ['mine', 'all'] as sc (sc)}
				<label class="chip cursor-pointer {exportScope === sc ? 'chip-active' : 'hover:bg-surface-2'}"><input type="radio" class="sr-only" name="export-scope" value={sc} bind:group={exportScope} />{t(`export.scope_${sc}`)}</label>
			{/each}
		</fieldset>
		{#if exportScope === 'all'}<p class="hint !mt-0">{t('export.scope_all_hint')}</p>{/if}
		<div class="grid gap-2 sm:grid-cols-2">
			<a class="btn btn-soft" href="/api/v1/me/export?scope={exportScope}&format=wokko" download data-sveltekit-reload>{t('export.wokko')}</a>
			<a class="btn btn-soft" href="/api/v1/me/export?scope={exportScope}&format=schema" download data-sveltekit-reload>{t('export.schema')}</a>
		</div>
		<p class="hint !mt-0">{t('export.formats_hint')}</p>
	</section>

	<div class="pt-3"><AssistantsCard /></div>
</div>
