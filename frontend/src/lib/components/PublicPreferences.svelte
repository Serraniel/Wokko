<script lang="ts">
	import { Languages } from '@lucide/svelte';
	import { chooseLocale, currentLocale, languages, t } from '#lib/i18n/index.svelte.ts';
	import { theme, type ThemeChoice } from '#lib/theme.svelte.ts';

	// Language and theme for people who aren't logged in. Logged-in people use Settings.
	let menu = $state<HTMLDetailsElement>();
	const themes: ThemeChoice[] = ['system', 'light', 'dark'];

	function closeOutside(e: PointerEvent) {
		if (menu?.open && !menu.contains(e.target as Node)) menu.open = false;
	}
	function closeOnEscape(e: KeyboardEvent) {
		if (e.key === 'Escape' && menu?.open) {
			menu.open = false;
			menu.querySelector('summary')?.focus();
		}
	}
</script>

<svelte:window onpointerdown={closeOutside} onkeydown={closeOnEscape} />

<details class="relative" bind:this={menu}>
	<summary class="btn btn-ghost btn-icon list-none [&::-webkit-details-marker]:hidden" aria-label={t('prefs.title')} title={t('prefs.title')}>
		<Languages size={18} />
	</summary>
	<div class="absolute right-0 z-30 mt-2 w-64 space-y-3 rounded-xl border border-line bg-surface p-3 shadow-card">
		<div>
			<label class="label" for="public-language">{t('settings.language')}</label>
			<select id="public-language" class="input" value={currentLocale()} onchange={(e) => chooseLocale(e.currentTarget.value)}>
				{#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
			</select>
		</div>
		<div>
			<p class="label">{t('settings.theme')}</p>
			<div class="grid grid-cols-3 gap-1 rounded-xl bg-surface-2 p-1" role="group" aria-label={t('settings.theme')}>
				{#each themes as id (id)}
					<button type="button" class="rounded-lg py-1.5 text-sm font-medium transition-colors {theme.choice === id ? 'bg-surface shadow-card' : 'text-muted'}" aria-pressed={theme.choice === id} onclick={() => theme.set(id)}>{t(`settings.theme_${id}`)}</button>
				{/each}
			</div>
		</div>
	</div>
</details>
