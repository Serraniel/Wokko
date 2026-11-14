<script lang="ts">
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import AuthCard from '#lib/components/AuthCard.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	let done = $state(false);
	let error = $state('');

	// Confirms the address the link was mailed to.
	$effect(() => {
		api.post('/auth/confirm-email', { token: page.url.searchParams.get('token') ?? '' })
			.then(async () => {
				done = true;
				if (session.user) await session.load();
			})
			.catch((e) => (error = errorText(e)));
	});
</script>

<svelte:head><title>{t('email.title')} · Wokko</title></svelte:head>

<AuthCard title={t('email.title')}>
	<ErrorNote text={error} />
	{#if done}<p class="text-center" role="status">{t('email.confirmed_done')}</p>{/if}
	<p class="text-center text-sm"><a class="font-semibold text-brand" href={session.user ? '/settings' : '/login'}>{session.user ? t('nav.settings') : t('auth.login')}</a></p>
</AuthCard>
