<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api, type RecipeDetail } from '$lib/api/client';
  import RecipeEditor from '$lib/components/RecipeEditor.svelte';

  let recipe = $state<RecipeDetail | null>(null);

  onMount(async () => {
    try {
      recipe = await api.getRecipe($page.params.id!);
    } catch {
      goto('/recipes');
    }
  });
</script>

{#if recipe}
  <RecipeEditor {recipe} />
{:else}
  <div class="flex justify-center py-20">
    <div class="animate-spin w-8 h-8 border-2 border-wokko-red border-t-transparent rounded-full"></div>
  </div>
{/if}
