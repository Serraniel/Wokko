<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api, type User, type InviteLink, type SystemSettings } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import Button from '$lib/components/Button.svelte';
  import { Copy, Trash2, UserX } from 'lucide-svelte';

  let tab = $state<'users' | 'invites' | 'settings'>('users');
  let users = $state<User[]>([]);
  let invites = $state<InviteLink[]>([]);
  let settings = $state<SystemSettings>({ open_registration: false, allow_public_recipes: true, mcp_enabled: false });
  let loading = $state(true);
  let newInviteUrl = $state('');
  let savingSettings = $state(false);

  onMount(async () => {
    if (!authStore.initialized) authStore.init();
    if (!authStore.isAdmin) { goto('/recipes'); return; }
    await Promise.all([loadUsers(), loadInvites(), loadSettings()]);
    loading = false;
  });

  async function loadUsers() { users = await api.listUsers().catch(() => []); }
  async function loadInvites() { invites = await api.listInvites().catch(() => []); }
  async function loadSettings() { settings = await api.getSettings().catch(() => settings); }

  async function deleteUser(id: string) {
    if (!confirm('Delete this user?')) return;
    await api.deleteUser(id);
    users = users.filter(u => u.id !== id);
  }

  async function createInvite() {
    const invite = await api.createInvite({ max_uses: 1 });
    const url = `${window.location.origin}/auth/register?invite=${invite.token}`;
    newInviteUrl = url;
    await navigator.clipboard.writeText(url).catch(() => {});
    await loadInvites();
  }

  async function deleteInvite(id: string) {
    await api.deleteInvite(id);
    invites = invites.filter(i => i.id !== id);
  }

  async function saveSettings() {
    savingSettings = true;
    await api.updateSettings(settings).catch(() => {});
    savingSettings = false;
  }
</script>

<div class="space-y-6">
  <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Admin Panel</h1>

  <div class="flex gap-1 border-b border-gray-200 dark:border-gray-700">
    {#each [['users', 'Users'], ['invites', 'Invite Links'], ['settings', 'Settings']] as [id, label]}
      <button
        onclick={() => tab = id as typeof tab}
        class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
               {tab === id ? 'border-wokko-red text-wokko-red' : 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'}"
      >{label}</button>
    {/each}
  </div>

  {#if tab === 'users'}
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 dark:bg-gray-700/50">
          <tr>
            <th class="text-left px-4 py-3 text-gray-700 dark:text-gray-300">Username</th>
            <th class="text-left px-4 py-3 text-gray-700 dark:text-gray-300">Role</th>
            <th class="text-left px-4 py-3 text-gray-700 dark:text-gray-300">Email</th>
            <th class="text-left px-4 py-3 text-gray-700 dark:text-gray-300">Joined</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
          {#each users as user}
            <tr>
              <td class="px-4 py-3 font-medium text-gray-900 dark:text-white">{user.username}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-xs {user.role === 'admin' ? 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300' : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'}">
                  {user.role}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-500 dark:text-gray-400">{user.email || '—'}</td>
              <td class="px-4 py-3 text-gray-500 dark:text-gray-400">{new Date(user.created_at).toLocaleDateString()}</td>
              <td class="px-4 py-3 text-right">
                {#if user.id !== authStore.user?.id}
                  <button onclick={() => deleteUser(user.id)} class="text-red-400 hover:text-red-600 transition-colors p-1">
                    <UserX class="w-4 h-4" />
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

  {:else if tab === 'invites'}
    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <Button onclick={createInvite}><Copy class="w-4 h-4" /> Generate Invite Link</Button>
      </div>
      {#if newInviteUrl}
        <div class="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700 rounded-lg p-3 text-sm">
          Copied to clipboard: <code class="font-mono text-xs break-all">{newInviteUrl}</code>
        </div>
      {/if}
      <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th class="text-left px-4 py-3 text-gray-700 dark:text-gray-300">Token</th>
              <th class="text-left px-4 py-3 text-gray-700 dark:text-gray-300">Uses</th>
              <th class="text-left px-4 py-3 text-gray-700 dark:text-gray-300">Expires</th>
              <th class="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
            {#each invites as invite}
              <tr>
                <td class="px-4 py-3 font-mono text-xs text-gray-500 dark:text-gray-400">{invite.token.slice(0, 16)}…</td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">{invite.use_count}/{invite.max_uses}</td>
                <td class="px-4 py-3 text-gray-500 dark:text-gray-400">{invite.expires_at ? new Date(invite.expires_at).toLocaleDateString() : 'Never'}</td>
                <td class="px-4 py-3 text-right">
                  <button onclick={() => deleteInvite(invite.id)} class="text-red-400 hover:text-red-600 p-1">
                    <Trash2 class="w-4 h-4" />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

  {:else if tab === 'settings'}
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 space-y-6 max-w-lg">
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="font-medium text-gray-900 dark:text-white">Open Registration</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">Allow anyone to create an account without invite</p>
          </div>
          <input type="checkbox" bind:checked={settings.open_registration}
            class="rounded border-gray-300 text-wokko-red focus:ring-wokko-red w-5 h-5 cursor-pointer" />
        </div>
        <div class="flex items-center justify-between">
          <div>
            <p class="font-medium text-gray-900 dark:text-white">Allow Public Recipes</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">Let users make recipes publicly visible</p>
          </div>
          <input type="checkbox" bind:checked={settings.allow_public_recipes}
            class="rounded border-gray-300 text-wokko-red focus:ring-wokko-red w-5 h-5 cursor-pointer" />
        </div>
        <div class="flex items-center justify-between">
          <div>
            <p class="font-medium text-gray-900 dark:text-white">MCP Protocol</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">Enable AI integration via Model Context Protocol</p>
          </div>
          <input type="checkbox" bind:checked={settings.mcp_enabled}
            class="rounded border-gray-300 text-wokko-red focus:ring-wokko-red w-5 h-5 cursor-pointer" />
        </div>
      </div>
      <Button onclick={saveSettings} loading={savingSettings}>Save Settings</Button>
    </div>
  {/if}
</div>
