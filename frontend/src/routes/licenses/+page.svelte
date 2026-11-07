<script lang="ts">
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import PublicShell from '#lib/components/PublicShell.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let text = $state('');
	let error = $state('');

	fetch('/third-party-notices.txt')
		.then((r) => {
			if (!r.ok) throw new Error(String(r.status));
			return r.text();
		})
		.then((body) => (text = body))
		.catch(() => (error = t('err.network')));
</script>

<svelte:head><title>{t('settings.notices')} · Wokko</title></svelte:head>

<PublicShell>
	<div class="mx-auto max-w-3xl space-y-3 p-4">
		<h1 class="text-xl font-semibold">{t('settings.notices')}</h1>
		<ErrorNote text={error} />
		<pre class="overflow-x-auto rounded-xl bg-surface p-3 text-xs whitespace-pre-wrap ring-1 ring-line ring-inset">{text}</pre>
	</div>
</PublicShell>
