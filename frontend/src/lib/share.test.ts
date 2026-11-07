import { describe, expect, it } from 'vitest';
import { bookmarklet, readShared } from './share.ts';

describe('readShared', () => {
	it('uses the url field', () => {
		expect(readShared(new URLSearchParams({ url: 'https://e.com/r', text: 'x' }))).toEqual({ url: 'https://e.com/r', text: '' });
	});
	it('finds a link inside shared text, without trailing punctuation', () => {
		expect(readShared(new URLSearchParams({ text: 'Lecker! https://www.chefkoch.de/rezepte/1/L.html.' })).url).toBe('https://www.chefkoch.de/rezepte/1/L.html');
	});
	it('treats anything else as recipe text', () => {
		expect(readShared(new URLSearchParams({ title: 'Suppe', text: '2 Karotten\nKochen.' }))).toEqual({ url: '', text: 'Suppe\n2 Karotten\nKochen.' });
	});
	it('ignores links that aren\'t web addresses', () => {
		expect(readShared(new URLSearchParams({ url: 'javascript:alert(1)' })).url).toBe('');
	});
	it('is empty when nothing was shared', () => {
		expect(readShared(new URLSearchParams())).toEqual({ url: '', text: '' });
	});
});

describe('bookmarklet', () => {
	it('sends the current page to this server', () => {
		expect(bookmarklet('https://w.example')).toBe('javascript:location.href="https://w.example/recipes/new?url="+encodeURIComponent(location.href)');
	});
});
