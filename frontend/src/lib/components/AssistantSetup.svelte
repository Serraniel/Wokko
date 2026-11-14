<script lang="ts">
	import { ExternalLink, Trash2, TriangleAlert } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { APIToken, NewAPIToken } from '#lib/api/types.gen.ts';
	import type { AssistantApp } from '#lib/assistantApps.ts';
	import { formatDate, t } from '#lib/i18n/index.svelte.ts';
	import CopyField from './CopyField.svelte';
	import ErrorNote from './ErrorNote.svelte';

	// Connecting one assistant app, step by step, and managing what's connected for it.
	let { app, address, tokens, onchange }: { app: AssistantApp; address: string; tokens: APIToken[]; onchange: () => void } = $props();

	let created = $state<NewAPIToken | null>(null);
	let name = $state('');
	let scope = $state('write');
	let days = $state(0);
	let error = $state('');
	let busy = $state(false);

	const secure = location.protocol === 'https:';
	const token = $derived(created ? created.secret : '<token>');
	const localConfig = $derived(JSON.stringify({ mcpServers: { wokko: { command: 'docker', args: ['exec', '-i', 'wokko', '/wokko', 'mcp-stdio', '--token', token] } } }, null, 2));
	const scopeLabel = (s: string) => t(`assistant.scope_${s}`);

	async function create(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			created = await api.post<NewAPIToken>('/me/tokens', { name: name.trim() || app.name, scope, expires_in_days: days, app: app.id });
			name = '';
			onchange();
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}

	async function revoke(id: number) {
		error = '';
		try {
			await api.del(`/me/tokens/${id}`);
			if (created?.id === id) created = null;
			onchange();
		} catch (err) {
			error = errorText(err);
		}
	}
</script>

<div class="space-y-4">
	<ErrorNote text={error} />
	<ol class="list-decimal space-y-1 pl-5 text-sm">
		{#each Array.from({ length: app.steps }, (_, n) => n + 1) as n (n)}<li>{t(`assistant.${app.id}_${n}`)}</li>{/each}
	</ol>

	{#if app.byAddress}
		{#if app.link}<a class="btn btn-primary w-full" href={app.link('')} target="_blank" rel="noopener noreferrer">{t('assistant.add_to', { app: app.name })}</a>{/if}
		<CopyField value={address} label={t('assistant.setup_address')} />
		{#if !secure}<p class="flex items-start gap-2 text-sm text-warn"><TriangleAlert size={15} class="mt-0.5 shrink-0" />{t('assistant.needs_https')}</p>{/if}
	{:else}
		{#if app.id === 'other'}<CopyField value={address} label={t('assistant.setup_address')} />{/if}
		<form class="space-y-2 rounded-xl bg-surface-2 p-3" onsubmit={create}>
			<p class="label !mb-0">{t('assistant.step_access')}</p>
			<div class="grid gap-2 sm:grid-cols-2">
				<div class="sm:col-span-2">
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
			<button class="btn btn-soft w-full" disabled={busy}>{t('assistant.new')}</button>
		</form>
		{#if created}
			<div class="rise rounded-xl bg-warn-soft p-3 text-sm text-warn">
				<p class="font-semibold">{t('assistant.secret_title')}</p>
				<p>{t('assistant.secret_hint')}</p>
			</div>
		{/if}
		{#if app.link && created}
			<a class="btn btn-primary w-full" href={app.link(created.secret)}>{t('assistant.add_to', { app: app.name })}</a>
			<p class="hint !mt-0">{t('assistant.or_by_hand')}</p>
		{/if}
		{#if app.snippet}
			<div>
				<p class="label">{app.id === 'other' ? t('assistant.setup_header') : t('assistant.setup_run')}</p>
				{#if app.snippet(token).includes('\n')}
					<pre class="overflow-x-auto rounded-lg bg-surface-2 p-2 text-xs">{app.snippet(token)}</pre>
				{:else}
					<CopyField value={app.snippet(token)} label={app.name} />
				{/if}
			</div>
		{/if}
		{#if app.id === 'other'}
			<details>
				<summary class="min-h-9 cursor-pointer text-sm font-medium">{t('assistant.setup_local')}</summary>
				<p class="hint !mt-0">{t('assistant.setup_local_hint')}</p>
				<pre class="mt-2 overflow-x-auto rounded-lg bg-surface-2 p-2 text-xs">{localConfig}</pre>
			</details>
		{/if}
	{/if}
	{#if app.doc}<a class="inline-flex min-h-9 items-center gap-1.5 text-sm font-medium text-muted hover:text-ink" href={app.doc} target="_blank" rel="noopener noreferrer"><ExternalLink size={14} />{t('assistant.their_guide', { app: app.name })}</a>{/if}

	{#if tokens.length}
		<div class="border-t border-line pt-3">
			<h3 class="label">{t('assistant.connected_here')}</h3>
			<ul class="divide-y divide-line">
				{#each tokens as tk (tk.id)}
					<li class="flex min-h-11 items-center justify-between gap-2 text-sm">
						<span class="min-w-0">
							<span class="font-medium">{tk.name}</span>
							<span class="block text-xs text-muted">{tk.is_app ? `${t('assistant.is_app')} · ` : ''}{scopeLabel(tk.scope)} · {tk.last_used_at ? t('assistant.last_used', { date: formatDate(tk.last_used_at) }) : t('assistant.never_used')}</span>
						</span>
						<button type="button" class="btn btn-ghost shrink-0 text-danger" onclick={() => revoke(tk.id)}><Trash2 size={16} />{t('assistant.disconnect')}</button>
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</div>
