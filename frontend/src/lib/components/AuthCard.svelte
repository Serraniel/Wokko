<script lang="ts">
	import type { Snippet } from 'svelte';
	import Logo from './Logo.svelte';

	let { title, subtitle, children }: { title: string; subtitle?: string; children: Snippet } = $props();
</script>

<main class="flex min-h-dvh items-center justify-center p-4">
	<div class="w-full max-w-sm space-y-6">
		<div class="flex flex-col items-center gap-2 text-center">
			<Logo size={56} />
			<h1 class="text-2xl font-bold">{title}</h1>
			{#if subtitle}<p class="text-muted">{subtitle}</p>{/if}
		</div>
		{@render children()}
	</div>
</main>
