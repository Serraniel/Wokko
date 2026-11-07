// The browser's passkey calls use raw bytes where the server's JSON has base64url text.
type Json = Record<string, unknown>;

function toBytes(s: string): ArrayBuffer {
	const raw = atob(s.replace(/-/g, '+').replace(/_/g, '/').padEnd(Math.ceil(s.length / 4) * 4, '='));
	return Uint8Array.from(raw, (c) => c.charCodeAt(0)).buffer;
}

function toText(b: ArrayBuffer): string {
	return btoa(String.fromCharCode(...new Uint8Array(b))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/** Passkeys only work on HTTPS or localhost, and only in browsers that have them. */
export function passkeysAvailable(): boolean {
	return typeof window !== 'undefined' && window.isSecureContext && 'PublicKeyCredential' in window;
}

export async function createPasskey(options: { publicKey: Json }): Promise<Json> {
	const o = options.publicKey as Json & { challenge: string; user: { id: string }; excludeCredentials?: { id: string }[] };
	const cred = (await navigator.credentials.create({
		publicKey: { ...o, challenge: toBytes(o.challenge), user: { ...o.user, id: toBytes(o.user.id) }, excludeCredentials: (o.excludeCredentials ?? []).map((c) => ({ ...c, id: toBytes(c.id) })) } as unknown as PublicKeyCredentialCreationOptions
	})) as PublicKeyCredential;
	const r = cred.response as AuthenticatorAttestationResponse;
	return { id: cred.id, rawId: toText(cred.rawId), type: cred.type, response: { clientDataJSON: toText(r.clientDataJSON), attestationObject: toText(r.attestationObject) } };
}

export async function usePasskey(options: { publicKey: Json }): Promise<Json> {
	const o = options.publicKey as Json & { challenge: string; allowCredentials?: { id: string }[] };
	const cred = (await navigator.credentials.get({
		publicKey: { ...o, challenge: toBytes(o.challenge), allowCredentials: (o.allowCredentials ?? []).map((c) => ({ ...c, id: toBytes(c.id) })) } as unknown as PublicKeyCredentialRequestOptions
	})) as PublicKeyCredential;
	const r = cred.response as AuthenticatorAssertionResponse;
	return { id: cred.id, rawId: toText(cred.rawId), type: cred.type, response: { clientDataJSON: toText(r.clientDataJSON), authenticatorData: toText(r.authenticatorData), signature: toText(r.signature), userHandle: r.userHandle ? toText(r.userHandle) : null } };
}
