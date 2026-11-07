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
		api.get<Recipe>(`/public/share/${page.params.token}`)
			.then((r) => (recipe = r))
			.catch((e) => {
				if (e instanceof ApiError && e.status === 404) missing = true;
				else error = errorText(e);
			});
	});
</script>

<svelte:head>
	<title>{recipe?.name ?? 'Wokko'} · Wokko</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<PublicShell>
	<ErrorNote text={error} />
	{#if recipe}
		<RecipeView {recipe} cookHref="/s/{page.params.token}/cook" />
	{:else if missing}
		<div class="flex flex-col items-center gap-3 p-10 text-center">
			<Mascot />
			<h1 class="text-xl font-bold">{t('share.link_gone')}</h1>
			<p class="text-muted">{t('share.link_gone_hint')}</p>
		</div>
	{/if}
</PublicShell>
