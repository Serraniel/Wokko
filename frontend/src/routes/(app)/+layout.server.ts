import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080';

export const load: LayoutServerLoad = async ({ request }) => {
  const cookie = request.headers.get('cookie') || '';

  // Try to validate the session using the refresh token cookie.
  // If the backend returns 401, the user is not authenticated.
  try {
    const res = await fetch(`${BACKEND_URL}/api/auth/refresh`, {
      method: 'POST',
      headers: { cookie },
    });
    if (res.status === 401) {
      throw redirect(302, '/auth/login');
    }
  } catch (e) {
    if (e && typeof e === 'object' && 'status' in e) throw e;
    // Backend unreachable — let client-side guard handle it
  }

  return {};
};
