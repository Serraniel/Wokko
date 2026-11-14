<script lang="ts">
	import { Monitor, Moon, Sun } from '@lucide/svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { theme, type ThemeChoice } from '#lib/theme.svelte.ts';

	// One click moves to the next choice. The label says what is active, so it isn't a guess.
	let { withLabel = false, class: cls = '' }: { withLabel?: boolean; class?: string } = $props();

	const order: ThemeChoice[] = ['system', 'light', 'dark'];
	const icons = { system: Monitor, light: Sun, dark: Moon };
	const Icon = $derived(icons[theme.choice]);
	const label = $derived(t('theme.switch', { theme: t(`settings.theme_${theme.choice}`) }));

	function next() {
		theme.set(order[(order.indexOf(theme.choice) + 1) % order.length]);
	}
</script>

<button type="button" class={cls} aria-label={label} title={label} onclick={next}>
	<Icon size={17} />{#if withLabel}<span class="flex-1 text-left">{label}</span>{/if}
</button>
