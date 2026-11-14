<script lang="ts">
	import { Bot, ExternalLink, Trash2, TriangleAlert } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { APIToken, AssistantActivity, NewAPIToken, User } from '#lib/api/types.gen.ts';
	import { formatDate, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import CopyField from './CopyField.svelte';
	import ErrorNote from './ErrorNote.svelte';

	let tokens = $state<APIToken[]>([]);
	let activity = $state<AssistantActivity[]>([]);
	let created = $state<NewAPIToken | null>(null);
	let name = $state('');
	let scope = $state('write');
	let days = $state(0);
	let error = $state('');

	const on = $derived(session.status?.mcp_enabled);
	const address = $derived(`${location.origin}/mcp`);
	const secure = location.protocol === 'https:';

	// Each app's own way in. Apps that connect by address log in through Wokko and need no
	// token. The others get a snippet that fills in the token once it's created.
	type App = { id: string; name: string; byAddress?: boolean; steps: number; snippet?: (token: string) => string; link?: (token: string) => string; doc?: string };
	// What Cursor and VS Code store for a server that's reached over HTTP with a token.
	const remote = (tk: string) => ({ url: address, headers: { Authorization: `Bearer ${tk}` } });
	const apps: App[] = [
		{ id: 'claude', name: 'Claude', byAddress: true, steps: 3, doc: 'https://www.claude.com/docs/connectors/custom/remote-mcp', link: () => `https://claude.ai/customize/connectors?${new URLSearchParams({ modal: 'add-custom-connector', connectorName: 'Wokko', connectorUrl: address })}` },
		{ id: 'chatgpt', name: 'ChatGPT', byAddress: true, steps: 3 },
		{ id: 'claude_code', name: 'Claude Code', steps: 1, snippet: (tk) => `claude mcp add --transport http wokko ${address} --header "Authorization: Bearer ${tk}"` },
		{ id: 'codex', name: 'Codex', steps: 1, doc: 'https://developers.openai.com/codex/mcp', snippet: (tk) => `export WOKKO_TOKEN=${tk}\ncodex mcp add wokko --url ${address} --bearer-token-env-var WOKKO_TOKEN` },
		{ id: 'gemini', name: 'Gemini CLI', steps: 1, snippet: (tk) => `gemini mcp add --transport http --header "Authorization: Bearer ${tk}" wokko ${address}` },
		{ id: 'cursor', name: 'Cursor', steps: 1, doc: 'https://cursor.com/docs/mcp/install-links', snippet: (tk) => JSON.stringify({ mcpServers: { wokko: remote(tk) } }, null, 2), link: (tk) => `cursor://anysphere.cursor-deeplink/mcp/install?name=wokko&config=${encodeURIComponent(btoa(JSON.stringify(remote(tk))))}` },
		{ id: 'vscode', name: 'VS Code', steps: 1, snippet: (tk) => JSON.stringify({ servers: { wokko: { type: 'http', ...remote(tk) } } }, null, 2), link: (tk) => `vscode:mcp/install?${encodeURIComponent(JSON.stringify({ name: 'wokko', type: 'http', ...remote(tk) }))}` },
		{ id: 'openclaw', name: 'OpenClaw', steps: 1, doc: 'https://docs.openclaw.ai/gateway/configuration-reference', snippet: (tk) => JSON.stringify({ mcp: { servers: { wokko: { url: address, transport: 'streamable-http', headers: { Authorization: `Bearer ${tk}` } } } } }, null, 2) },
		{ id: 'hermes', name: 'Hermes Agent', steps: 1, doc: 'https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp', snippet: (tk) => `mcp_servers:\n  wokko:\n    url: "${address}"\n    headers:\n      Authorization: "Bearer ${tk}"` },
		{ id: 'other', name: t('assistant.app_other'), steps: 2, snippet: (tk) => `Authorization: Bearer ${tk}` }
	];
	const newEntryChoices = ['never', 'ask', 'allow'] as const;
	let savingEntries = $state(false);
	async function setNewEntries(choice: string) {
		savingEntries = true;
		error = '';
		try {
			session.user = await api.patch<User>('/me', { assistant_new_entries: choice });
		} catch (e) {
			error = errorText(e);
		} finally {
			savingEntries = false;
		}
	}

	let appId = $state('claude');
	let createdFor = $state('');
	const app = $derived(apps.find((a) => a.id === appId) ?? apps[0]);
	const token = $derived(created && createdFor === app.id ? created.secret : '<token>');
	const localConfig = $derived(JSON.stringify({ mcpServers: { wokko: { command: 'docker', args: ['exec', '-i', 'wokko', '/wokko', 'mcp-stdio', '--token', token] } } }, null, 2));

	async function load() {
		try {
			[tokens, activity] = await Promise.all([api.get<APIToken[]>('/me/tokens'), api.get<AssistantActivity[]>('/me/assistant-activity')]);
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		try {
			created = await api.post<NewAPIToken>('/me/tokens', { name: name.trim() || app.name, scope, expires_in_days: days });
			createdFor = app.id;
			name = '';
			await load();
		} catch (err) {
			error = errorText(err);
		}
	}

	async function revoke(id: number) {
		try {
			await api.del(`/me/tokens/${id}`);
			if (created?.id === id) created = null;
			await load();
		} catch (err) {
			error = errorText(err);
		}
	}

	const scopeLabel = (s: string) => t(`assistant.scope_${s}`);
</script>

<section class="card space-y-3 p-4" aria-labelledby="assistants-title">
	<h2 id="assistants-title" class="flex items-center gap-2 font-semibold"><Bot size={17} class="text-muted" />{t('assistant.title')}</h2>
	<p class="hint !mt-0">{t('assistant.hint')}</p>
	<ErrorNote text={error} />
	{#if !on}
		<p class="rounded-lg bg-surface-2 px-3 py-2 text-sm text-muted">{session.isAdmin ? t('assistant.off_admin') : t('assistant.off')}</p>
	{:else}
		<div>
			<h3 class="label" id="assistant-app">{t('assistant.choose')}</h3>
			<div class="flex flex-wrap gap-2" role="group" aria-labelledby="assistant-app">
				{#each apps as a (a.id)}
					<button type="button" class="chip {a.id === appId ? 'chip-active' : 'hover:bg-surface-2'}" aria-pressed={a.id === appId} onclick={() => (appId = a.id)}>{a.name}</button>
				{/each}
			</div>
		</div>

		<div class="space-y-3 rounded-xl bg-surface-2 p-3">
			<ol class="list-decimal space-y-1 pl-5 text-sm">
				{#each Array.from({ length: app.steps }, (_, n) => n + 1) as n (n)}<li>{t(`assistant.${app.id}_${n}`)}</li>{/each}
			</ol>
			{#if app.byAddress}
				{#if app.link}<a class="btn btn-primary w-full" href={app.link('')} target="_blank" rel="noopener noreferrer">{t('assistant.add_to', { app: app.name })}</a>{/if}
				<CopyField value={address} label={t('assistant.setup_address')} />
				{#if !secure}<p class="flex items-start gap-2 text-sm text-warn"><TriangleAlert size={15} class="mt-0.5 shrink-0" />{t('assistant.needs_https')}</p>{/if}
			{:else}
				{#if app.id === 'other'}<CopyField value={address} label={t('assistant.setup_address')} />{/if}
				<form class="space-y-2" onsubmit={create}>
					<div class="grid gap-2 sm:grid-cols-3">
						<div>
							<label class="label" for="token-name">{t('assistant.name')}</label>
							<input id="token-name" class="input" bind:value={name} maxlength="100" placeholder={app.name} />
						</div>
						<div>
							<label class="label" for="token-scope">{t('assistant.scope')}</label>
							<select id="token-scope" class="input" bind:value={scope}>
								<option value="read">{t('assistant.scope_read')}</option>
								<option value="write">{t('assistant.scope_write')}</option>
								<option value="delete">{t('assistant.scope_delete')}</option>
							</select>
						</div>
						<div>
							<label class="label" for="token-days">{t('assistant.expires')}</label>
							<select id="token-days" class="input" bind:value={days}>
								<option value={0}>{t('assistant.expires_never')}</option>
								<option value={30}>{t('assistant.expires_30')}</option>
								<option value={365}>{t('assistant.expires_365')}</option>
							</select>
						</div>
					</div>
					<button class="btn btn-soft w-full">{t('assistant.new')}</button>
				</form>
				{#if created && createdFor === app.id}
					<div class="rise rounded-xl bg-warn-soft p-3 text-sm text-warn">
						<p class="font-semibold">{t('assistant.secret_title')}</p>
						<p>{t('assistant.secret_hint')}</p>
					</div>
				{/if}
				{#if app.link && created && createdFor === app.id}
					<a class="btn btn-primary w-full" href={app.link(created.secret)}>{t('assistant.add_to', { app: app.name })}</a>
					<p class="hint !mt-0">{t('assistant.or_by_hand')}</p>
				{/if}
				{#if app.snippet}
					<div>
						<p class="label">{app.id === 'other' ? t('assistant.setup_header') : t('assistant.setup_run')}</p>
						{#if app.snippet(token).includes('\n')}
							<pre class="overflow-x-auto rounded-lg bg-surface p-2 text-xs">{app.snippet(token)}</pre>
						{:else}
							<CopyField value={app.snippet(token)} label={app.name} />
						{/if}
					</div>
				{/if}
				{#if app.id === 'other'}
					<details>
						<summary class="min-h-9 cursor-pointer text-sm font-medium">{t('assistant.setup_local')}</summary>
						<p class="hint !mt-0">{t('assistant.setup_local_hint')}</p>
						<pre class="mt-2 overflow-x-auto rounded-lg bg-surface p-2 text-xs">{localConfig}</pre>
					</details>
				{/if}
			{/if}
			{#if app.doc}<a class="inline-flex min-h-9 items-center gap-1.5 text-sm font-medium text-muted hover:text-ink" href={app.doc} target="_blank" rel="noopener noreferrer"><ExternalLink size={14} />{t('assistant.their_guide', { app: app.name })}</a>{/if}
		</div>
		<div>
			<h3 class="label" id="assistant-new-entries">{t('assistant.new_entries')}</h3>
			<div class="grid grid-cols-3 gap-1 rounded-xl bg-surface-2 p-1" role="group" aria-labelledby="assistant-new-entries">
				{#each newEntryChoices as c (c)}
					<button type="button" class="rounded-lg py-2 text-sm font-medium transition-colors {session.user?.assistant_new_entries === c ? 'bg-surface shadow-card' : 'text-muted'}" aria-pressed={session.user?.assistant_new_entries === c} disabled={savingEntries} onclick={() => setNewEntries(c)}>{t(`assistant.new_entries_${c}`)}</button>
				{/each}
			</div>
			<p class="hint">{t(`assistant.new_entries_${session.user?.assistant_new_entries ?? 'ask'}_hint`)}</p>
		</div>
		{#if tokens.length}
			<div>
				<h3 class="label">{t('assistant.tokens')}</h3>
				<ul class="divide-y divide-line">
					{#each tokens as tk (tk.id)}
						<li class="flex min-h-11 items-center justify-between gap-2 text-sm">
							<span class="min-w-0">
								<span class="font-medium">{tk.name}</span>
								<span class="block text-xs text-muted">{tk.is_app ? `${t('assistant.is_app')} · ` : ''}{scopeLabel(tk.scope)} · {tk.last_used_at ? t('assistant.last_used', { date: formatDate(tk.last_used_at) }) : t('assistant.never_used')}</span>
							</span>
							<button class="btn btn-ghost btn-icon shrink-0 text-danger" aria-label={t('assistant.revoke', { name: tk.name })} onclick={() => revoke(tk.id)}><Trash2 size={16} /></button>
						</li>
					{/each}
				</ul>
			</div>
		{/if}
		<details>
			<summary class="min-h-9 cursor-pointer text-sm font-medium">{t('assistant.activity')}</summary>
			{#if activity.length}
				<ul class="mt-1 divide-y divide-line text-xs">
					{#each activity as a (a.id)}
						<li class="py-1.5"><span class="font-medium">{a.tool}</span> {a.target}<span class="block text-muted">{a.token_name} · {formatDate(a.created_at, 'datetime')}</span></li>
					{/each}
				</ul>
			{:else}
				<p class="py-2 text-xs text-muted">{t('assistant.activity_empty')}</p>
			{/if}
		</details>
	{/if}
</section>
