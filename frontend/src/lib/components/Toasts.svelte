<script lang="ts">
	import { toasts } from '#lib/toast.svelte.ts';
</script>

<div class="pointer-events-none fixed inset-x-0 bottom-[calc(5.25rem+env(safe-area-inset-bottom))] z-50 flex flex-col items-center gap-2 px-4 md:bottom-6" aria-live="polite">
	{#each toasts.items as toast (toast.id)}
		<div class="rise pointer-events-auto flex max-w-md items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium shadow-card {toast.kind === 'error' ? 'bg-danger text-surface' : 'bg-ink text-bg'}">
			<span>{toast.text}</span>
			{#if toast.action}
				<button
					class="shrink-0 font-bold underline underline-offset-2"
					onclick={() => {
						toast.action?.run();
						toasts.dismiss(toast.id);
					}}>{toast.action.label}</button
				>
			{/if}
		</div>
	{/each}
</div>
