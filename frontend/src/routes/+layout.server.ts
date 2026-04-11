import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080';

export const load: LayoutServerLoad = async ({ url, cookies }) => {
  // Don't redirect if already heading to /setup
  if (url.pathname.startsWith('/setup')) return {};

  // If the user has a refresh token, setup is already complete — skip the check
  if (cookies.get('refresh_token')) return {};

  try {
    const res = await fetch(`${BACKEND_URL}/api/auth/setup/status`);
    if (res.ok) {
      const data = await res.json();
      if (!data.setup_complete) {
        throw redirect(302, '/setup');
      }
    }
  } catch (e) {
    // If it's a redirect, re-throw it
    if (e && typeof e === 'object' && 'status' in e) throw e;
    // Otherwise ignore — backend might be starting up
  }

  return {};
};
