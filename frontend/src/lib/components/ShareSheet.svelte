<script lang="ts">
	import { Globe, Home, Link2, Lock, Trash2, UserPlus, X } from '@lucide/svelte';
	import { api, errorText } from '#lib/api/client.ts';
	import type { Friend, Recipe, RecipeInput, ShareLink, Sharing } from '#lib/api/types.gen.ts';
	import { formatDate, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { toInput } from '#lib/recipe.ts';
	import { toasts } from '#lib/toast.svelte.ts';
	import CopyField from './CopyField.svelte';
	import ErrorNote from './ErrorNote.svelte';
	import Sheet from './Sheet.svelte';

	let { open = $bindable(false), recipe, onchange }: { open?: boolean; recipe: Recipe; onchange: (r: Recipe) => void } = $props();

	let sharing = $state<Sharing | null>(null);
	let friends = $state<Friend[]>([]);
	let handle = $state('');
	let error = $state('');
	let busy = $state(false);

	$effect(() => {
		if (open) load();
	});

	async function load() {
		error = '';
		try {
			[sharing, friends] = await Promise.all([api.get<Sharing>(`/recipes/${recipe.id}/sharing`), api.get<Friend[]>('/friends')]);
		} catch (e) {
			error = errorText(e);
		}
	}

	const pickable = $derived(friends.filter((f) => f.accepted && !sharing?.users.some((u) => u.id === f.person.id)));

	async function shareWith(h: string) {
		busy = true;
		error = '';
		try {
			await api.post(`/recipes/${recipe.id}/sharing/users`, { handle: h });
			handle = '';
			await load();
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	async function act(fn: () => Promise<unknown>) {
		error = '';
		try {
			await fn();
			await load();
		} catch (e) {
			error = errorText(e);
		}
	}

	async function setVisibility(visibility: string) {
		if (visibility === recipe.visibility) return;
		const input: RecipeInput = { ...toInput(recipe), visibility };
		try {
			onchange(await api.put<Recipe>(`/recipes/${recipe.id}`, input));
			toasts.show(t('share.visibility_saved'));
		} catch (e) {
			error = errorText(e);
		}
	}

	/** Falls back to this site's address when the server has no PUBLIC_URL configured. */
	function absolute(url: string): string {
		return url.startsWith('/') ? `${location.origin}${url}` : url;
	}

	const levels = $derived([
		{ id: 'private', icon: Lock, title: t('visibility.private'), text: t('visibility.private_hint') },
		{ id: 'household', icon: Home, title: t('visibility.household'), text: t('visibility.household_hint') },
		...(session.status?.allow_public ? [{ id: 'public', icon: Globe, title: t('visibility.public'), text: t('visibility.public_hint') }] : [])
	]);
</script>

<Sheet bind:open title={t('share.title')}>
	<div class="space-y-6">
		<ErrorNote text={error} />
		<fieldset class="space-y-2">
			<legend class="label">{t('share.who_can_see')}</legend>
			{#each levels as level (level.id)}
				{@const Icon = level.icon}
				<button type="button" class="flex w-full items-center gap-3 rounded-xl border p-3 text-left transition-colors {recipe.visibility === level.id ? 'border-brand bg-brand-soft' : 'border-line hover:bg-surface-2'}" aria-pressed={recipe.visibility === level.id} onclick={() => setVisibility(level.id)}>
					<Icon size={20} class="shrink-0" />
					<span class="min-w-0"><span class="block font-semibold">{level.title}</span><span class="block text-sm text-muted">{level.text}</span></span>
				</button>
			{/each}
		</fieldset>

		<section class="space-y-2">
			<h3 class="label !mb-0">{t('share.with_person')}</h3>
			<p class="hint !mt-0">{t('share.with_person_hint')}</p>
			<form class="flex gap-2" onsubmit={(e) => { e.preventDefault(); if (handle.trim()) shareWith(handle.trim()); }}>
				<input class="input min-w-0 flex-1" bind:value={handle} placeholder={t('common.exact_handle')} aria-label={t('common.exact_handle')} autocomplete="off" autocapitalize="off" spellcheck="false" />
				<button class="btn btn-soft shrink-0" disabled={busy || !handle.trim()}><UserPlus size={18} />{t('share.add')}</button>
			</form>
			{#if pickable.length}
				<div class="flex flex-wrap gap-1.5">
					{#each pickable as f (f.id)}
						<button type="button" class="chip hover:bg-line" onclick={() => shareWith(f.person.handle)}><UserPlus size={14} />{f.person.display_name || f.person.handle}</button>
					{/each}
				</div>
			{/if}
			{#if sharing?.users.length}
				<ul class="divide-y divide-line">
					{#each sharing.users as u (u.id)}
						<li class="flex items-center justify-between gap-2 py-1.5">
							<span class="truncate">{u.display_name || u.handle} <span class="text-sm text-muted">@{u.handle}</span></span>
							<button class="btn btn-ghost btn-icon shrink-0" aria-label={t('share.remove_person', { name: u.handle })} onclick={() => act(() => api.del(`/recipes/${recipe.id}/sharing/users/${u.id}`))}><X size={18} /></button>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="space-y-2">
			<h3 class="label !mb-0">{t('share.links')}</h3>
			<p class="hint !mt-0">{t('share.links_hint')}</p>
			{#each sharing?.links ?? [] as link (link.id)}
				<div class="space-y-1">
					<div class="flex items-center gap-2">
						<div class="min-w-0 flex-1"><CopyField value={absolute(link.url)} label={t('share.link')} /></div>
						<button class="btn btn-ghost btn-icon shrink-0 text-danger" aria-label={t('share.revoke')} onclick={() => act(() => api.del(`/recipes/${recipe.id}/sharing/links/${link.id}`))}><Trash2 size={18} /></button>
					</div>
					{#if link.expires_at}<p class="hint">{t('share.expires', { date: formatDate(link.expires_at) })}</p>{/if}
				</div>
			{/each}
			<button class="btn btn-soft w-full" onclick={() => act(() => api.post<ShareLink>(`/recipes/${recipe.id}/sharing/links`, {}))}><Link2 size={18} />{t('share.create_link')}</button>
		</section>
	</div>
</Sheet>
