<script lang="ts">
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import type { Recipe } from '#lib/api/types.gen.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import RecipeEditor from '#lib/components/RecipeEditor.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let recipe = $state<Recipe | null>(null);
	let error = $state('');

	// Always loads fresh, so editing starts from the current version.
	$effect(() => {
		api.get<Recipe>(`/recipes/${page.params.id}`)
			.then((r) => (recipe = r))
			.catch((e) => (error = errorText(e)));
	});
</script>

<svelte:head><title>{t('editor.edit_title')} · Wokko</title></svelte:head>

<ErrorNote text={error} />
{#if recipe}
	{#key recipe.id}<RecipeEditor {recipe} />{/key}
{/if}
