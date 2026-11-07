<script lang="ts">
	import '../app.css';
	import { RefreshCw } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { setUnauthorizedHandler } from '#lib/api/client.ts';
	import Logo from '#lib/components/Logo.svelte';
	import Toasts from '#lib/components/Toasts.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	let { children }: { children: Snippet } = $props();
	let failed = $state(false);

	const publicPaths = ['/login', '/register', '/reset', '/setup', '/gallery', '/s/', '/shared/', '/oauth/'];

	setUnauthorizedHandler(() => {
		session.signedOut();
		if (session.ready && !publicPaths.some((p) => page.url.pathname.startsWith(p))) {
			goto(`/login?next=${encodeURIComponent(page.url.pathname + page.url.search)}`);
		}
	});

	async function start() {
		failed = false;
		try {
			await session.load();
		} catch {
			failed = true;
		}
	}
	start();

	$effect(() => {
		if (session.ready && session.status && !session.status.setup_complete && page.url.pathname !== '/setup') goto('/setup', { replaceState: true });
	});

</script>

{#if failed}
	<main class="flex min-h-dvh flex-col items-center justify-center gap-4 p-6 text-center">
		<Logo size={72} />
		<h1 class="text-xl font-bold">{t('offline.title')}</h1>
		<p class="max-w-sm text-muted">{t('offline.text')}</p>
		<button class="btn btn-primary" onclick={start}><RefreshCw size={18} />{t('common.retry')}</button>
	</main>
{:else if session.ready}
	{@render children()}
{:else}
	<div class="late flex min-h-dvh items-center justify-center" aria-hidden="true"><Logo size={48} /></div>
{/if}
<Toasts />
