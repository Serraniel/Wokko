<script lang="ts">
	import { Blocks, BookMarked, Carrot, ChevronRight, Settings, Shield, Trash2, Users } from '@lucide/svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	const links = $derived([
		{ href: '/books', icon: BookMarked, label: t('nav.books') },
		{ href: '/integrations', icon: Blocks, label: t('nav.integrations') },
		{ href: '/ingredients', icon: Carrot, label: t('nav.ingredients') },
		{ href: '/people', icon: Users, label: t('nav.people') },
		{ href: '/settings', icon: Settings, label: t('nav.settings') },
		{ href: '/trash', icon: Trash2, label: t('trash.title') },
		...(session.isAdmin ? [{ href: '/admin', icon: Shield, label: t('nav.admin') }] : [])
	]);
</script>

<svelte:head><title>{t('nav.more')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-3 p-4">
	<h1 class="text-xl font-semibold">{t('nav.more')}</h1>
	<ul class="card divide-y divide-line overflow-hidden">
		{#each links as l (l.href)}
			{@const Icon = l.icon}
			<li><a class="flex min-h-12 items-center gap-3 px-4 font-medium hover:bg-surface-2" href={l.href}><Icon size={18} class="text-muted" /><span class="flex-1">{l.label}</span><ChevronRight size={16} class="text-muted" /></a></li>
		{/each}
	</ul>
</div>
