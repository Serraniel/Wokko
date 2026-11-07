<script lang="ts">
	import { Download, LogOut, Monitor, Moon, Shield, Sun, Trash2 } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { CollectionLink, User } from '#lib/api/types.gen.ts';
	import { forget } from '#lib/cache.ts';
	import CopyField from '#lib/components/CopyField.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { prefs } from '#lib/prefs.svelte.ts';
	import { currentLocale, languages, setLocale, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { theme, type ThemeChoice } from '#lib/theme.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let displayName = $state(session.user?.display_name ?? '');
	let current = $state('');
	let next = $state('');
	let error = $state('');
	let pwError = $state('');

	let links = $state<CollectionLink[]>([]);
	api.get<CollectionLink[]>('/me/collection-links').then((l) => (links = l)).catch(() => {});

	async function createLink() {
		try {
			links = [...links, await api.post<CollectionLink>('/me/collection-links', {})];
		} catch (e) {
			toasts.error(errorText(e));
		}
	}

	async function revokeLink(id: number) {
		try {
			await api.del(`/me/collection-links/${id}`);
			links = links.filter((l) => l.id !== id);
		} catch (e) {
			toasts.error(errorText(e));
		}
	}

	const themes: { id: ThemeChoice; icon: typeof Sun; key: string }[] = [
		{ id: 'system', icon: Monitor, key: 'settings.theme_system' },
		{ id: 'light', icon: Sun, key: 'settings.theme_light' },
		{ id: 'dark', icon: Moon, key: 'settings.theme_dark' }
	];

	async function changeLanguage(e: Event) {
		const code = (e.currentTarget as HTMLSelectElement).value;
		error = '';
		await setLocale(code);
		try {
			session.user = await api.patch<User>('/me', { locale: code });
			forget(); // Ingredient names come back in the new language.
		} catch (err) {
			error = errorText(err);
		}
	}

	async function saveName(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		try {
			session.user = await api.patch<User>('/me', { display_name: displayName });
			toasts.show(t('settings.saved'));
		} catch (err) {
			error = errorText(err);
		}
	}

	async function savePassword(e: SubmitEvent) {
		e.preventDefault();
		pwError = '';
		try {
			await api.post('/me/password', { current, next });
			current = '';
			next = '';
			toasts.show(t('settings.password_changed'));
		} catch (err) {
			pwError = errorText(err);
		}
	}

	async function logout() {
		await api.post('/auth/logout').catch(() => {});
		forget();
		session.signedOut();
	}
</script>

<svelte:head><title>{t('nav.settings')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-4 p-4">
	<h1 class="text-2xl font-bold">{t('nav.settings')}</h1>
	<ErrorNote text={error} />

	<section class="card space-y-4 p-4">
		<div>
			<label class="label" for="language">{t('settings.language')}</label>
			<select id="language" class="input" value={currentLocale()} onchange={changeLanguage}>
				{#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
			</select>
		</div>
		<div>
			<p class="label">{t('settings.theme')}</p>
			<div class="grid grid-cols-3 gap-1 rounded-xl bg-surface-2 p-1" role="group" aria-label={t('settings.theme')}>
				{#each themes as item (item.id)}
					{@const Icon = item.icon}
					<button class="flex flex-col items-center gap-1 rounded-lg py-2 text-sm font-medium transition-colors {theme.choice === item.id ? 'bg-surface shadow-card' : 'text-muted'}" aria-pressed={theme.choice === item.id} onclick={() => theme.set(item.id)}><Icon size={18} />{t(item.key)}</button>
				{/each}
			</div>
		</div>
	</section>

	<section class="card space-y-3 p-4">
		<h2 class="text-lg font-semibold">{t('settings.profile')}</h2>
		<p class="text-sm text-muted">{t('settings.your_handle')} <span class="font-mono font-semibold text-ink">@{session.user?.handle}</span></p>
		<form class="flex items-end gap-2" onsubmit={saveName}>
			<div class="min-w-0 flex-1">
				<label class="label" for="display">{t('auth.display_name')}</label>
				<input id="display" class="input" bind:value={displayName} autocomplete="name" />
			</div>
			<button class="btn btn-soft shrink-0">{t('common.save')}</button>
		</form>
	</section>

	<section class="card space-y-3 p-4">
		<div>
			<label class="label" for="amounts">{t('settings.amounts')}</label>
			<select id="amounts" class="input" value={prefs.amounts} onchange={(e) => prefs.setAmounts(e.currentTarget.value === 'fraction' ? 'fraction' : 'decimal')}>
				<option value="decimal">{t('settings.amounts_decimal')}</option>
				<option value="fraction">{t('settings.amounts_fraction')}</option>
			</select>
		</div>
	</section>

	<section class="card space-y-3 p-4">
		<h2 class="font-semibold">{t('settings.collection')}</h2>
		<p class="hint !mt-0">{t('settings.collection_hint')}</p>
		{#each links as l (l.id)}
			<div class="flex items-center gap-2">
				<div class="min-w-0 flex-1"><CopyField value={l.url} label={t('settings.collection_label')} /></div>
				<button class="btn btn-ghost btn-icon shrink-0 text-danger" aria-label={t('share.revoke')} onclick={() => revokeLink(l.id)}><Trash2 size={17} /></button>
			</div>
		{/each}
		{#if !links.length}<button class="btn btn-soft w-full" onclick={createLink}>{t('settings.collection_create')}</button>{/if}
	</section>

	<section class="card p-4">
		<h2 class="mb-3 text-lg font-semibold">{t('settings.change_password')}</h2>
		<form class="space-y-3" onsubmit={savePassword}>
			<ErrorNote text={pwError} />
			<div>
				<label class="label" for="current">{t('settings.current_password')}</label>
				<input id="current" class="input" type="password" bind:value={current} required autocomplete="current-password" />
			</div>
			<div>
				<label class="label" for="next">{t('reset.new_password')}</label>
				<input id="next" class="input" type="password" bind:value={next} required minlength="8" autocomplete="new-password" />
			</div>
			<button class="btn btn-soft w-full">{t('settings.change_password')}</button>
		</form>
	</section>

	<section class="card divide-y divide-line overflow-hidden">
		<a class="flex items-center gap-3 p-4 font-medium hover:bg-surface-2" href="/api/v1/me/export" download data-sveltekit-reload><Download size={20} class="shrink-0 text-muted" /><span><span class="block">{t('settings.export')}</span><span class="block text-sm font-normal text-muted">{t('settings.export_hint')}</span></span></a>
		<a class="flex items-center gap-3 p-4 font-medium hover:bg-surface-2" href="/trash"><Trash2 size={20} class="text-muted" />{t('trash.title')}</a>
		{#if session.isAdmin}<a class="flex items-center gap-3 p-4 font-medium hover:bg-surface-2 md:hidden" href="/admin"><Shield size={20} class="text-muted" />{t('nav.admin')}</a>{/if}
		<button class="flex w-full items-center gap-3 p-4 text-left font-medium hover:bg-surface-2" onclick={logout}><LogOut size={20} class="text-muted" />{t('auth.logout')}</button>
	</section>

	<footer class="flex flex-wrap justify-center gap-x-3 gap-y-1 py-2 text-sm text-muted">
		<a class="hover:underline" href="https://github.com/Serraniel/Wokko" rel="noopener noreferrer" target="_blank">GitHub</a>
		<span>·</span><span>v{session.status?.version}</span>
		<span>·</span><a class="hover:underline" href="https://github.com/Serraniel/Wokko/releases" rel="noopener noreferrer" target="_blank">{t('settings.changelog')}</a>
		<span>·</span><a class="hover:underline" href="https://eupl.eu/1.2/" rel="noopener noreferrer" target="_blank">EUPL-1.2</a>
		<span>·</span><a class="hover:underline" href="/third-party-notices.txt" target="_blank" data-sveltekit-reload>{t('settings.notices')}</a>
	</footer>
</div>
