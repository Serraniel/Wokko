// Whether the device thinks it has a connection. A server can still be out of reach while this
// says yes, so nothing relies on it alone. Requests decide for themselves when they fail.
class Connection {
	online = $state(typeof navigator === 'undefined' ? true : navigator.onLine);

	constructor() {
		if (typeof window === 'undefined') return;
		window.addEventListener('online', () => (this.online = true));
		window.addEventListener('offline', () => (this.online = false));
	}
}

export const connection = new Connection();
