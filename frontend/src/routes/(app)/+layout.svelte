<script lang="ts">
	import { BookOpen, Carrot, LogOut, Plus, Settings, Shield, Users } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '#lib/api/client.ts';
	import { forget } from '#lib/cache.ts';
	import Logo from '#lib/components/Logo.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	let { children }: { children: Snippet } = $props();

	$effect(() => {
		if (session.ready && !session.user && session.status?.setup_complete) goto(`/login?next=${encodeURIComponent(page.url.pathname + page.url.search)}`, { replaceState: true });
	});

	const items = $derived([
		{ href: '/recipes', icon: BookOpen, label: t('nav.recipes') },
		{ href: '/ingredients', icon: Carrot, label: t('nav.ingredients') },
		{ href: '/people', icon: Users, label: t('nav.people') },
		{ href: '/settings', icon: Settings, label: t('nav.settings') }
	]);

	const active = (href: string) => page.url.pathname === href || page.url.pathname.startsWith(href + '/');

	async function logout() {
		await api.post('/auth/logout').catch(() => {});
		forget();
		session.signedOut();
	}
</script>

{#if session.user}
	<div class="flex min-h-dvh">
		<!-- Desktop: a fixed side rail. -->
		<aside class="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col gap-1 border-r border-line bg-surface p-3 md:flex">
			<a href="/recipes" class="mb-3 flex items-center gap-2 px-2 py-1 text-lg font-bold"><Logo size={34} />Wokko</a>
			<a href="/recipes/new" class="btn btn-primary mb-3"><Plus size={18} />{t('recipes.new')}</a>
			{#each items as item (item.href)}
				{@const Icon = item.icon}
				<a href={item.href} aria-current={active(item.href) ? 'page' : undefined} class="flex items-center gap-3 rounded-xl px-3 py-2.5 font-medium transition-colors {active(item.href) ? 'bg-brand-soft text-brand-soft-ink' : 'hover:bg-surface-2'}"><Icon size={20} />{item.label}</a>
			{/each}
			{#if session.isAdmin}
				<a href="/admin" aria-current={active('/admin') ? 'page' : undefined} class="flex items-center gap-3 rounded-xl px-3 py-2.5 font-medium transition-colors {active('/admin') ? 'bg-brand-soft text-brand-soft-ink' : 'hover:bg-surface-2'}"><Shield size={20} />{t('nav.admin')}</a>
			{/if}
			<div class="mt-auto">
				<button class="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 font-medium text-muted hover:bg-surface-2" onclick={logout}><LogOut size={20} />{t('auth.logout')}</button>
			</div>
		</aside>

		<main class="min-w-0 flex-1 pb-[calc(4.5rem+env(safe-area-inset-bottom))] md:pb-0">
			{@render children()}
		</main>
	</div>

	<!-- Phone: a bottom bar within thumb reach, with "new recipe" in the middle. -->
	<nav class="fixed inset-x-0 bottom-0 z-30 grid grid-cols-5 border-t border-line bg-surface/95 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden" aria-label={t('nav.main')}>
		{#each items.slice(0, 2) as item (item.href)}
			{@const Icon = item.icon}
			<a href={item.href} aria-current={active(item.href) ? 'page' : undefined} class="flex h-16 flex-col items-center justify-center gap-0.5 text-[0.6875rem] font-medium {active(item.href) ? 'text-brand' : 'text-muted'}"><Icon size={22} /><span class="max-w-full truncate px-1">{item.label}</span></a>
		{/each}
		<a href="/recipes/new" class="flex h-16 items-center justify-center" aria-label={t('recipes.new')}><span class="flex h-12 w-12 items-center justify-center rounded-2xl bg-brand text-brand-ink shadow-card"><Plus size={26} /></span></a>
		{#each items.slice(2) as item (item.href)}
			{@const Icon = item.icon}
			<a href={item.href} aria-current={active(item.href) ? 'page' : undefined} class="flex h-16 flex-col items-center justify-center gap-0.5 text-[0.6875rem] font-medium {active(item.href) ? 'text-brand' : 'text-muted'}"><Icon size={22} /><span class="max-w-full truncate px-1">{item.label}</span></a>
		{/each}
	</nav>
{/if}
