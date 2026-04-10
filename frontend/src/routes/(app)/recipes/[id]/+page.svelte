<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api, type RecipeDetail } from '$lib/api/client';
  import Button from '$lib/components/Button.svelte';
  import { Clock, Users, Share2, Edit, Trash2, Globe, Lock } from 'lucide-svelte';

  let recipe = $state<RecipeDetail | null>(null);
  let loading = $state(true);
  let shareUrl = $state('');
  let sharing = $state(false);

  const API_BASE = import.meta.env.PUBLIC_API_URL || '/api';

  onMount(async () => {
    try {
      recipe = await api.getRecipe($page.params.id!);
    } catch {
      goto('/recipes');
    } finally {
      loading = false;
    }
  });

  async function share() {
    if (!recipe) return;
    sharing = true;
    try {
      const res = await api.createShareLink(recipe.id);
      shareUrl = res.url;
      await navigator.clipboard.writeText(shareUrl).catch(() => {});
    } finally {
      sharing = false;
    }
  }

  async function handleDelete() {
    if (!recipe || !confirm('Delete this recipe?')) return;
    await api.deleteRecipe(recipe.id);
    goto('/recipes');
  }

  let allIngredients = $derived(
    recipe?.steps?.flatMap(s =>
      s.ingredients?.map(i => ({
        ...i,
        stepName: `Step ${s.step_order}`
      })) ?? []
    ) ?? []
  );
</script>

{#if loading}
  <div class="flex justify-center py-20">
    <div class="animate-spin w-8 h-8 border-2 border-wokko-red border-t-transparent rounded-full"></div>
  </div>
{:else if recipe}
  <div class="max-w-3xl mx-auto space-y-8">
    {#if recipe.photo_path}
      <img src="{API_BASE}/media/{recipe.photo_path}" alt={recipe.name}
           class="w-full h-64 object-cover rounded-2xl" />
    {/if}

    <div class="flex items-start justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">{recipe.name}</h1>
        {#if recipe.description}
          <p class="text-gray-600 dark:text-gray-400 mt-2">{recipe.description}</p>
        {/if}
        <div class="flex items-center gap-4 mt-3 text-sm text-gray-500 dark:text-gray-400">
          {#if recipe.work_time_min}<span class="flex items-center gap-1"><Clock class="w-4 h-4" />{recipe.work_time_min} min</span>{/if}
          {#if recipe.servings}<span class="flex items-center gap-1"><Users class="w-4 h-4" />{recipe.servings} servings</span>{/if}
          <span class="flex items-center gap-1">
            {#if recipe.is_public}<Globe class="w-4 h-4 text-green-500" />Public{:else}<Lock class="w-4 h-4" />Private{/if}
          </span>
        </div>
      </div>
      <div class="flex gap-2 flex-wrap">
        <Button variant="secondary" onclick={share} loading={sharing}><Share2 class="w-4 h-4" /> Share</Button>
        <a href="/recipes/{recipe.id}/edit"><Button variant="secondary"><Edit class="w-4 h-4" /> Edit</Button></a>
        <Button variant="danger" onclick={handleDelete}><Trash2 class="w-4 h-4" /></Button>
      </div>
    </div>

    {#if shareUrl}
      <div class="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-3 text-sm text-green-800 dark:text-green-200">
        Share link copied: <code class="font-mono text-xs">{shareUrl}</code>
      </div>
    {/if}

    {#if recipe.tags?.length}
      <div class="flex flex-wrap gap-2">
        {#each recipe.tags as tag}
          <span class="px-3 py-1 rounded-full bg-gray-100 dark:bg-gray-700 text-sm text-gray-700 dark:text-gray-300">{tag}</span>
        {/each}
      </div>
    {/if}

    {#if allIngredients.length > 0}
      <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-4">Ingredients</h2>
        <ul class="space-y-2">
          {#each allIngredients as ing}
            <li class="flex items-center gap-2 text-gray-700 dark:text-gray-300">
              <span class="w-2 h-2 rounded-full bg-wokko-red flex-shrink-0"></span>
              {#if ing.amount}{ing.amount}{/if}
              {#if ing.unit_abbr || ing.unit_name}<span class="text-gray-500">{ing.unit_abbr || ing.unit_name}</span>{/if}
              <span class="font-medium">{ing.ingredient_name}</span>
              {#if ing.notes}<span class="text-gray-400 text-sm">({ing.notes})</span>{/if}
            </li>
          {/each}
        </ul>
      </div>
    {/if}

    {#if recipe.steps?.length}
      <div class="space-y-6">
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">Instructions</h2>
        {#each recipe.steps as step}
          <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
            <div class="flex items-center gap-3 mb-3">
              <span class="w-8 h-8 rounded-full bg-wokko-red text-white flex items-center justify-center text-sm font-bold flex-shrink-0">
                {step.step_order}
              </span>
              <h3 class="font-semibold text-gray-900 dark:text-white">
                Step {step.step_order}{#if step.duration_min} · {step.duration_min} min{/if}
              </h3>
            </div>
            <p class="text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{step.description}</p>
            {#if step.ingredients?.length}
              <ul class="mt-3 space-y-1">
                {#each step.ingredients as ing}
                  <li class="text-sm text-gray-500 dark:text-gray-400 flex items-center gap-1">
                    <span class="w-1.5 h-1.5 rounded-full bg-wokko-orange flex-shrink-0"></span>
                    {#if ing.amount}{ing.amount} {/if}{ing.unit_abbr || ''} {ing.ingredient_name}
                    {#if ing.notes} — {ing.notes}{/if}
                  </li>
                {/each}
              </ul>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}
