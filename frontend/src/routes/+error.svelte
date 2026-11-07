<script lang="ts">
	import { page } from '$app/state';
	import Mascot from '#lib/components/Mascot.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
</script>

<main class="flex min-h-dvh flex-col items-center justify-center gap-3 p-6 text-center">
	<Mascot />
	<h1 class="text-2xl font-bold">{page.status === 404 ? t('notfound.title') : t('err.server_error')}</h1>
	<p class="max-w-sm text-muted">{t('notfound.text')}</p>
	<a class="btn btn-primary" href="/recipes">{t('notfound.home')}</a>
</main>
