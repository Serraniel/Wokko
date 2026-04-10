<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import Nav from '$lib/components/Nav.svelte';
  import Footer from '$lib/components/Footer.svelte';

  let { children } = $props();

  onMount(() => {
    authStore.init();
    if (!authStore.isLoggedIn) {
      goto('/auth/login');
    }
  });
</script>

<div class="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-950">
  <Nav />
  <main class="flex-1 max-w-7xl mx-auto w-full px-4 py-8">
    {@render children()}
  </main>
  <Footer />
</div>
