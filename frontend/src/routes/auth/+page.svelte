<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { api } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import Button from '$lib/components/Button.svelte';
  import Input from '$lib/components/Input.svelte';
  import ThemeToggle from '$lib/components/ThemeToggle.svelte';

  // Redirect to /auth/login
  import { onMount } from 'svelte';
  onMount(() => goto('/auth/login'));
</script>
