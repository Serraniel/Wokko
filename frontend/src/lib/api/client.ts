import { t } from '#lib/i18n/index.svelte.ts';

export class ApiError extends Error {
	constructor(
		public status: number,
		public code: string,
		message: string
	) {
		super(message);
	}

	/** The message in the user's language, whichever language the server used. */
	get localized(): string {
		const key = `err.${this.code}`;
		const out = t(key);
		return out === key ? this.message : out;
	}
}

export function errorText(e: unknown): string {
	if (e instanceof ApiError) return e.localized;
	return t('err.network');
}

/** Called when any request finds the session has ended. */
let onUnauthorized: () => void = () => {};
export function setUnauthorizedHandler(fn: () => void) {
	onUnauthorized = fn;
}

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

async function request<T>(method: Method, path: string, body?: unknown): Promise<T> {
	let res: Response;
	try {
		res = await fetch(`/api/v1${path}`, {
			method,
			credentials: 'same-origin',
			headers: body !== undefined && !(body instanceof FormData) ? { 'Content-Type': 'application/json' } : undefined,
			body: body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined
		});
	} catch {
		throw new ApiError(0, 'network', 'Network error');
	}
	if (res.ok) return (await res.json()) as T;
	let code = 'server_error';
	let message = res.statusText;
	try {
		const data = await res.json();
		code = data.error?.code ?? code;
		message = data.error?.message ?? message;
	} catch {
		// Not JSON, for example from a reverse proxy. The status text will do.
	}
	if (res.status === 401 && code === 'unauthorized') onUnauthorized();
	throw new ApiError(res.status, code, message);
}

export const api = {
	get: <T>(path: string) => request<T>('GET', path),
	post: <T>(path: string, body: unknown = {}) => request<T>('POST', path, body),
	put: <T>(path: string, body: unknown) => request<T>('PUT', path, body),
	patch: <T>(path: string, body: unknown) => request<T>('PATCH', path, body),
	del: <T = { ok: boolean }>(path: string) => request<T>('DELETE', path),
	upload: <T>(path: string, file: File) => {
		const form = new FormData();
		form.append('photo', file);
		return request<T>('POST', path, form);
	}
};

export function query(params: Record<string, string | number | undefined | null>): string {
	const q = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== null && v !== '' && v !== 0) q.set(k, String(v));
	const s = q.toString();
	return s ? `?${s}` : '';
}
