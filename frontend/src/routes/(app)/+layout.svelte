<script lang="ts">
	import { BookMarked, BookOpen, CalendarDays, Carrot, LogOut, Menu, Plus, Search, Settings, Shield, ShoppingCart, Users } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '#lib/api/client.ts';
	import { forget } from '#lib/cache.ts';
	import AppFooter from '#lib/components/AppFooter.svelte';
	import CommandPalette from '#lib/components/CommandPalette.svelte';
	import Logo from '#lib/components/Logo.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	let { children }: { children: Snippet } = $props();

	$effect(() => {
		if (session.ready && !session.user && session.status?.setup_complete) goto(`/login?next=${encodeURIComponent(page.url.pathname + page.url.search)}`, { replaceState: true });
	});

	const items = $derived([
		{ href: '/recipes', icon: BookOpen, label: t('nav.recipes') },
		{ href: '/plan', icon: CalendarDays, label: t('nav.plan') },
		{ href: '/shopping', icon: ShoppingCart, label: t('nav.shopping') },
		{ href: '/books', icon: BookMarked, label: t('nav.books') },
		{ href: '/ingredients', icon: Carrot, label: t('nav.ingredients') },
		{ href: '/people', icon: Users, label: t('nav.people') },
		{ href: '/settings', icon: Settings, label: t('nav.settings') }
	]);
	// A phone has room for the three things used most. The rest live behind "More".
	const phoneItems = $derived([items[0], items[1], items[2], { href: '/more', icon: Menu, label: t('nav.more') }]);
	const inMore = (path: string) => ['/more', '/books', '/ingredients', '/people', '/settings', '/trash', '/admin'].some((h) => path === h || path.startsWith(h + '/'));
	const activeOnPhone = (href: string) => (href === '/more' ? inMore(page.url.pathname) : active(href));

	const active = (href: string) => page.url.pathname === href || page.url.pathname.startsWith(href + '/');

	let paletteOpen = $state(false);
	const isMac = typeof navigator !== 'undefined' && /mac|iphone|ipad/i.test(navigator.platform);

	function shortcuts(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			paletteOpen = !paletteOpen;
			return;
		}
		const el = e.target as HTMLElement;
		if (e.metaKey || e.ctrlKey || e.altKey || paletteOpen || el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName) || document.querySelector('dialog[open]')) return;
		if (e.key === '/') {
			e.preventDefault();
			paletteOpen = true;
		} else if (e.key === 'n') {
			e.preventDefault();
			goto('/recipes/new');
		}
	}

	async function logout() {
		await api.post('/auth/logout').catch(() => {});
		forget();
		session.signedOut();
	}
</script>

<svelte:window onkeydown={shortcuts} />

{#if session.user}
	<div class="flex min-h-dvh">
		<!-- Desktop: a fixed side rail. -->
		<aside class="print-hide sticky top-0 hidden h-dvh w-56 shrink-0 flex-col gap-0.5 overflow-y-auto border-r border-line p-2.5 md:flex">
			<a href="/recipes" class="mb-2 flex items-center gap-2 px-2 py-1.5 font-semibold tracking-tight"><Logo size={24} />Wokko</a>
			<button class="mb-2 flex h-9 items-center gap-2 rounded-lg bg-surface px-2.5 text-sm text-muted ring-1 ring-line ring-inset hover:bg-surface-2" onclick={() => (paletteOpen = true)}>
				<Search size={15} /><span class="min-w-0 flex-1 truncate text-left">{t('palette.open')}</span><span class="kbd">{isMac ? '⌘K' : 'Ctrl K'}</span>
			</button>
			{#each items as item (item.href)}
				{@const Icon = item.icon}
				<a href={item.href} aria-current={active(item.href) ? 'page' : undefined} class="flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-sm font-medium {active(item.href) ? 'bg-surface-2 text-ink' : 'text-muted hover:bg-surface-2 hover:text-ink'}"><Icon size={17} />{item.label}</a>
			{/each}
			{#if session.isAdmin}
				<a href="/admin" aria-current={active('/admin') ? 'page' : undefined} class="flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-sm font-medium {active('/admin') ? 'bg-surface-2 text-ink' : 'text-muted hover:bg-surface-2 hover:text-ink'}"><Shield size={17} />{t('nav.admin')}</a>
			{/if}
			<a href="/recipes/new" class="btn btn-primary mt-3"><Plus size={16} />{t('recipes.new')}</a>
			<div class="mt-auto">
				<button class="flex h-9 w-full items-center gap-2.5 rounded-lg px-2.5 text-sm font-medium text-muted hover:bg-surface-2 hover:text-ink" onclick={logout}><LogOut size={17} />{t('auth.logout')}</button>
			</div>
		</aside>

		<main class="min-w-0 flex-1 pb-[calc(4rem+env(safe-area-inset-bottom))] md:pb-0">
			{@render children()}
			<AppFooter />
		</main>
	</div>

	<!-- Phone: a bottom bar within thumb reach, with "new recipe" in the middle. -->
	<nav class="print-hide fixed inset-x-0 bottom-0 z-30 grid grid-cols-5 border-t border-line bg-bg/90 pb-[env(safe-area-inset-bottom)] backdrop-blur-lg md:hidden" aria-label={t('nav.main')}>
		{#each phoneItems.slice(0, 2) as item (item.href)}
			{@const Icon = item.icon}
			<a href={item.href} aria-current={activeOnPhone(item.href) ? 'page' : undefined} class="flex h-14 flex-col items-center justify-center gap-0.5 text-[0.625rem] font-medium {activeOnPhone(item.href) ? 'text-ink' : 'text-muted'}"><Icon size={20} strokeWidth={activeOnPhone(item.href) ? 2.4 : 1.8} /><span class="max-w-full truncate px-1">{item.label}</span></a>
		{/each}
		<a href="/recipes/new" class="flex h-14 items-center justify-center" aria-label={t('recipes.new')}><span class="flex h-10 w-10 items-center justify-center rounded-xl bg-brand text-brand-ink"><Plus size={22} /></span></a>
		{#each phoneItems.slice(2) as item (item.href)}
			{@const Icon = item.icon}
			<a href={item.href} aria-current={activeOnPhone(item.href) ? 'page' : undefined} class="flex h-14 flex-col items-center justify-center gap-0.5 text-[0.625rem] font-medium {activeOnPhone(item.href) ? 'text-ink' : 'text-muted'}"><Icon size={20} strokeWidth={activeOnPhone(item.href) ? 2.4 : 1.8} /><span class="max-w-full truncate px-1">{item.label}</span></a>
		{/each}
	</nav>

	<CommandPalette bind:open={paletteOpen} />
{/if}
