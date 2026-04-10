<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { authStore } from '$lib/stores/auth.svelte';
  import { themeStore } from '$lib/stores/theme.svelte';

  onMount(() => {
    themeStore.init();
    authStore.init();
  });

  let { children } = $props();
</script>

{@render children()}
