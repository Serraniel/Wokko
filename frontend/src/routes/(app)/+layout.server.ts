import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies }) => {
  if (!cookies.get('refresh_token')) {
    throw redirect(302, '/auth/login');
  }

  return {};
};
