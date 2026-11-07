<script lang="ts">
	import { ExternalLink } from '@lucide/svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import PublicShell from '#lib/components/PublicShell.svelte';
	import { formatChangelog } from '#lib/changelog.ts';
	import { t } from '#lib/i18n/index.svelte.ts';

	let html = $state('');
	let error = $state('');

	fetch('/changelog.md')
		.then((r) => {
			if (!r.ok) throw new Error(String(r.status));
			return r.text();
		})
		.then((text) => (html = formatChangelog(text)))
		.catch(() => (error = t('err.network')));
</script>

<svelte:head><title>{t('settings.changelog')} · Wokko</title></svelte:head>

<PublicShell>
	<div class="mx-auto max-w-2xl space-y-3 p-4">
		<h1 class="text-xl font-semibold">{t('settings.changelog')}</h1>
		<a class="inline-flex min-h-9 items-center gap-1.5 text-sm text-muted hover:text-ink" href="https://github.com/Serraniel/Wokko/releases" rel="noopener noreferrer" target="_blank"><ExternalLink size={14} />{t('changelog.releases')}</a>
		<ErrorNote text={error} />
		<div class="changelog space-y-2 text-sm">{@html html}</div>
	</div>
</PublicShell>
