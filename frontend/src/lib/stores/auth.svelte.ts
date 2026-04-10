import { browser } from '$app/environment';
import type { User } from '$lib/api/client';

function createAuthStore() {
  let user = $state<User | null>(null);
  let token = $state<string | null>(null);
  let initialized = $state(false);

  function init() {
    if (!browser) return;
    token = localStorage.getItem('access_token');
    const stored = localStorage.getItem('user');
    if (stored) {
      try { user = JSON.parse(stored); } catch { user = null; }
    }
    initialized = true;
  }

  function setAuth(newToken: string, newUser: User) {
    token = newToken;
    user = newUser;
    if (browser) {
      localStorage.setItem('access_token', newToken);
      localStorage.setItem('user', JSON.stringify(newUser));
    }
  }

  function clearAuth() {
    token = null;
    user = null;
    if (browser) {
      localStorage.removeItem('access_token');
      localStorage.removeItem('user');
    }
  }

  return {
    get user() { return user; },
    get token() { return token; },
    get initialized() { return initialized; },
    get isLoggedIn() { return !!token && !!user; },
    get isAdmin() { return user?.role === 'admin'; },
    init,
    setAuth,
    clearAuth,
  };
}

export const authStore = createAuthStore();
