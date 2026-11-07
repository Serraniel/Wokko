// Recipes from kitshn and Tandoor can hold simple Markdown. Everything is escaped first, so
// only the tags made here can reach the page. Links must be http or https.
function escapeHtml(s: string): string {
	return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

export function renderMarkdown(text: string): string {
	return escapeHtml(text)
		.replace(/\[([^\]\n]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" rel="noopener noreferrer nofollow" target="_blank" class="underline">$1</a>')
		.replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>')
		.replace(/(^|[\s(])\*(?=\S)([^*\n]*\S)\*/g, '$1<em>$2</em>');
}
