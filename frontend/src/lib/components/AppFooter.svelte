<script lang="ts">
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	const version = $derived((session.status?.version ?? '').replace(/^v/, ''));
	const label = $derived(/^\d/.test(version) ? `v${version}` : version);
</script>

<footer class="print-hide flex flex-wrap justify-center gap-x-3 gap-y-1 px-4 py-4 text-xs text-muted">
	<a class="inline-flex min-h-6 items-center hover:text-ink hover:underline" href="https://github.com/Serraniel/Wokko" rel="noopener noreferrer" target="_blank">GitHub</a>
	{#if version}<a class="inline-flex min-h-6 items-center tabular-nums hover:text-ink hover:underline" href="/changelog">{label}</a>{/if}
	<a class="inline-flex min-h-6 items-center hover:text-ink hover:underline" href="/changelog">{t('settings.changelog')}</a>
	<a class="inline-flex min-h-6 items-center hover:text-ink hover:underline" href="https://eupl.eu/1.2/en/" rel="noopener noreferrer" target="_blank">EUPL-1.2</a>
	<a class="inline-flex min-h-6 items-center hover:text-ink hover:underline" href="/licenses">{t('settings.notices')}</a>
</footer>
