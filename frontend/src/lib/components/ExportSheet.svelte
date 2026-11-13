<script lang="ts">
	import { FileArchive, Share } from '@lucide/svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import Sheet from './Sheet.svelte';

	// base is the export address without its format, such as /api/v1/recipes/12/export.
	let { open = $bindable(false), base, title }: { open: boolean; base: string; title: string } = $props();
</script>

<Sheet bind:open {title}>
	<p class="hint !mt-0 mb-2">{t('export.hint')}</p>
	<div class="divide-y divide-line">
		<a class="flex min-h-14 items-center gap-3 py-2" href="{base}?format=wokko" download data-sveltekit-reload onclick={() => (open = false)}>
			<FileArchive size={18} class="shrink-0 text-muted" />
			<span><span class="block font-medium">{t('export.wokko')}</span><span class="block text-sm text-muted">{t('export.wokko_hint')}</span></span>
		</a>
		<a class="flex min-h-14 items-center gap-3 py-2" href="{base}?format=schema" download data-sveltekit-reload onclick={() => (open = false)}>
			<Share size={18} class="shrink-0 text-muted" />
			<span><span class="block font-medium">{t('export.schema')}</span><span class="block text-sm text-muted">{t('export.schema_hint')}</span></span>
		</a>
	</div>
</Sheet>
