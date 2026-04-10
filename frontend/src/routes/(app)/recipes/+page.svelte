<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type Recipe } from '$lib/api/client';
  import RecipeCard from '$lib/components/RecipeCard.svelte';
  import Button from '$lib/components/Button.svelte';
  import { Plus, Search } from 'lucide-svelte';

  let recipes = $state<Recipe[]>([]);
  let loading = $state(true);
  let search = $state('');

  onMount(async () => {
    try {
      recipes = await api.listRecipes({ limit: 50 });
    } finally {
      loading = false;
    }
  });

  let filtered = $derived(
    search
      ? recipes.filter(r => r.name.toLowerCase().includes(search.toLowerCase()))
      : recipes
  );
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between gap-4 flex-wrap">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white">My Recipes</h1>
    <a href="/recipes/new">
      <Button><Plus class="w-4 h-4" /> New Recipe</Button>
    </a>
  </div>

  <div class="relative">
    <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
    <input
      bind:value={search}
      placeholder="Search recipes..."
      class="w-full max-w-sm pl-9 pr-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
             bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm
             focus:outline-none focus:ring-2 focus:ring-wokko-red"
    />
  </div>

  {#if loading}
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {#each Array(8) as _}
        <div class="rounded-xl bg-gray-200 dark:bg-gray-700 animate-pulse h-64"></div>
      {/each}
    </div>
  {:else if filtered.length === 0}
    <div class="text-center py-20">
      <div class="text-6xl mb-4">🍳</div>
      <h2 class="text-xl font-semibold text-gray-700 dark:text-gray-300 mb-2">
        {search ? 'No recipes match your search' : 'No recipes yet'}
      </h2>
      {#if !search}
        <p class="text-gray-500 dark:text-gray-400 mb-6">Create your first recipe to get started!</p>
        <a href="/recipes/new"><Button>+ Create Recipe</Button></a>
      {/if}
    </div>
  {:else}
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {#each filtered as recipe}
        <RecipeCard {recipe} />
      {/each}
    </div>
  {/if}
</div>
