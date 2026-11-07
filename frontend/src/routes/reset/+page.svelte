<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import AuthCard from '#lib/components/AuthCard.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			await api.post('/auth/reset', { token: page.url.searchParams.get('token') ?? '', password });
			session.signedOut();
			toasts.show(t('reset.done'));
			await goto('/login', { replaceState: true });
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}
</script>

<AuthCard title={t('reset.title')}>
	<form class="space-y-4" onsubmit={submit}>
		<ErrorNote text={error} />
		<div>
			<label class="label" for="password">{t('reset.new_password')}</label>
			<!-- svelte-ignore a11y_autofocus -->
			<input id="password" class="input" type="password" bind:value={password} required minlength="8" autofocus autocomplete="new-password" />
			<p class="hint">{t('auth.password_hint')}</p>
		</div>
		<button class="btn btn-primary w-full" disabled={busy}>{t('common.save')}</button>
	</form>
</AuthCard>
