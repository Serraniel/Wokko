<script lang="ts">
	import { Mail } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { User } from '#lib/api/types.gen.ts';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';

	// An optional address for mail from this server, used once confirmed.
	let email = $state(session.user?.email ?? '');
	let error = $state('');
	let busy = $state(false);

	async function save(e?: SubmitEvent, address = email.trim()) {
		e?.preventDefault();
		busy = true;
		error = '';
		try {
			session.user = await api.put<User>('/me/email', { email: address });
			email = session.user.email;
			toasts.show(address ? t('email.sent') : t('email.removed'));
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}
</script>

{#if session.status?.mail_enabled || session.user?.email}
	<section class="card space-y-3 p-4" aria-labelledby="email-title">
		<h2 id="email-title" class="flex items-center gap-2 text-lg font-semibold"><Mail size={18} class="text-muted" />{t('email.title')}</h2>
		<p class="hint !mt-0">{t('email.hint')}</p>
		<ErrorNote text={error} />
		<form class="flex items-end gap-2" onsubmit={save}>
			<div class="min-w-0 flex-1">
				<label class="label" for="email">{t('email.address')}</label>
				<input id="email" class="input" type="email" bind:value={email} autocomplete="email" />
			</div>
			<button class="btn btn-soft shrink-0" disabled={busy || !email.trim() || email.trim() === session.user?.email}>{t('common.save')}</button>
		</form>
		{#if session.user?.email}
			<p class="flex flex-wrap items-center gap-2 text-sm">
				{#if session.user.email_verified}<span class="chip !min-h-6 bg-ok-soft text-xs text-ok">{t('email.confirmed')}</span>{:else}<span class="chip !min-h-6 bg-warn-soft text-xs text-warn">{t('email.waiting')}</span>{/if}
				<button type="button" class="btn btn-ghost !min-h-8 text-sm text-danger" disabled={busy} onclick={() => save(undefined, '')}>{t('email.remove')}</button>
			</p>
		{/if}
	</section>
{/if}
