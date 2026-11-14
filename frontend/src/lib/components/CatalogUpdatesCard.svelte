<script lang="ts">
	import { Check, PackageCheck, Undo2 } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { CatalogStatus, CatalogSuggestion } from '#lib/api/types.gen.ts';
	import { currentLocale, formatDate, t } from '#lib/i18n/index.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';

	let status = $state<CatalogStatus | null>(null);
	let error = $state('');
	let busy = $state(0);

	async function load() {
		try {
			status = await api.get<CatalogStatus>('/admin/catalog');
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	async function resolve(s: CatalogSuggestion, take: boolean) {
		busy = s.id;
		error = '';
		try {
			await api.post(`/admin/catalog/suggestions/${s.id}`, { take });
			await load();
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = 0;
		}
	}

	function language(code: string): string {
		try {
			return new Intl.DisplayNames([currentLocale()], { type: 'language' }).of(code) ?? code;
		} catch {
			return code;
		}
	}

	function label(s: CatalogSuggestion): string {
		const [kind, rest] = s.field.split('/', 2);
		if (kind === 'name') return t('catalog.field_name', { language: language(rest) });
		if (kind === 'alias') return t('catalog.field_alias');
		if (kind === 'nutrient') return `${s.nutrient ?? t('catalog.field_nutrient')} (${s.nutrient_unit ?? ''} / 100 g)`;
		if (kind === 'default_unit') return t('catalog.field_default_unit');
		if (kind === 'grams') return t('catalog.field_grams');
		return s.field;
	}
</script>

<section class="card space-y-3 p-4" aria-labelledby="catalog-title">
	<h2 id="catalog-title" class="flex items-center gap-2 text-lg font-semibold"><PackageCheck size={18} class="text-muted" />{t('catalog.title')}</h2>
	<p class="hint !mt-0">{t('catalog.hint')}</p>
	<ErrorNote text={error} />
	{#if status?.last}
		<p class="text-sm">{t('catalog.last', { date: formatDate(status.last.at, 'datetime'), version: status.last.version, added: status.last.added, updated: status.last.updated })}</p>
	{/if}
	{#if status && status.suggestions.length}
		<div>
			<h3 class="label">{t('catalog.suggestions', { n: status.suggestions.length })}</h3>
			<ul class="divide-y divide-line">
				{#each status.suggestions as s (s.id)}
					<li class="space-y-2 py-3 text-sm">
						<p><span class="font-medium">{s.name}</span> · <span class="text-muted">{label(s)}</span></p>
						<dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
							<dt class="text-muted">{t('catalog.ours')}</dt>
							<dd>{s.ours || t('catalog.none')}</dd>
							<dt class="text-muted">{t('catalog.theirs')}</dt>
							<dd class="font-medium">{s.theirs}</dd>
						</dl>
						<div class="flex flex-wrap gap-2">
							<button class="btn btn-soft" disabled={busy === s.id} onclick={() => resolve(s, true)}><Check size={16} />{t('catalog.take')}</button>
							<button class="btn btn-ghost" disabled={busy === s.id} onclick={() => resolve(s, false)}><Undo2 size={16} />{t('catalog.keep')}</button>
						</div>
					</li>
				{/each}
			</ul>
		</div>
	{:else if status}
		<p class="text-sm text-muted">{t('catalog.nothing')}</p>
	{/if}
</section>
