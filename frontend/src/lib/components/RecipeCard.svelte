<script lang="ts">
  import type { Recipe } from '$lib/api/client';
  import { Clock, Users } from 'lucide-svelte';

  let { recipe, href = `/recipes/${recipe.id}` } = $props<{ recipe: Recipe; href?: string }>();

  const API_BASE = import.meta.env.PUBLIC_API_URL || '/api';
  const photoUrl = recipe.photo_path ? `${API_BASE}/media/${recipe.photo_path}` : null;
</script>

<a {href} class="group block rounded-xl overflow-hidden border border-gray-200 dark:border-gray-700
                  bg-white dark:bg-gray-800 hover:shadow-md transition-all duration-200
                  hover:border-wokko-red/30">
  <div class="aspect-video bg-gray-100 dark:bg-gray-700 overflow-hidden">
    {#if photoUrl}
      <img src={photoUrl} alt={recipe.name} class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
    {:else}
      <div class="w-full h-full flex items-center justify-center text-4xl">🍳</div>
    {/if}
  </div>
  <div class="p-4">
    <h3 class="font-semibold text-gray-900 dark:text-gray-100 group-hover:text-wokko-red transition-colors line-clamp-1">
      {recipe.name}
    </h3>
    {#if recipe.description}
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1 line-clamp-2">{recipe.description}</p>
    {/if}
    <div class="flex items-center gap-3 mt-2 text-xs text-gray-400 dark:text-gray-500">
      {#if recipe.work_time_min}
        <span class="flex items-center gap-1"><Clock class="w-3 h-3" />{recipe.work_time_min} min</span>
      {/if}
      {#if recipe.servings}
        <span class="flex items-center gap-1"><Users class="w-3 h-3" />{recipe.servings}</span>
      {/if}
    </div>
    {#if recipe.tags?.length}
      <div class="flex flex-wrap gap-1 mt-2">
        {#each recipe.tags.slice(0, 3) as tag}
          <span class="px-2 py-0.5 text-xs rounded-full bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
            {tag}
          </span>
        {/each}
      </div>
    {/if}
  </div>
</a>
