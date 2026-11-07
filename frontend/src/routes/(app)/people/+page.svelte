<script lang="ts">
	import { Check, DoorOpen, Link2, Trash2, UserPlus, X } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText } from '#lib/api/client.ts';
	import type { Friend, Household, Invite } from '#lib/api/types.gen.ts';
	import CopyField from '#lib/components/CopyField.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let friends = $state<Friend[] | null>(null);
	let household = $state<Household | null>(null);
	let handle = $state('');
	let houseName = $state('');
	let joinCode = $state(page.url.searchParams.get('join') ?? '');
	let inviteLink = $state('');
	let error = $state('');

	async function load() {
		try {
			[friends, household] = await Promise.all([api.get<Friend[]>('/friends'), api.get<Household>('/household')]);
			houseName = household.name;
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	async function run(fn: () => Promise<unknown>) {
		error = '';
		try {
			await fn();
			await load();
			return true;
		} catch (e) {
			error = errorText(e);
			window.scrollTo({ top: 0, behavior: 'smooth' });
			return false;
		}
	}

	async function addFriend(e: SubmitEvent) {
		e.preventDefault();
		if (await run(() => api.post('/friends', { handle: handle.trim() }))) {
			toasts.show(t('people.request_sent', { handle: handle.trim() }));
			handle = '';
		}
	}

	async function createInvite() {
		await run(async () => {
			const inv = await api.post<Invite>('/household/invites', { max_uses: 1, expires_in_days: 7 });
			inviteLink = `${location.origin}/register?invite=${inv.token}`;
		});
	}

	async function join(e: SubmitEvent) {
		e.preventDefault();
		if (await run(() => api.post('/household/join', { token: joinCode.trim() }))) {
			toasts.show(t('people.joined'));
			joinCode = '';
			await session.load();
			goto('/people', { replaceState: true });
		}
	}

	async function leave() {
		if (await run(() => api.post('/household/leave'))) await session.load();
	}

	const accepted = $derived(friends?.filter((f) => f.accepted) ?? []);
	const incoming = $derived(friends?.filter((f) => !f.accepted && f.incoming) ?? []);
	const outgoing = $derived(friends?.filter((f) => !f.accepted && !f.incoming) ?? []);
</script>

<svelte:head><title>{t('nav.people')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-4 p-4">
	<h1 class="text-2xl font-bold">{t('nav.people')}</h1>
	<ErrorNote text={error} />

	<section class="card space-y-3 p-4">
		<h2 class="text-lg font-semibold">{t('people.friends')}</h2>
		<p class="text-sm text-muted">{t('people.friends_hint')}</p>
		<form class="flex gap-2" onsubmit={addFriend}>
			<input class="input min-w-0 flex-1" bind:value={handle} placeholder={t('common.exact_handle')} aria-label={t('common.exact_handle')} autocomplete="off" autocapitalize="off" spellcheck="false" />
			<button class="btn btn-soft shrink-0" disabled={!handle.trim()}><UserPlus size={18} />{t('people.add')}</button>
		</form>

		{#if incoming.length}
			<h3 class="label !mb-0 pt-1">{t('people.requests')}</h3>
			<ul class="divide-y divide-line">
				{#each incoming as f (f.id)}
					<li class="flex items-center gap-2 py-2">
						<span class="min-w-0 flex-1 truncate">{f.person.display_name || f.person.handle} <span class="text-sm text-muted">@{f.person.handle}</span></span>
						<button class="btn btn-primary btn-icon !min-h-9 !w-9" aria-label={t('people.accept')} onclick={() => run(() => api.post(`/friends/${f.id}/accept`))}><Check size={18} /></button>
						<button class="btn btn-ghost btn-icon !min-h-9 !w-9" aria-label={t('people.decline')} onclick={() => run(() => api.del(`/friends/${f.id}`))}><X size={18} /></button>
					</li>
				{/each}
			</ul>
		{/if}
		{#if accepted.length}
			<ul class="divide-y divide-line">
				{#each accepted as f (f.id)}
					<li class="flex items-center gap-2 py-2">
						<span class="min-w-0 flex-1 truncate">{f.person.display_name || f.person.handle} <span class="text-sm text-muted">@{f.person.handle}</span></span>
						<button class="btn btn-ghost btn-icon !min-h-9 !w-9 text-muted" aria-label={t('people.remove')} onclick={() => run(() => api.del(`/friends/${f.id}`))}><Trash2 size={17} /></button>
					</li>
				{/each}
			</ul>
		{/if}
		{#if outgoing.length}
			<h3 class="label !mb-0 pt-1">{t('people.pending')}</h3>
			<ul class="divide-y divide-line">
				{#each outgoing as f (f.id)}
					<li class="flex items-center gap-2 py-2">
						<span class="min-w-0 flex-1 truncate text-muted">@{f.person.handle}</span>
						<button class="btn btn-ghost btn-icon !min-h-9 !w-9 text-muted" aria-label={t('common.cancel')} onclick={() => run(() => api.del(`/friends/${f.id}`))}><X size={18} /></button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section class="card space-y-3 p-4">
		<h2 class="text-lg font-semibold">{t('people.household')}</h2>
		<p class="text-sm text-muted">{t('people.household_hint')}</p>
		<form class="flex gap-2" onsubmit={(e) => { e.preventDefault(); run(() => api.patch('/household', { name: houseName })); }}>
			<input class="input min-w-0 flex-1" bind:value={houseName} aria-label={t('people.household_name')} />
			<button class="btn btn-soft shrink-0" disabled={!houseName.trim() || houseName === household?.name}>{t('common.save')}</button>
		</form>
		{#if household}
			<ul class="divide-y divide-line">
				{#each household.members as m (m.id)}
					<li class="truncate py-2">{m.display_name || m.handle} <span class="text-sm text-muted">@{m.handle}</span></li>
				{/each}
			</ul>
		{/if}
		{#if inviteLink}
			<div class="rise space-y-1">
				<CopyField value={inviteLink} label={t('people.invite_link')} />
				<p class="hint">{t('people.invite_hint')}</p>
			</div>
		{/if}
		<button class="btn btn-soft w-full" onclick={createInvite}><Link2 size={18} />{t('people.invite')}</button>

		<form class="flex gap-2 border-t border-line pt-3" onsubmit={join}>
			<input class="input min-w-0 flex-1" bind:value={joinCode} placeholder={t('people.join_code')} aria-label={t('people.join_code')} autocomplete="off" autocapitalize="off" spellcheck="false" />
			<button class="btn btn-soft shrink-0" disabled={!joinCode.trim()}>{t('people.join')}</button>
		</form>
		{#if (household?.members.length ?? 0) > 1}
			<button class="btn btn-ghost w-full text-danger" onclick={leave}><DoorOpen size={18} />{t('people.leave')}</button>
		{/if}
	</section>
</div>
