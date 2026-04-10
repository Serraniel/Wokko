<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { api } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import Button from '$lib/components/Button.svelte';
  import Input from '$lib/components/Input.svelte';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleLogin(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;
    try {
      const res = await api.login({ username, password });
      authStore.setAuth(res.access_token, res.user);
      const redirect = $page.url.searchParams.get('redirect') || '/recipes';
      goto(redirect);
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen bg-wokko-cream dark:bg-gray-950 flex items-center justify-center p-4">
  <div class="w-full max-w-sm">
    <div class="text-center mb-8">
      <a href="/" class="inline-block">
        <Logo size={64} class="mx-auto mb-3" />
      </a>
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Sign in to Wokko</h1>
    </div>

    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <form onsubmit={handleLogin} class="space-y-4">
        <Input label="Username" bind:value={username} placeholder="username" required autocomplete="username" />
        <Input label="Password" type="password" bind:value={password} placeholder="••••••••" required autocomplete="current-password" />
        {#if error}<p class="text-sm text-red-600 dark:text-red-400">{error}</p>{/if}
        <Button type="submit" class="w-full" loading={loading} disabled={!username || !password}>Sign in</Button>
      </form>
      <p class="text-center text-sm text-gray-500 dark:text-gray-400 mt-4">
        Don't have an account? <a href="/auth/register" class="text-wokko-red hover:underline">Register</a>
      </p>
    </div>
  </div>
</div>
