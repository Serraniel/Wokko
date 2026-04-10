import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080';

export const load: PageServerLoad = async () => {
  try {
    const res = await fetch(`${BACKEND_URL}/api/auth/setup/status`);
    if (res.ok) {
      const data = await res.json();
      if (data.setup_complete) {
        throw redirect(302, '/auth/login');
      }
    }
  } catch (e) {
    if (e && typeof e === 'object' && 'status' in e) throw e;
  }
  return {};
};
