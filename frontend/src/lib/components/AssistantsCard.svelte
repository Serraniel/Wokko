<script lang="ts">
	import { Bot, Trash2 } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { APIToken, AssistantActivity, NewAPIToken } from '#lib/api/types.gen.ts';
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
	const header = $derived(`Authorization: Bearer ${created?.secret ?? ''}`);
	const codeCommand = $derived(`claude mcp add --transport http wokko ${address} --header "${header}"`);
	const desktopConfig = $derived(JSON.stringify({ mcpServers: { wokko: { command: 'docker', args: ['exec', '-i', 'wokko', '/wokko', 'mcp-stdio', '--token', created?.secret ?? ''] } } }, null, 2));

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
			created = await api.post<NewAPIToken>('/me/tokens', { name, scope, expires_in_days: days });
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
			<h3 class="label">{t('assistant.setup_web')}</h3>
			<CopyField value={address} label={t('assistant.setup_address')} />
			<p class="hint">{t('assistant.setup_web_hint')}</p>
		</div>
		<h3 class="label !mb-0">{t('assistant.setup_token')}</h3>
		<p class="hint !mt-0">{t('assistant.setup_token_hint')}</p>
		<form class="space-y-2" onsubmit={create}>
			<div>
				<label class="label" for="token-name">{t('assistant.name')}</label>
				<input id="token-name" class="input" bind:value={name} required maxlength="100" placeholder={t('assistant.name_placeholder')} />
			</div>
			<div class="grid gap-2 sm:grid-cols-2">
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
		{#if created}
			<div class="rise space-y-3 rounded-xl bg-warn-soft p-3 text-warn">
				<div>
					<p class="text-sm font-semibold">{t('assistant.secret_title')}</p>
					<p class="text-sm">{t('assistant.secret_hint')}</p>
				</div>
				<CopyField value={created.secret} label={t('assistant.secret_title')} />
				<div class="text-ink">
					<p class="label">{t('assistant.setup_any')}</p>
					<CopyField value={address} label={t('assistant.setup_address')} />
					<div class="mt-2"><CopyField value={header} label={t('assistant.setup_header')} /></div>
				</div>
				<details class="text-ink">
					<summary class="min-h-9 cursor-pointer text-sm font-medium">{t('assistant.setup_examples')}</summary>
					<p class="label mt-2">{t('assistant.setup_code')}</p>
					<CopyField value={codeCommand} label={t('assistant.setup_code')} />
					<p class="label mt-3">{t('assistant.setup_desktop')}</p>
					<pre class="overflow-x-auto rounded-lg bg-surface p-2 text-xs">{desktopConfig}</pre>
				</details>
			</div>
		{/if}
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
