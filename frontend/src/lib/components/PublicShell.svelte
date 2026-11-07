<script lang="ts">
	import type { Snippet } from 'svelte';
	import { session } from '#lib/session.svelte.ts';
	import { t } from '#lib/i18n/index.svelte.ts';
	import AppFooter from './AppFooter.svelte';
	import Logo from './Logo.svelte';

	let { children }: { children: Snippet } = $props();
</script>

<div class="flex min-h-dvh flex-col">
	<header class="print-hide sticky top-0 z-20 flex items-center justify-between gap-3 border-b border-line bg-surface/90 px-4 py-2 backdrop-blur">
		<a href="/" class="flex items-center gap-2 font-bold"><Logo size={30} />Wokko</a>
		{#if session.user}
			<a class="btn btn-soft" href="/recipes">{t('nav.recipes')}</a>
		{:else}
			<a class="btn btn-soft" href="/login">{t('auth.login')}</a>
		{/if}
	</header>
	<main class="flex-1">{@render children()}</main>
	<AppFooter />
</div>
