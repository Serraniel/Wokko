<script lang="ts">
	let { text }: { text: string } = $props();
</script>

{#if text}
	<p role="alert" class="rounded-xl bg-danger-soft px-3 py-2 text-sm font-medium text-danger">{text}</p>
{/if}
