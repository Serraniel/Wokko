<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import Button from '$lib/components/Button.svelte';
  import Input from '$lib/components/Input.svelte';

  let username = $state('');
  let password = $state('');
  let email = $state('');
  let inviteToken = $state($page.url.searchParams.get('invite') || '');
  let error = $state('');
  let loading = $state(false);

  async function handleRegister(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;
    try {
      const res = await api.register({
        username, password,
        email: email || undefined,
        invite_token: inviteToken || undefined,
      });
      authStore.setAuth(res.access_token, res.user);
      goto('/recipes');
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Registration failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen bg-wokko-cream dark:bg-gray-950 flex items-center justify-center p-4">
  <div class="w-full max-w-sm">
    <div class="text-center mb-8">
      <Logo size={64} class="mx-auto mb-3" />
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Create account</h1>
    </div>
    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <form onsubmit={handleRegister} class="space-y-4">
        <Input label="Username" bind:value={username} placeholder="username" required autofocus />
        <Input label="Password" type="password" bind:value={password} placeholder="Min. 8 characters" required />
        <Input label="Email (optional)" type="email" bind:value={email} placeholder="you@example.com" />
        <Input label="Invite Code" bind:value={inviteToken} placeholder="Paste your invite code" />
        {#if error}<p class="text-sm text-red-600 dark:text-red-400">{error}</p>{/if}
        <Button type="submit" class="w-full" loading={loading} disabled={!username || !password}>Create account</Button>
      </form>
      <p class="text-center text-sm text-gray-500 dark:text-gray-400 mt-4">
        Already have an account? <a href="/auth/login" class="text-wokko-red hover:underline">Sign in</a>
      </p>
    </div>
  </div>
</div>
