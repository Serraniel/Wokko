<script lang="ts">
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let { open = $bindable(false), title, children }: { open?: boolean; title: string; children: Snippet } = $props();
	let dialog = $state<HTMLDialogElement>();

	$effect(() => {
		if (open && dialog && !dialog.open) dialog.showModal();
		else if (!open && dialog?.open) dialog.close();
	});
</script>

<!-- A bottom sheet on phones and a centered dialog on larger screens. -->
<dialog
	bind:this={dialog}
	onclose={() => (open = false)}
	onclick={(e) => e.target === dialog && (open = false)}
	class="m-0 mt-auto max-h-[90dvh] w-full max-w-none rounded-t-2xl border-0 bg-surface p-0 text-ink backdrop:bg-black/50 sm:m-auto sm:max-w-lg sm:rounded-2xl"
>
	{#if open}
		<div class="rise flex max-h-[90dvh] flex-col">
			<header class="flex items-center justify-between gap-2 border-b border-line px-4 py-2">
				<h2 class="text-lg font-semibold">{title}</h2>
				<button class="btn btn-ghost btn-icon" onclick={() => (open = false)} aria-label={t('common.close')}><X size={20} /></button>
			</header>
			<div class="overflow-y-auto p-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
				{@render children()}
			</div>
		</div>
	{/if}
</dialog>
