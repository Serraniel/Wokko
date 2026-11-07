<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import type { User } from '#lib/api/types.gen.ts';
	import AuthCard from '#lib/components/AuthCard.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	let handle = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	function target(): string {
		const next = page.url.searchParams.get('next') ?? '';
		// Only follow paths on this site, never another address someone put in a link.
		return next.startsWith('/') && !next.startsWith('//') ? next : '/recipes';
	}

	$effect(() => {
		if (session.user) goto(target(), { replaceState: true });
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			await session.signedIn(await api.post<User>('/auth/login', { handle, password }));
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}
</script>

<AuthCard title={t('auth.welcome')}>
	<form class="space-y-4" onsubmit={submit}>
		<ErrorNote text={error} />
		<div>
			<label class="label" for="handle">{t('auth.handle')}</label>
			<!-- svelte-ignore a11y_autofocus -->
			<input id="handle" class="input" bind:value={handle} required autofocus autocomplete="username" autocapitalize="off" spellcheck="false" />
		</div>
		<div>
			<label class="label" for="password">{t('auth.password')}</label>
			<input id="password" class="input" type="password" bind:value={password} required autocomplete="current-password" />
		</div>
		<button class="btn btn-primary w-full" disabled={busy}>{busy ? t('auth.logging_in') : t('auth.login')}</button>
	</form>
	<div class="space-y-1 text-center text-sm text-muted">
		<p><a class="font-semibold text-brand" href="/register">{t('auth.no_account')}</a></p>
		{#if session.status?.allow_public}<p><a class="font-semibold text-brand" href="/gallery">{t('gallery.title')}</a></p>{/if}
		<p>{t('auth.forgot_hint')}</p>
	</div>
</AuthCard>
