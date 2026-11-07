<script lang="ts">
	import { page } from '$app/state';
	import { api, ApiError, errorText } from '#lib/api/client.ts';
	import type { Recipe } from '#lib/api/types.gen.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import PublicShell from '#lib/components/PublicShell.svelte';
	import RecipeView from '#lib/components/RecipeView.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let recipe = $state<Recipe | null>(null);
	let missing = $state(false);
	let error = $state('');

	$effect(() => {
		api.get<Recipe>(`/public/recipes/${page.params.id}`)
			.then((r) => (recipe = r))
			.catch((e) => {
				if (e instanceof ApiError && e.status === 404) missing = true;
				else error = errorText(e);
			});
	});
</script>

<svelte:head><title>{recipe?.name ?? t('gallery.title')} · Wokko</title></svelte:head>

<PublicShell>
	<ErrorNote text={error} />
	{#if recipe}
		<RecipeView {recipe} />
	{:else if missing}
		<div class="flex flex-col items-center gap-3 p-10 text-center"><Mascot /><h1 class="text-xl font-bold">{t('notfound.title')}</h1></div>
	{/if}
</PublicShell>
