<script lang="ts">
	import { Check, Copy } from '@lucide/svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let { value, label }: { value: string; label: string } = $props();
	let copied = $state(false);
	let input = $state<HTMLInputElement>();

	async function copy() {
		try {
			await navigator.clipboard.writeText(value);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			// The clipboard API needs HTTPS. Selecting the text lets people copy it by hand.
			input?.select();
		}
	}
</script>

<div class="flex gap-2">
	<input bind:this={input} class="input min-w-0 flex-1 font-mono text-sm" readonly {value} aria-label={label} onfocus={(e) => e.currentTarget.select()} />
	<button type="button" class="btn btn-soft btn-icon shrink-0" onclick={copy} aria-label={t('common.copy')}>
		{#if copied}<Check size={18} class="text-ok" />{:else}<Copy size={18} />{/if}
	</button>
</div>
