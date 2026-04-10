<script lang="ts">
  import Footer from '$lib/components/Footer.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import ThemeToggle from '$lib/components/ThemeToggle.svelte';
  let { children } = $props();
</script>
<nav class="border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
  <div class="max-w-7xl mx-auto px-4 flex items-center justify-between h-16">
    <a href="/" class="flex items-center gap-2 font-bold text-lg"><Logo size={32} />Wokko</a>
    <div class="flex items-center gap-3"><ThemeToggle /><a href="/auth/login" class="text-sm text-gray-600 dark:text-gray-300 hover:text-wokko-red">Sign in</a></div>
  </div>
</nav>
<main class="flex-1 max-w-7xl mx-auto w-full px-4 py-8">{@render children()}</main>
<Footer />
