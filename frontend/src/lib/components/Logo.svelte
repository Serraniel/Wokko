<script lang="ts">
	let { size = 32 }: { size?: number } = $props();
</script>

<img src="/logo.svg" alt="" width={size} height={size} class="shrink-0" style="height: {size}px; width: {size}px" />
