<script lang="ts">
	import { Bookmark, Bot, Download, FileUp, Smartphone } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import type { APIToken, AssistantActivity } from '#lib/api/types.gen.ts';
	import { assistantApps, type AssistantApp } from '#lib/assistantApps.ts';
	import AssistantSettings from '#lib/components/AssistantSettings.svelte';
	import AssistantSetup from '#lib/components/AssistantSetup.svelte';
	import CopyField from '#lib/components/CopyField.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import FileImport from '#lib/components/FileImport.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { formatDate, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { bookmarklet } from '#lib/share.ts';

	// An overview of every integration. Each opens its own dialog to set it up or manage it.
	type Card = { id: string; icon: Component; title: string; text: string };
	const cards = $derived<Card[]>([
		{ id: 'file', icon: FileUp, title: t('fileimport.from_file'), text: t('integrations.file_short') },
		{ id: 'website', icon: Bookmark, title: t('fileimport.bookmark'), text: t('integrations.website_short') },
		{ id: 'phone', icon: Smartphone, title: t('fileimport.phone'), text: t('integrations.phone_short') },
		{ id: 'export', icon: Download, title: t('export.all_title'), text: t('integrations.export_short') }
	]);

	const secure = typeof window !== 'undefined' && window.isSecureContext;
	const mark = $derived(bookmarklet(location.origin));
	const address = $derived(`${location.origin}/mcp`);
	const apps = $derived(assistantApps(address));
	const on = $derived(session.status?.mcp_enabled);

	let opened = $state(''); // a card's id, or an app's id after "app:"
	let sheetOpen = $state(false);
	let exportScope = $state('mine');
	let tokens = $state<APIToken[]>([]);
	let activity = $state<AssistantActivity[]>([]);
	let error = $state('');

	function open(id: string) {
		opened = id;
		sheetOpen = true;
	}

	// Links like /integrations?open=file or ?connect=claude open that dialog directly.
	$effect(() => {
		const want = page.url.searchParams.get('open');
		const app = page.url.searchParams.get('connect');
		if (want && ['file', 'website', 'phone', 'export'].includes(want)) open(want);
		else if (app && apps.some((a) => a.id === app)) open(`app:${app}`);
	});

	async function load() {
		if (!on) return;
		try {
			[tokens, activity] = await Promise.all([api.get<APIToken[]>('/me/tokens'), api.get<AssistantActivity[]>('/me/assistant-activity')]);
		} catch (e) {
			error = errorText(e);
		}
	}
	$effect(() => {
		if (on) load();
	});

	const tokensFor = (app: AssistantApp) => tokens.filter((tk) => tk.app === app.id);
	const appName = (id: string) => apps.find((a) => a.id === id)?.name ?? t('assistant.app_other');
	/** The newest thing this app did, from the activity of its tokens. */
	function lastActivity(app: AssistantApp): string {
		const names = new Set(tokensFor(app).map((tk) => tk.name));
		return activity.find((a) => names.has(a.token_name))?.created_at ?? '';
	}

	const openedApp = $derived(opened.startsWith('app:') ? apps.find((a) => `app:${a.id}` === opened) : undefined);
	const openedCard = $derived(cards.find((c) => c.id === opened));
	const sheetTitle = $derived(openedApp ? t('assistant.connect_title', { app: openedApp.name }) : (openedCard?.title ?? ''));
</script>

<svelte:head><title>{t('nav.integrations')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-6xl space-y-3 p-4 md:px-6">
	<header>
		<h1 class="text-xl font-semibold">{t('nav.integrations')}</h1>
		<p class="hint !mt-1">{t('integrations.hint')}</p>
	</header>

	<h2 class="label !mt-5">{t('integrations.in_out')}</h2>
	<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
		{#each cards as c (c.id)}
			{@const Icon = c.icon}
			<section class="card flex flex-col gap-2 p-4" aria-labelledby="card-{c.id}">
				<h3 id="card-{c.id}" class="flex items-center gap-2 font-semibold"><Icon size={17} class="text-muted" />{c.title}</h3>
				<p class="hint !mt-0 flex-1">{c.text}</p>
				<button type="button" class="btn btn-soft w-full" onclick={() => open(c.id)}>{t('integrations.open')}</button>
			</section>
		{/each}
	</div>

	<h2 id="assistants" class="label !mt-6 flex items-center gap-1.5"><Bot size={15} />{t('assistant.title')}</h2>
	<p class="hint !mt-0">{t('assistant.hint')}</p>
	<ErrorNote text={error} />
	{#if !on}
		<p class="card p-4 text-sm text-muted">{session.isAdmin ? t('assistant.off_admin') : t('assistant.off')}</p>
	{:else}
		<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
			{#each apps as a (a.id)}
				{@const connected = tokensFor(a).length > 0}
				{@const last = lastActivity(a)}
				<section class="card flex flex-col gap-2 p-4" aria-labelledby="app-{a.id}">
					<div class="flex items-start justify-between gap-2">
						<h3 id="app-{a.id}" class="font-semibold">{a.name}</h3>
						<span class="chip !min-h-6 shrink-0 text-xs {connected ? 'bg-ok-soft text-ok' : 'text-muted'}">{connected ? t('assistant.connected') : t('assistant.not_connected')}</span>
					</div>
					<p class="hint !mt-0 flex-1">{last ? t('assistant.last_activity', { date: formatDate(last, 'datetime') }) : t(`assistant.${a.id}_short`)}</p>
					<button type="button" class="btn btn-soft w-full" onclick={() => open(`app:${a.id}`)}>{connected ? t('assistant.manage') : t('assistant.connect')}</button>
				</section>
			{/each}
		</div>
		<AssistantSettings {tokens} {activity} {appName} onchange={load} />
	{/if}
</div>

<Sheet bind:open={sheetOpen} title={sheetTitle}>
	{#if openedApp}
		<AssistantSetup app={openedApp} {address} tokens={tokensFor(openedApp)} onchange={load} />
	{:else if opened === 'file'}
		<div class="space-y-3">
			<p class="hint !mt-0">{t('fileimport.from_file_hint')}</p>
			<FileImport />
		</div>
	{:else if opened === 'website'}
		<div class="space-y-3">
			<p class="hint !mt-0">{t('fileimport.bookmark_hint')}</p>
			<CopyField value={mark} label={t('fileimport.bookmark')} />
			<p class="hint !mt-0">{t('integrations.website_how')}</p>
		</div>
	{:else if opened === 'phone'}
		<p class="hint !mt-0">{secure ? t('fileimport.phone_hint') : t('fileimport.phone_needs_https')}</p>
	{:else if opened === 'export'}
		<div class="space-y-3">
			<p class="hint !mt-0">{t('settings.export_hint')}</p>
			<fieldset class="flex flex-wrap gap-2">
				<legend class="label">{t('export.which')}</legend>
				{#each ['mine', 'all'] as sc (sc)}
					<label class="chip cursor-pointer {exportScope === sc ? 'chip-active' : 'hover:bg-surface-2'}"><input type="radio" class="sr-only" name="export-scope" value={sc} bind:group={exportScope} />{t(`export.scope_${sc}`)}</label>
				{/each}
			</fieldset>
			{#if exportScope === 'all'}<p class="hint !mt-0">{t('export.scope_all_hint')}</p>{/if}
			<div class="grid gap-2 sm:grid-cols-2">
				<a class="btn btn-soft" href="/api/v1/me/export?scope={exportScope}&format=wokko" download data-sveltekit-reload>{t('export.wokko')}</a>
				<a class="btn btn-soft" href="/api/v1/me/export?scope={exportScope}&format=schema" download data-sveltekit-reload>{t('export.schema')}</a>
			</div>
			<p class="hint !mt-0">{t('export.formats_hint')}</p>
		</div>
	{/if}
</Sheet>
