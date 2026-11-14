<script lang="ts">
	import { Mail, Send } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { MailStatus } from '#lib/api/types.gen.ts';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';

	// Whether the server can send mail, and a test mail to the admin's own address.
	let status = $state<MailStatus | null>(null);
	let error = $state('');
	let busy = $state(false);

	api.get<MailStatus>('/admin/mail')
		.then((s) => (status = s))
		.catch((e) => (error = errorText(e)));

	async function test() {
		busy = true;
		error = '';
		try {
			await api.post('/admin/mail/test');
			toasts.show(t('mail.test_sent'));
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}
</script>

<section class="card space-y-3 p-4" aria-labelledby="mail-title">
	<h2 id="mail-title" class="flex items-center gap-2 text-lg font-semibold"><Mail size={18} class="text-muted" />{t('mail.title')}</h2>
	<ErrorNote text={error} />
	{#if status}
		{#if status.ready}
			<p class="text-sm"><span class="chip !min-h-6 bg-ok-soft text-xs text-ok">{t('mail.ready')}</span> <span class="text-muted">{t('mail.via', { host: status.host, from: status.from })}</span></p>
			<button class="btn btn-soft w-full" disabled={busy} onclick={test}><Send size={16} />{t('mail.test')}</button>
		{:else if status.configured}
			<p class="hint !mt-0">{t('mail.needs_public_url')}</p>
		{:else}
			<p class="hint !mt-0">{t('mail.off')}</p>
		{/if}
	{/if}
</section>
