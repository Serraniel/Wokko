// Editor drafts live in the browser, so a failed save, an expired session or a closed tab
// never loses what someone typed.
export function draftKey(recipeId: number | null): string {
	return `wokko-draft-${recipeId ?? 'new'}`;
}

export function saveDraft<T>(key: string, value: T): void {
	try {
		localStorage.setItem(key, JSON.stringify({ savedAt: Date.now(), value }));
	} catch {
		// Storage can be full or blocked. Editing keeps working without the safety net.
	}
}

export function loadDraft<T>(key: string): { savedAt: number; value: T } | null {
	try {
		const raw = localStorage.getItem(key);
		return raw ? JSON.parse(raw) : null;
	} catch {
		return null;
	}
}

export function clearDraft(key: string): void {
	try {
		localStorage.removeItem(key);
	} catch {
		// Nothing to do: the draft just stays until storage works again.
	}
}
