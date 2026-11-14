<script lang="ts">
	import { Check, ImagePlus } from '@lucide/svelte';
	import { page } from '$app/state';
	import { api, ApiError, errorText } from '#lib/api/client.ts';
	import type { PhotoUpload } from '#lib/api/types.gen.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Mascot from '#lib/components/Mascot.svelte';
	import PublicShell from '#lib/components/PublicShell.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';

	// A one-time link from an assistant: pick or take a photo, and it becomes the recipe's photo.
	let info = $state<PhotoUpload | null>(null);
	let missing = $state(false);
	let error = $state('');
	let busy = $state(false);
	let done = $state(false);

	$effect(() => {
		api.get<PhotoUpload>(`/public/photo-upload/${page.params.token}`)
			.then((r) => (info = r))
			.catch((e) => {
				if (e instanceof ApiError && e.status === 404) missing = true;
				else error = errorText(e);
			});
	});

	async function choose(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		busy = true;
		error = '';
		try {
			await api.upload(`/public/photo-upload/${page.params.token}`, file);
			done = true;
		} catch (err) {
			if (err instanceof ApiError && err.status === 404) missing = true;
			else error = errorText(err);
			input.value = '';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>{t('upload.title')} · Wokko</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<PublicShell>
	<div class="mx-auto max-w-md space-y-4 p-4">
		{#if done}
			<div class="flex flex-col items-center gap-3 p-6 text-center" role="status">
				<Check size={36} class="text-ok" />
				<h1 class="text-xl font-bold">{t('upload.done')}</h1>
				<p class="text-muted">{t('upload.done_hint', { name: info?.recipe_name ?? '' })}</p>
			</div>
		{:else if missing}
			<div class="flex flex-col items-center gap-3 p-10 text-center">
				<Mascot />
				<h1 class="text-xl font-bold">{t('upload.gone')}</h1>
				<p class="text-muted">{t('upload.gone_hint')}</p>
			</div>
		{:else if info}
			<h1 class="text-xl font-bold">{t('upload.title')}</h1>
			<p class="text-muted">{t('upload.for', { name: info.recipe_name })}</p>
			<ErrorNote text={error} />
			<label class="btn btn-primary w-full {busy ? 'pointer-events-none opacity-60' : ''}">
				<ImagePlus size={18} />{busy ? t('upload.working') : t('upload.choose')}
				<input class="sr-only" type="file" accept="image/*" disabled={busy} onchange={choose} />
			</label>
			<p class="hint">{t('upload.hint')}</p>
		{:else}
			<ErrorNote text={error} />
		{/if}
	</div>
</PublicShell>
