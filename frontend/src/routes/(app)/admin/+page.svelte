<script lang="ts">
	import { CheckCircle2, DatabaseBackup, KeyRound, Link2, Trash2, TriangleAlert, UserRoundCheck, UserRoundX } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { api, ApiError, errorText } from '#lib/api/client.ts';
	import type { Health, Invite, Settings, User } from '#lib/api/types.gen.ts';
	import CopyField from '#lib/components/CopyField.svelte';
	import ErrorNote from '#lib/components/ErrorNote.svelte';
	import Sheet from '#lib/components/Sheet.svelte';
	import { formatBytes, formatDate, t } from '#lib/i18n/index.svelte.ts';
	import { session } from '#lib/session.svelte.ts';
	import { toasts } from '#lib/toast.svelte.ts';

	let health = $state<Health | null>(null);
	let users = $state<User[]>([]);
	let invites = $state<Invite[]>([]);
	let settings = $state<Settings | null>(null);
	let error = $state('');
	let backingUp = $state(false);
	let link = $state('');
	let linkLabel = $state('');

	let deleting = $state<User | null>(null);
	let deleteOpen = $state(false);
	let needsChoice = $state(false);
	let reassignTo = $state(0);
	let deleteError = $state('');

	$effect(() => {
		if (session.ready && session.user && !session.isAdmin) goto('/recipes', { replaceState: true });
	});

	async function load() {
		try {
			[health, users, invites, settings] = await Promise.all([
				api.get<Health>('/admin/health'),
				api.get<User[]>('/admin/users'),
				api.get<Invite[]>('/admin/invites'),
				api.get<Settings>('/admin/settings')
			]);
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	const warnings = $derived.by(() => {
		if (!health) return [];
		const out: string[] = [];
		if (!health.integrity_ok && health.checked_at) out.push(t('admin.warn_integrity', { reason: health.integrity_error }));
		if (health.backup_failing) out.push(t('admin.warn_backup_failing', { reason: health.last_backup?.error ?? '' }));
		if (health.backup_overdue) out.push(health.last_success ? t('admin.warn_backup_overdue', { date: formatDate(health.last_success.started_at, 'datetime') }) : t('admin.warn_no_backup'));
		if (health.low_disk) out.push(t('admin.warn_low_disk'));
		if (health.missing_photos) out.push(t('admin.warn_missing_photos', { count: health.missing_photos }));
		return out;
	});

	async function backupNow() {
		backingUp = true;
		error = '';
		try {
			await api.post('/admin/backups');
			toasts.show(t('admin.backup_done'));
		} catch (e) {
			error = errorText(e);
		}
		await load();
		backingUp = false;
	}

	async function run(fn: () => Promise<unknown>) {
		error = '';
		try {
			await fn();
			await load();
		} catch (e) {
			error = errorText(e);
		}
	}

	async function createInvite() {
		await run(async () => {
			const inv = await api.post<Invite>('/admin/invites', { max_uses: 1, expires_in_days: 14 });
			link = `${location.origin}/register?invite=${inv.token}`;
			linkLabel = t('admin.invite_link');
		});
	}

	async function resetLink(u: User) {
		await run(async () => {
			const res = await api.post<{ token: string }>(`/admin/users/${u.id}/reset`);
			link = `${location.origin}/reset?token=${res.token}`;
			linkLabel = t('admin.reset_link', { handle: u.handle });
		});
	}

	function startDelete(u: User) {
		deleting = u;
		needsChoice = false;
		reassignTo = 0;
		deleteError = '';
		deleteOpen = true;
	}

	async function doDelete(deleteRecipes: boolean) {
		if (!deleting) return;
		const params = reassignTo ? `?reassign_to=${reassignTo}` : deleteRecipes ? '?delete_recipes=true' : '';
		try {
			await api.del(`/admin/users/${deleting.id}${params}`);
			deleteOpen = false;
			await load();
		} catch (e) {
			if (e instanceof ApiError && e.code === 'user_has_recipes') needsChoice = true;
			else deleteError = errorText(e);
		}
	}

	async function saveSettings() {
		if (!settings) return;
		await run(async () => {
			await api.put('/admin/settings', settings);
			await session.load();
		});
	}
</script>

<svelte:head><title>{t('nav.admin')} · Wokko</title></svelte:head>

<div class="mx-auto max-w-3xl space-y-4 p-4">
	<h1 class="text-2xl font-bold">{t('nav.admin')}</h1>
	<ErrorNote text={error} />

	<section class="card space-y-3 p-4">
		<h2 class="text-lg font-semibold">{t('admin.data_safety')}</h2>
		{#each warnings as w}
			<p role="alert" class="flex gap-2 rounded-xl bg-warn-soft p-3 text-sm font-medium text-warn"><TriangleAlert size={18} class="shrink-0" />{w}</p>
		{:else}
			{#if health}<p class="flex items-center gap-2 rounded-xl bg-ok-soft p-3 text-sm font-medium text-ok"><CheckCircle2 size={18} class="shrink-0" />{t('admin.all_good')}</p>{/if}
		{/each}
		{#if health}
			<dl class="grid grid-cols-2 gap-x-4 gap-y-3 text-sm sm:grid-cols-3">
				<div><dt class="text-muted">{t('admin.last_backup')}</dt><dd class="font-semibold">{health.last_success ? formatDate(health.last_success.started_at, 'datetime') : t('admin.never')}</dd></div>
				<div><dt class="text-muted">{t('admin.backup_size')}</dt><dd class="font-semibold">{health.last_success ? formatBytes(health.last_success.size_bytes) : '–'}</dd></div>
				<div><dt class="text-muted">{t('admin.integrity')}</dt><dd class="font-semibold">{health.checked_at ? formatDate(health.checked_at, 'datetime') : '–'}</dd></div>
				<div><dt class="text-muted">{t('admin.disk_data')}</dt><dd class="font-semibold">{formatBytes(health.data_free_bytes)}</dd></div>
				<div><dt class="text-muted">{t('admin.disk_backups')}</dt><dd class="font-semibold">{formatBytes(health.backup_free_bytes)}</dd></div>
				<div><dt class="text-muted">{t('admin.contents')}</dt><dd class="font-semibold">{t('admin.contents_value', { recipes: health.recipes, foods: health.foods, units: health.units })}</dd></div>
			</dl>
		{/if}
		<button class="btn btn-soft w-full" disabled={backingUp} onclick={backupNow}><DatabaseBackup size={18} />{backingUp ? t('admin.backing_up') : t('admin.backup_now')}</button>
	</section>

	{#if link}
		<section class="rise card space-y-1 p-4">
			<p class="label">{linkLabel}</p>
			<CopyField value={link} label={linkLabel} />
			<p class="hint">{t('admin.link_once')}</p>
		</section>
	{/if}

	<section class="card space-y-2 p-4">
		<h2 class="text-lg font-semibold">{t('admin.users')}</h2>
		<ul class="divide-y divide-line">
			{#each users as u (u.id)}
				<li class="flex flex-wrap items-center gap-1 py-2">
					<span class="min-w-0 flex-1 basis-40">
						<span class="block truncate font-medium">{u.display_name || u.handle} {#if u.disabled}<span class="text-sm font-normal text-danger">({t('admin.disabled')})</span>{/if}</span>
						<span class="block text-sm text-muted">@{u.handle}{u.role === 'admin' ? ` · ${t('admin.role_admin')}` : ''}</span>
					</span>
					{#if u.id !== session.user?.id}
						<button class="btn btn-ghost btn-icon !min-h-9 !w-9" aria-label={t('admin.reset_password')} onclick={() => resetLink(u)}><KeyRound size={17} /></button>
						<button class="btn btn-ghost btn-icon !min-h-9 !w-9" aria-label={u.disabled ? t('admin.enable') : t('admin.disable')} onclick={() => run(() => api.patch(`/admin/users/${u.id}`, { disabled: !u.disabled }))}>
							{#if u.disabled}<UserRoundCheck size={17} />{:else}<UserRoundX size={17} />{/if}
						</button>
						<button class="btn btn-ghost btn-icon !min-h-9 !w-9 text-danger" aria-label={t('common.delete')} onclick={() => startDelete(u)}><Trash2 size={17} /></button>
					{/if}
				</li>
			{/each}
		</ul>
	</section>

	<section class="card space-y-2 p-4">
		<h2 class="text-lg font-semibold">{t('admin.invites')}</h2>
		{#if invites.length}
			<ul class="divide-y divide-line">
				{#each invites as inv (inv.id)}
					<li class="flex items-center gap-2 py-2 text-sm">
						<span class="min-w-0 flex-1">{t('admin.invite_used', { used: inv.use_count, max: inv.max_uses })}{#if inv.expires_at} · {t('share.expires', { date: formatDate(inv.expires_at) })}{/if}</span>
						<button class="btn btn-ghost btn-icon !min-h-9 !w-9 text-danger" aria-label={t('common.delete')} onclick={() => run(() => api.del(`/admin/invites/${inv.id}`))}><Trash2 size={17} /></button>
					</li>
				{/each}
			</ul>
		{/if}
		<button class="btn btn-soft w-full" onclick={createInvite}><Link2 size={18} />{t('admin.create_invite')}</button>
	</section>

	{#if settings}
		<section class="card divide-y divide-line">
			<label class="flex items-center justify-between gap-4 p-4">
				<span><span class="block font-medium">{t('admin.open_registration')}</span><span class="block text-sm text-muted">{t('admin.open_registration_hint')}</span></span>
				<input type="checkbox" class="h-5 w-5 shrink-0 accent-[var(--brand)]" bind:checked={settings.open_registration} onchange={saveSettings} />
			</label>
			<label class="flex items-center justify-between gap-4 p-4">
				<span><span class="block font-medium">{t('admin.allow_public')}</span><span class="block text-sm text-muted">{t('admin.allow_public_hint')}</span></span>
				<input type="checkbox" class="h-5 w-5 shrink-0 accent-[var(--brand)]" bind:checked={settings.allow_public} onchange={saveSettings} />
			</label>
			<label class="flex items-center justify-between gap-3 py-2">
				<span><span class="block font-medium">{t('admin.mcp')}</span><span class="block text-sm text-muted">{t('admin.mcp_hint')}</span></span>
				<input id="mcp-enabled" type="checkbox" class="h-5 w-5 shrink-0 accent-[var(--brand)]" bind:checked={settings.mcp_enabled} onchange={saveSettings} />
			</label>
		</section>
	{/if}
</div>

<Sheet bind:open={deleteOpen} title={t('admin.delete_user', { handle: deleting?.handle ?? '' })}>
	<div class="space-y-4">
		<ErrorNote text={deleteError} />
		{#if needsChoice}
			<p>{t('admin.delete_has_recipes')}</p>
			<div>
				<label class="label" for="target">{t('admin.give_recipes_to')}</label>
				<select id="target" class="input" bind:value={reassignTo}>
					<option value={0}>{t('admin.choose_user')}</option>
					{#each users.filter((u) => u.id !== deleting?.id) as u (u.id)}<option value={u.id}>{u.handle}</option>{/each}
				</select>
			</div>
			<button class="btn btn-primary w-full" disabled={!reassignTo} onclick={() => doDelete(false)}>{t('admin.delete_keep_recipes')}</button>
			<button class="btn btn-danger w-full" onclick={() => { reassignTo = 0; doDelete(true); }}>{t('admin.delete_with_recipes')}</button>
		{:else}
			<p>{t('admin.delete_confirm')}</p>
			<div class="flex gap-2">
				<button class="btn btn-soft flex-1" onclick={() => (deleteOpen = false)}>{t('common.cancel')}</button>
				<button class="btn btn-danger flex-1" onclick={() => doDelete(false)}>{t('common.delete')}</button>
			</div>
		{/if}
	</div>
</Sheet>
