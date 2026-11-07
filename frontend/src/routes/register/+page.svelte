<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import type { User } from '#lib/api/types.gen.ts';
	import AuthCard from '#lib/components/AuthCard.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { currentLocale, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	let handle = $state('');
	let displayName = $state('');
	let password = $state('');
	let invite = $state(page.url.searchParams.get('invite') ?? '');
	let error = $state('');
	let busy = $state(false);

	// Someone with an account who opens a household invite joins from their people page.
	$effect(() => {
		if (session.user) goto(invite ? `/people?join=${encodeURIComponent(invite)}` : '/recipes', { replaceState: true });
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			const user = await api.post<User>('/auth/register', { handle, display_name: displayName, password, invite, locale: currentLocale() });
			invite = '';
			await session.signedIn(user);
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}
</script>

<AuthCard title={t('auth.register_title')}>
	<form class="space-y-4" onsubmit={submit}>
		<ErrorNote text={error} />
		<div>
			<label class="label" for="invite">{t('auth.invite')}</label>
			<input id="invite" class="input" bind:value={invite} required={!session.status?.open_registration} autocomplete="off" autocapitalize="off" spellcheck="false" />
			<p class="hint">{session.status?.open_registration ? t('auth.invite_optional') : t('auth.invite_required')}</p>
		</div>
		<div>
			<label class="label" for="handle">{t('auth.handle')}</label>
			<!-- svelte-ignore a11y_autofocus -->
			<input id="handle" class="input" bind:value={handle} required autofocus autocomplete="username" autocapitalize="off" spellcheck="false" />
			<p class="hint">{t('auth.handle_hint')}</p>
		</div>
		<div>
			<label class="label" for="display">{t('auth.display_name')}</label>
			<input id="display" class="input" bind:value={displayName} autocomplete="name" />
		</div>
		<div>
			<label class="label" for="password">{t('auth.password')}</label>
			<input id="password" class="input" type="password" bind:value={password} required minlength="8" autocomplete="new-password" />
			<p class="hint">{t('auth.password_hint')}</p>
		</div>
		<button class="btn btn-primary w-full" disabled={busy}>{busy ? t('common.saving') : t('auth.register')}</button>
	</form>
	<p class="text-center text-sm"><a class="font-semibold text-brand" href="/login">{t('auth.have_account')}</a></p>
</AuthCard>
