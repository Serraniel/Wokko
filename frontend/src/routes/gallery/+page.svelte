<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type Recipe } from '$lib/api/client';
  import RecipeCard from '$lib/components/RecipeCard.svelte';
  import Footer from '$lib/components/Footer.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import ThemeToggle from '$lib/components/ThemeToggle.svelte';

  let recipes = $state<Recipe[]>([]);
  let loading = $state(true);

  onMount(async () => {
    try {
      recipes = await api.listPublicRecipes({ limit: 24 });
    } finally {
      loading = false;
    }
  });
</script>

<div class="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-950">
  <nav class="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 px-4 h-16 flex items-center justify-between">
    <a href="/" class="flex items-center gap-2 font-bold text-xl text-wokko-red">
      <Logo size={32} />Wokko
    </a>
    <div class="flex items-center gap-3">
      <ThemeToggle />
      <a href="/auth/login" class="text-sm text-gray-600 dark:text-gray-300 hover:text-wokko-red transition-colors">Sign in</a>
    </div>
  </nav>

  <main class="flex-1 max-w-7xl mx-auto w-full px-4 py-12">
    <div class="text-center mb-10">
      <h1 class="text-4xl font-bold text-gray-900 dark:text-white mb-3">Community Recipes</h1>
      <p class="text-lg text-gray-500 dark:text-gray-400">Delicious recipes shared by our community</p>
    </div>

    {#if loading}
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {#each Array(8) as _}
          <div class="rounded-xl bg-gray-200 dark:bg-gray-700 animate-pulse h-64"></div>
        {/each}
      </div>
    {:else if recipes.length === 0}
      <div class="text-center py-20 text-gray-500 dark:text-gray-400">
        <div class="text-6xl mb-4">🍳</div>
        <p>No public recipes yet.</p>
      </div>
    {:else}
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {#each recipes as recipe}
          <RecipeCard {recipe} href="/public/{recipe.id}" />
        {/each}
      </div>
    {/if}
  </main>

  <Footer />
</div>
