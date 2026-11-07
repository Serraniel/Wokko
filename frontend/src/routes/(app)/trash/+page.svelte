<script lang="ts">
	import { ArrowLeft, RotateCcw, Trash2 } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { RecipeSummary } from '#lib/api/types.gen.ts';
	import { forget } from '#lib/cache.ts';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { formatDate, t } from '#lib/i18n/index.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let items = $state<RecipeSummary[] | null>(null);
	let error = $state('');
	let purging = $state<RecipeSummary | null>(null);
	let confirmOpen = $state(false);

	async function load() {
		try {
			items = await api.get<RecipeSummary[]>('/trash');
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	async function restore(r: RecipeSummary) {
		try {
			await api.post(`/recipes/${r.id}/restore`);
			forget('recipe');
			toasts.show(t('trash.restored', { name: r.name }));
			await load();
		} catch (e) {
			error = errorText(e);
		}
	}

	async function purge() {
		if (!purging) return;
		try {
			await api.del(`/trash/${purging.id}`);
			confirmOpen = false;
			await load();
		} catch (e) {
			error = errorText(e);
		}
	}
</script>

<svelte:head><title>{t('trash.title')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-4 p-4">
	<header class="flex items-center gap-2">
		<a href="/recipes" class="btn btn-ghost btn-icon" aria-label={t('common.back')}><ArrowLeft size={20} /></a>
		<h1 class="text-2xl font-bold">{t('trash.title')}</h1>
	</header>
	<p class="text-sm text-muted">{t('trash.hint')}</p>
	<ErrorNote text={error} />
	{#if items?.length === 0}
		<p class="py-12 text-center text-muted">{t('trash.empty')}</p>
	{:else if items}
		<ul class="card divide-y divide-line">
			{#each items as r (r.id)}
				<li class="flex items-center gap-2 p-3">
					<div class="min-w-0 flex-1">
						<p class="truncate font-medium">{r.name}</p>
						<p class="text-sm text-muted">{t('trash.deleted_on', { date: formatDate(r.deleted_at) })}</p>
					</div>
					<button class="btn btn-soft shrink-0" onclick={() => restore(r)}><RotateCcw size={17} />{t('trash.restore')}</button>
					<button class="btn btn-ghost btn-icon shrink-0 text-danger" aria-label={t('trash.purge')} onclick={() => { purging = r; confirmOpen = true; }}><Trash2 size={18} /></button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<Sheet bind:open={confirmOpen} title={t('trash.purge')}>
	<div class="space-y-4">
		<p>{t('trash.purge_confirm', { name: purging?.name ?? '' })}</p>
		<div class="flex gap-2">
			<button class="btn btn-soft flex-1" onclick={() => (confirmOpen = false)}>{t('common.cancel')}</button>
			<button class="btn btn-danger flex-1" onclick={purge}>{t('trash.purge')}</button>
		</div>
	</div>
</Sheet>
