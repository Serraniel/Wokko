<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api, type RecipeDetail } from '$lib/api/client';
  import Footer from '$lib/components/Footer.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import ThemeToggle from '$lib/components/ThemeToggle.svelte';
  import { Clock, Users } from 'lucide-svelte';

  let recipe = $state<RecipeDetail | null>(null);
  let loading = $state(true);
  const API_BASE = import.meta.env.PUBLIC_API_URL || '/api';

  onMount(async () => {
    try {
      recipe = await api.getPublicRecipe($page.params.id!);
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

  <main class="flex-1 max-w-3xl mx-auto w-full px-4 py-8">
    {#if loading}
      <div class="flex justify-center py-20">
        <div class="animate-spin w-8 h-8 border-2 border-wokko-red border-t-transparent rounded-full"></div>
      </div>
    {:else if recipe}
      <div class="space-y-8">
        {#if recipe.photo_path}
          <img src="{API_BASE}/media/{recipe.photo_path}" alt={recipe.name} class="w-full h-64 object-cover rounded-2xl" />
        {/if}
        <div>
          <h1 class="text-3xl font-bold text-gray-900 dark:text-white">{recipe.name}</h1>
          {#if recipe.description}<p class="text-gray-600 dark:text-gray-400 mt-2">{recipe.description}</p>{/if}
          <div class="flex items-center gap-4 mt-3 text-sm text-gray-500 dark:text-gray-400">
            {#if recipe.work_time_min}<span class="flex items-center gap-1"><Clock class="w-4 h-4" />{recipe.work_time_min} min</span>{/if}
            {#if recipe.servings}<span class="flex items-center gap-1"><Users class="w-4 h-4" />{recipe.servings} servings</span>{/if}
          </div>
        </div>
        {#if recipe.steps?.length}
          <div class="space-y-4">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">Instructions</h2>
            {#each recipe.steps as step}
              <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
                <div class="flex items-center gap-3 mb-3">
                  <span class="w-8 h-8 rounded-full bg-wokko-red text-white flex items-center justify-center text-sm font-bold flex-shrink-0">{step.step_order}</span>
                  <h3 class="font-semibold text-gray-900 dark:text-white">Step {step.step_order}</h3>
                </div>
                <p class="text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{step.description}</p>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {:else}
      <div class="text-center py-20 text-gray-500">Recipe not found.</div>
    {/if}
  </main>

  <Footer />
</div>
