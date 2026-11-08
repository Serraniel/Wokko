import { redirect } from '@sveltejs/kit';

// Bringing in recipes moved to Integrations. Old bookmarks still land in the right place.
export function load() {
	redirect(307, '/integrations');
}
