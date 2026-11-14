<script lang="ts">
	import { api, errorText } from '#lib/api/client.ts';
	import AuthCard from '#lib/components/AuthCard.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let login = $state('');
	let sent = $state(false);
	let busy = $state(false);
	let error = $state('');

	// The answer is the same whether or not the account exists.
	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			await api.post('/auth/forgot', { login: login.trim() });
			sent = true;
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>{t('forgot.title')} · Wokko</title></svelte:head>

<AuthCard title={t('forgot.title')}>
	{#if sent}
		<p class="text-center" role="status">{t('forgot.sent')}</p>
	{:else}
		<form class="space-y-4" onsubmit={submit}>
			<ErrorNote text={error} />
			<p class="hint !mt-0">{t('forgot.hint')}</p>
			<div>
				<label class="label" for="login">{t('forgot.login')}</label>
				<!-- svelte-ignore a11y_autofocus -->
				<input id="login" class="input" bind:value={login} required autofocus autocomplete="username" autocapitalize="off" spellcheck="false" />
			</div>
			<button class="btn btn-primary w-full" disabled={busy || !login.trim()}>{t('forgot.send')}</button>
		</form>
	{/if}
	<p class="text-center text-sm"><a class="font-semibold text-brand" href="/login">{t('auth.login')}</a></p>
</AuthCard>
