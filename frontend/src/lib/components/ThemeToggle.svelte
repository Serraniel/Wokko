<script lang="ts">
  import { themeStore } from '$lib/stores/theme.svelte';
  import { Moon, Sun } from 'lucide-svelte';
</script>

<button
  onclick={() => themeStore.toggle()}
  class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors text-gray-700 dark:text-gray-200"
  aria-label="Toggle theme"
>
  {#if themeStore.dark}
    <Sun class="w-5 h-5" />
  {:else}
    <Moon class="w-5 h-5" />
  {/if}
</button>
