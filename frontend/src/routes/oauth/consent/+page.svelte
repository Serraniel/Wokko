<script lang="ts">
	import { Bot } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText, query } from '#lib/api/client.ts';
	import type { OAuthRedirect, OAuthRequest } from '#lib/api/types.gen.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';

	const params = page.url.searchParams;
	const clientID = params.get('client_id') ?? '';
	const redirectURI = params.get('redirect_uri') ?? '';
	const stateParam = params.get('state') ?? '';
	const challenge = params.get('code_challenge') ?? '';
	const method = params.get('code_challenge_method') ?? 'S256';

	let info = $state<OAuthRequest | null>(null);
	let error = $state('');
	let busy = $state(false);

	$effect(() => {
		if (!session.ready) return;
		if (!session.user) {
			goto(`/login?next=${encodeURIComponent(page.url.pathname + page.url.search)}`, { replaceState: true });
			return;
		}
		if (method !== 'S256') {
			error = t('err.invalid_request');
			return;
		}
		api.get<OAuthRequest>(`/oauth/request${query({ client_id: clientID, redirect_uri: redirectURI, scope: params.get('scope') ?? 'write' })}`)
			.then((r) => (info = r))
			.catch((e) => (error = errorText(e)));
	});

	// The app asks for a level of access, and the person can always give less, or more.
	let chosen = $state('write');
	$effect(() => {
		if (info) chosen = info.scope;
	});

	async function allow() {
		busy = true;
		error = '';
		try {
			const out = await api.post<OAuthRedirect>('/oauth/approve', { client_id: clientID, redirect_uri: redirectURI, code_challenge: challenge, scope: chosen, state: stateParam });
			location.assign(out.redirect_url);
		} catch (e) {
			error = errorText(e);
			busy = false;
		}
	}

	// The address was checked against what the app registered, when the request was described.
	function deny() {
		const u = new URL(redirectURI);
		u.searchParams.set('error', 'access_denied');
		if (stateParam) u.searchParams.set('state', stateParam);
		location.assign(u.toString());
	}
</script>

<svelte:head><title>{t('oauth.title')} · Wokko</title></svelte:head>

<main class="flex min-h-dvh items-center justify-center p-4">
	<div class="card w-full max-w-sm space-y-4 p-5">
		<div class="flex h-11 w-11 items-center justify-center rounded-xl bg-surface-2 text-muted"><Bot size={22} /></div>
		<h1 class="text-xl font-semibold">{t('oauth.title')}</h1>
		<ErrorNote text={error} />
		{#if info}
			<p>{t('oauth.intro', { name: info.client_name })}</p>
			<fieldset class="space-y-1">
				<legend class="label">{t('oauth.choose')}</legend>
				{#each ['read', 'write', 'delete'] as sc (sc)}
					<label class="flex min-h-11 cursor-pointer items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium {chosen === sc ? 'bg-surface-2' : 'hover:bg-surface-2'}">
						<input type="radio" name="scope" value={sc} class="h-4 w-4 shrink-0 accent-[var(--brand)]" bind:group={chosen} />{t(`oauth.scope_${sc}`)}
					</label>
				{/each}
			</fieldset>
			<p class="hint !mt-0">{t('oauth.never')}</p>
			<p class="hint !mt-0">{t('oauth.after', { host: info.redirect_host })}</p>
			<div class="flex gap-2">
				<button class="btn btn-soft flex-1" onclick={deny}>{t('oauth.deny')}</button>
				<button class="btn btn-primary flex-1" disabled={busy} onclick={allow}>{t('oauth.allow')}</button>
			</div>
		{/if}
	</div>
</main>
