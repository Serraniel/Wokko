<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { api } from '$lib/api/client';
  import Logo from './Logo.svelte';
  import ThemeToggle from './ThemeToggle.svelte';
  import { Menu, X } from 'lucide-svelte';

  let menuOpen = $state(false);

  async function logout() {
    await authStore.clearAuth();
    await api.logout().catch(() => {});
    goto('/auth/login');
  }
</script>

<nav class="border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
  <div class="max-w-7xl mx-auto px-4 flex items-center justify-between h-16">
    <a href="/recipes" class="flex items-center gap-2 font-bold text-lg">
      <Logo size={32} />
      Wokko
    </a>
    <div class="hidden md:flex items-center gap-4">
      <a href="/recipes" class="text-gray-600 dark:text-gray-300 hover:text-wokko-red transition-colors">Recipes</a>
      {#if authStore.isAdmin}
        <a href="/admin" class="text-gray-600 dark:text-gray-300 hover:text-wokko-red transition-colors">Admin</a>
      {/if}
      <ThemeToggle />
      <button onclick={logout} class="text-sm text-gray-500 dark:text-gray-400 hover:text-wokko-red transition-colors">
        Sign out
      </button>
    </div>
    <button class="md:hidden p-2" onclick={() => menuOpen = !menuOpen}>
      {#if menuOpen}<X class="w-5 h-5" />{:else}<Menu class="w-5 h-5" />{/if}
    </button>
  </div>
  {#if menuOpen}
    <div class="md:hidden border-t border-gray-200 dark:border-gray-700 py-2 px-4 flex flex-col gap-2">
      <a href="/recipes" onclick={() => menuOpen = false} class="py-2 text-gray-700 dark:text-gray-300">Recipes</a>
      {#if authStore.isAdmin}
        <a href="/admin" onclick={() => menuOpen = false} class="py-2 text-gray-700 dark:text-gray-300">Admin</a>
      {/if}
      <div class="flex items-center justify-between py-2">
        <span class="text-gray-600 dark:text-gray-300 text-sm">Theme</span>
        <ThemeToggle />
      </div>
      <button onclick={logout} class="py-2 text-left text-sm text-gray-500 dark:text-gray-400">Sign out</button>
    </div>
  {/if}
</nav>
