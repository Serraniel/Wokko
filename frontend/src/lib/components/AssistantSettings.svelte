<script lang="ts">
	import { Settings2, Trash2 } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { APIToken, AssistantActivity, User } from '#lib/api/types.gen.ts';
	import { formatDate, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';

	// What applies to every assistant: whether they may add to the catalog, every token, and
	// what they did.
	let { tokens, activity, appName, onchange }: { tokens: APIToken[]; activity: AssistantActivity[]; appName: (id: string) => string; onchange: () => void } = $props();

	let error = $state('');
	let saving = $state(false);
	const choices = ['never', 'ask', 'allow'] as const;
	const scopeLabel = (s: string) => t(`assistant.scope_${s}`);

	async function setNewEntries(choice: string) {
		saving = true;
		error = '';
		try {
			session.user = await api.patch<User>('/me', { assistant_new_entries: choice });
		} catch (e) {
			error = errorText(e);
		} finally {
			saving = false;
		}
	}

	async function revoke(id: number) {
		error = '';
		try {
			await api.del(`/me/tokens/${id}`);
			onchange();
		} catch (err) {
			error = errorText(err);
		}
	}
</script>

<section class="card space-y-4 p-4" aria-labelledby="assistant-settings-title">
	<h3 id="assistant-settings-title" class="flex items-center gap-2 font-semibold"><Settings2 size={17} class="text-muted" />{t('assistant.settings_title')}</h3>
	<ErrorNote text={error} />
	<div>
		<h4 class="label" id="assistant-new-entries">{t('assistant.new_entries')}</h4>
		<div class="grid grid-cols-3 gap-1 rounded-xl bg-surface-2 p-1" role="group" aria-labelledby="assistant-new-entries">
			{#each choices as c (c)}
				<button type="button" class="rounded-lg py-2 text-sm font-medium transition-colors {session.user?.assistant_new_entries === c ? 'bg-surface shadow-card' : 'text-muted'}" aria-pressed={session.user?.assistant_new_entries === c} disabled={saving} onclick={() => setNewEntries(c)}>{t(`assistant.new_entries_${c}`)}</button>
			{/each}
		</div>
		<p class="hint">{t(`assistant.new_entries_${session.user?.assistant_new_entries ?? 'ask'}_hint`)}</p>
	</div>
	{#if tokens.length}
		<div>
			<h4 class="label">{t('assistant.tokens')}</h4>
			<ul class="divide-y divide-line">
				{#each tokens as tk (tk.id)}
					<li class="flex min-h-11 items-center justify-between gap-2 text-sm">
						<span class="min-w-0">
							<span class="font-medium">{tk.name}</span>
							<span class="block text-xs text-muted">{appName(tk.app)} · {tk.is_app ? `${t('assistant.is_app')} · ` : ''}{scopeLabel(tk.scope)} · {tk.last_used_at ? t('assistant.last_used', { date: formatDate(tk.last_used_at) }) : t('assistant.never_used')}</span>
						</span>
						<button type="button" class="btn btn-ghost btn-icon shrink-0 text-danger" aria-label={t('assistant.revoke', { name: tk.name })} onclick={() => revoke(tk.id)}><Trash2 size={16} /></button>
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
</section>
