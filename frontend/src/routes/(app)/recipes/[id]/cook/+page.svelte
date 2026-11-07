<script lang="ts">
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import type { Recipe } from '#lib/api/types.gen.ts';
	import { cached, remember } from '#lib/cache.ts';
	import CookMode from '#lib/components/CookMode.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';

	const id = page.params.id;
	let recipe = $state<Recipe | null>(cached(`recipe:${id}`) ?? null);
	let error = $state('');

	api.get<Recipe>(`/recipes/${id}`)
		.then((r) => (recipe = remember(`recipe:${id}`, r)))
		.catch((e) => {
			if (!recipe) error = errorText(e);
		});
</script>

<ErrorNote text={error} />
{#if recipe}<CookMode {recipe} servings={Number(page.url.searchParams.get('servings')) || 0} exitHref="/recipes/{id}" doneHref="/recipes/{id}?cooked=1" />{/if}
