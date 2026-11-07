import { renderMarkdown } from './markdown.ts';

// The changelog is written by release-please in a small, known Markdown subset. Each line goes
// through the escaping Markdown renderer, so nothing in it can add its own HTML.
export function formatChangelog(text: string): string {
	const out: string[] = [];
	let list = false;
	const closeList = () => {
		if (list) out.push('</ul>');
		list = false;
	};
	for (const raw of text.split('\n')) {
		const line = raw.trim();
		const heading = line.match(/^(#{1,4})\s+(.*)$/);
		if (heading) {
			closeList();
			// A release's title holds a compare link: [0.2.0](https://…) (2026-11-08).
			const size = heading[1].length <= 2 ? 'mt-5 text-base font-semibold' : 'mt-3 font-semibold text-muted';
			if (heading[1].length > 1 || !/^changelog$/i.test(heading[2])) out.push(`<h2 class="${size}">${renderMarkdown(heading[2])}</h2>`);
		} else if (/^[*-]\s+/.test(line)) {
			if (!list) out.push('<ul class="list-disc space-y-1 pl-5">');
			list = true;
			out.push(`<li>${renderMarkdown(line.replace(/^[*-]\s+/, ''))}</li>`);
		} else if (line) {
			closeList();
			out.push(`<p>${renderMarkdown(line)}</p>`);
		}
	}
	closeList();
	return out.join('');
}
