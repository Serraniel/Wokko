<script lang="ts">
	import type { Snippet } from 'svelte';
	import AppFooter from './AppFooter.svelte';
	import Logo from './Logo.svelte';
	import PublicPreferences from './PublicPreferences.svelte';

	let { title, subtitle, children }: { title: string; subtitle?: string; children: Snippet } = $props();
</script>

<main class="relative flex min-h-dvh flex-col items-center justify-center p-4">
	<div class="absolute top-[max(0.5rem,env(safe-area-inset-top))] right-2"><PublicPreferences /></div>
	<div class="w-full max-w-sm space-y-6">
		<div class="flex flex-col items-center gap-2 text-center">
			<Logo size={56} />
			<h1 class="text-2xl font-bold">{title}</h1>
			{#if subtitle}<p class="text-muted">{subtitle}</p>{/if}
		</div>
		{@render children()}
	</div>
	<AppFooter />
</main>
