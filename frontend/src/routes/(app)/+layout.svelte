<script lang="ts">
  import { browser } from '$app/environment';
  import { authStore } from '$lib/stores/auth.svelte';
  import Nav from '$lib/components/Nav.svelte';
  import Footer from '$lib/components/Footer.svelte';

  // Initialize synchronously so nav and child page guards have auth state immediately
  if (browser) {
    authStore.init();
  }

  let { children } = $props();
</script>

<div class="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-950">
  <Nav />
  <main class="flex-1 max-w-7xl mx-auto w-full px-4 py-8">
    {@render children()}
  </main>
  <Footer />
</div>
