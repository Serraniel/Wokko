<script lang="ts">
	import { KeyRound, Trash2 } from '@lucide/svelte';
	import { api, ApiError, errorText, query } from '#lib/api/client.ts';
	import type { Passkey } from '#lib/api/types.gen.ts';
	import { formatDate, t } from '#lib/i18n/index.svelte.ts';
	import { createPasskey, passkeysAvailable } from '#lib/passkeys.ts';
	import { toasts } from '#lib/toast.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';

	const available = passkeysAvailable();
	let keys = $state<Passkey[]>([]);
	let name = $state('');
	let error = $state('');
	let busy = $state(false);

	api.get<Passkey[]>('/me/passkeys').then((k) => (keys = k)).catch(() => {});

	async function add(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			const start = await api.post<{ flow: string; options: { publicKey: Record<string, unknown> } }>('/me/passkeys/begin');
			const answer = await createPasskey(start.options);
			keys = [...keys, await api.post<Passkey>(`/me/passkeys/finish${query({ flow: start.flow, name })}`, answer)];
			name = '';
			toasts.show(t('passkey.added'));
		} catch (err) {
			// Closing the browser's own dialog isn't an error worth reporting.
			if (err instanceof ApiError) error = errorText(err);
			else if (!(err instanceof DOMException && err.name === 'NotAllowedError')) error = t('err.passkey_failed');
		} finally {
			busy = false;
		}
	}

	async function remove(id: number) {
		try {
			await api.del(`/me/passkeys/${id}`);
			keys = keys.filter((k) => k.id !== id);
		} catch (err) {
			error = errorText(err);
		}
	}
</script>

<section class="card space-y-3 p-4" aria-labelledby="passkeys-title">
	<h2 id="passkeys-title" class="flex items-center gap-2 font-semibold"><KeyRound size={17} class="text-muted" />{t('passkey.title')}</h2>
	<p class="hint !mt-0">{t('passkey.hint')}</p>
	<ErrorNote text={error} />
	{#if keys.length}
		<ul class="divide-y divide-line">
			{#each keys as k (k.id)}
				<li class="flex min-h-11 items-center justify-between gap-2 text-sm">
					<span><span class="font-medium">{k.name}</span><span class="block text-xs text-muted">{k.last_used_at ? t('passkey.last_used', { date: formatDate(k.last_used_at) }) : t('passkey.never_used')}</span></span>
					<button class="btn btn-ghost btn-icon text-danger" aria-label={t('passkey.remove', { name: k.name })} onclick={() => remove(k.id)}><Trash2 size={16} /></button>
				</li>
			{/each}
		</ul>
	{/if}
	{#if available}
		<form class="flex gap-2" onsubmit={add}>
			<input class="input min-w-0 flex-1" bind:value={name} placeholder={t('passkey.name_placeholder')} aria-label={t('passkey.name')} maxlength="100" />
			<button class="btn btn-soft shrink-0" disabled={busy}>{t('passkey.add')}</button>
		</form>
	{:else}
		<p class="rounded-lg bg-surface-2 px-3 py-2 text-sm text-muted">{t('passkey.needs_https')}</p>
	{/if}
</section>
