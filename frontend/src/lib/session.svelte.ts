import { api, ApiError } from '#lib/api/client.ts';
import type { Status, User } from '#lib/api/types.gen.ts';
import { bestLocale, setLocale } from '#lib/i18n/index.svelte.ts';
import { clearOfflineData } from '#lib/offline.ts';

// One place knows who is logged in. It loads before any page shows, so pages never
// have to guess, which is what caused the old app's admin and redirect bugs.
class Session {
	user = $state<User | null>(null);
	status = $state<Status | null>(null);
	ready = $state(false);

	get isAdmin() {
		return this.user?.role === 'admin';
	}

	async load() {
		this.status = await api.get<Status>('/status');
		try {
			this.user = await api.get<User>('/me');
		} catch (e) {
			if (!(e instanceof ApiError) || e.status !== 401) throw e;
			this.user = null;
		}
		await setLocale(bestLocale(this.user?.locale));
		this.ready = true;
	}

	async signedIn(user: User) {
		this.user = user;
		if (this.status) this.status.setup_complete = true;
		await setLocale(bestLocale(user.locale));
	}

	signedOut() {
		this.user = null;
		clearOfflineData();
	}
}

export const session = new Session();
