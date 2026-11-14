<script lang="ts">
	import { ImagePlus, Share2, Trash2 } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { LinkPreviewSettings } from '#lib/api/types.gen.ts';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';
	import ErrorNote from './ErrorNote.svelte';

	// How a Wokko link looks when pasted into Discord, a messenger or elsewhere.
	let set = $state<LinkPreviewSettings | null>(null);
	let error = $state('');
	let busy = $state(false);

	api.get<LinkPreviewSettings>('/admin/link-preview')
		.then((s) => (set = s))
		.catch((e) => (error = errorText(e)));

	async function save(e?: SubmitEvent) {
		e?.preventDefault();
		if (!set) return;
		error = '';
		try {
			set = await api.put<LinkPreviewSettings>('/admin/link-preview', set);
			toasts.show(t('settings.saved'));
		} catch (err) {
			error = errorText(err);
		}
	}

	async function upload(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		busy = true;
		error = '';
		try {
			set = await api.upload<LinkPreviewSettings>('/admin/link-preview/image', file);
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
			input.value = '';
		}
	}

	async function removeImage() {
		error = '';
		try {
			set = await api.del<LinkPreviewSettings>('/admin/link-preview/image');
		} catch (err) {
			error = errorText(err);
		}
	}
</script>

<section class="card space-y-3 p-4" aria-labelledby="preview-title">
	<div>
		<h2 id="preview-title" class="flex items-center gap-2 text-lg font-semibold"><Share2 size={18} class="text-muted" />{t('preview.title')}</h2>
		<p class="hint !mt-0">{t('preview.hint')}</p>
	</div>
	<ErrorNote text={error} />
	{#if set}
		<form class="space-y-3" onsubmit={save}>
			<div>
				<label class="label" for="preview-name">{t('preview.site_name')}</label>
				<input id="preview-name" class="input" bind:value={set.site_name} maxlength="60" placeholder="Wokko" />
			</div>
			<div>
				<label class="label" for="preview-description">{t('preview.description')}</label>
				<textarea id="preview-description" class="input resize-y" rows="2" bind:value={set.description} maxlength="300" placeholder={t('preview.description_placeholder')}></textarea>
			</div>
			<label class="flex items-center justify-between gap-4">
				<span><span class="block font-medium">{t('preview.recipes')}</span><span class="block text-sm text-muted">{t('preview.recipes_hint')}</span></span>
				<input id="preview-recipes" type="checkbox" class="h-5 w-5 shrink-0 accent-[var(--brand)]" bind:checked={set.show_recipes} />
			</label>
			<button class="btn btn-soft w-full">{t('common.save')}</button>
		</form>
		<div class="flex flex-wrap items-center gap-3">
			{#if set.has_image}<img src="/api/v1/public/link-preview-image?v={Date.now()}" alt="" class="h-16 w-32 rounded-lg object-cover" />{:else}<img src="/og-image.png" alt="" class="h-16 w-32 rounded-lg border border-line object-cover" />{/if}
			<label class="btn btn-soft {busy ? 'pointer-events-none opacity-60' : ''}"><ImagePlus size={16} />{t('preview.image_choose')}<input class="sr-only" type="file" accept="image/*" onchange={upload} /></label>
			{#if set.has_image}<button type="button" class="btn btn-ghost text-danger" onclick={removeImage}><Trash2 size={16} />{t('preview.image_remove')}</button>{/if}
		</div>
		<p class="hint !mt-0">{t('preview.public_url')}</p>
	{/if}
</section>
