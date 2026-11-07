<script lang="ts">
	import { page } from '$app/state';
	import { api, ApiError, errorText } from '#lib/api/client.ts';
	import type { Recipe } from '#lib/api/types.gen.ts';
	import CookMode from '#lib/components/CookMode.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	let recipe = $state<Recipe | null>(null);
	let missing = $state(false);
	let error = $state('');

	api.get<Recipe>(`/public/recipes/${page.params.id}`)
		.then((r) => (recipe = r))
		.catch((e) => {
			if (e instanceof ApiError && e.status === 404) missing = true;
			else error = errorText(e);
		});
</script>

<svelte:head><meta name="robots" content="noindex" /></svelte:head>

<main>
	<ErrorNote text={error} />
	{#if recipe}
		<CookMode {recipe} servings={Number(page.url.searchParams.get('servings')) || 0} exitHref={`/gallery/${page.params.id}`} doneHref={`/gallery/${page.params.id}`} />
	{:else if missing}
		<div class="flex flex-col items-center gap-3 p-10 text-center"><Mascot /><h1 class="text-xl font-semibold">{t('share.link_gone')}</h1></div>
	{/if}
</main>
