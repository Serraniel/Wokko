/** Works out what another app shared. Android often puts the link inside the text. */
export function readShared(params: { get(name: string): string | null }): { url: string; text: string } {
	const url = (params.get('url') ?? '').trim();
	const text = (params.get('text') ?? '').trim();
	const title = (params.get('title') ?? '').trim();
	if (/^https?:\/\//i.test(url)) return { url, text: '' };
	const found = text.match(/https?:\/\/[^\s<>"']+/i);
	if (found) return { url: found[0].replace(/[.,;:!?)\]]+$/, ''), text: '' };
	return { url: '', text: [title, text].filter(Boolean).join('\n') };
}

/** A bookmark that sends the page you're on to Wokko's import. */
export function bookmarklet(origin: string): string {
	return `javascript:location.href=${JSON.stringify(`${origin}/recipes/new?url=`)}+encodeURIComponent(location.href)`;
}
