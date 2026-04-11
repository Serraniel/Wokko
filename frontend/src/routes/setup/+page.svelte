<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import Button from '$lib/components/Button.svelte';
  import Input from '$lib/components/Input.svelte';

  let step = $state(1);
  let username = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let displayName = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleSetup() {
    if (password !== confirmPassword) { error = 'Passwords do not match'; return; }
    if (password.length < 8) { error = 'Password must be at least 8 characters'; return; }
    error = '';
    loading = true;
    try {
      const res = await api.setup({ username, password, display_name: displayName || undefined });
      authStore.setAuth(res.access_token, res.user);
      goto('/recipes');
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Setup failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen bg-wokko-cream dark:bg-gray-950 flex items-center justify-center p-4">
  <div class="w-full max-w-md">
    <div class="text-center mb-8">
      <Logo size={80} class="mx-auto mb-4" />
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Welcome to Wokko</h1>
      <p class="text-gray-500 dark:text-gray-400 mt-2">Set up your recipe management instance</p>
    </div>

    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-200 dark:border-gray-700 p-8">
      {#if step === 1}
        <h2 class="text-xl font-semibold mb-6 text-gray-900 dark:text-white">Create Admin Account</h2>
        <form onsubmit={(e) => { e.preventDefault(); step = 2; }} class="space-y-4">
          <Input label="Username" bind:value={username} placeholder="admin" required autofocus />
          <Input label="Display Name (optional)" bind:value={displayName} placeholder="Your name" />
          <Button type="submit" class="w-full" disabled={!username}>Continue →</Button>
        </form>
      {:else if step === 2}
        <h2 class="text-xl font-semibold mb-6 text-gray-900 dark:text-white">Set Password</h2>
        <form onsubmit={(e) => { e.preventDefault(); handleSetup(); }} class="space-y-4">
          <Input label="Password" type="password" bind:value={password} placeholder="Min. 8 characters" required />
          <Input label="Confirm Password" type="password" bind:value={confirmPassword} placeholder="Repeat password" required />
          {#if error}<p class="text-sm text-red-600">{error}</p>{/if}
          <div class="flex gap-3">
            <Button type="button" variant="secondary" onclick={() => step = 1}>← Back</Button>
            <Button type="submit" class="flex-1" loading={loading} disabled={!password || !confirmPassword}>
              Create Account
            </Button>
          </div>
        </form>
      {/if}
    </div>
  </div>
</div>
