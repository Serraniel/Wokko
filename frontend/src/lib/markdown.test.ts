import { describe, expect, it } from 'vitest';
import { renderMarkdown } from './markdown.ts';

describe('renderMarkdown', () => {
	it('handles bold, italics and links', () => {
		expect(renderMarkdown('**a** and *b*')).toBe('<strong>a</strong> and <em>b</em>');
		expect(renderMarkdown('[x](https://e.com/a)')).toContain('href="https://e.com/a"');
	});
	it('never lets markup through', () => {
		expect(renderMarkdown('<img src=x onerror=alert(1)>')).not.toContain('<img');
		expect(renderMarkdown('[x](javascript:alert(1))')).not.toContain('<a');
		expect(renderMarkdown('[x](https://e.com/"onmouseover="alert(1))')).not.toContain('"onmouseover');
		expect(renderMarkdown('**<script>**')).toBe('<strong>&lt;script&gt;</strong>');
	});
	it('leaves ordinary text alone', () => {
		expect(renderMarkdown('2 * 3 and 5 * 6')).toBe('2 * 3 and 5 * 6');
	});
});
