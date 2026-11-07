import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

const here = import.meta.dirname;
const dir = join(here, 'messages');
const load = (f: string) => JSON.parse(readFileSync(join(dir, f), 'utf8')) as Record<string, string>;
const en = load('en.json');
const others = readdirSync(dir).filter((f) => f !== 'en.json');
const placeholders = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort();

describe('translations', () => {
	it('has at least one other language, proving the system is not English-only', () => {
		expect(others.length).toBeGreaterThan(0);
	});

	it.each(others)('%s has every key and no extras', (file: string) => {
		const m = load(file);
		// Plural keys depend on the language's rules, so they are compared by their base name.
		const base = (k: string) => k.replace(/_(zero|one|two|few|many|other)$/, '');
		expect([...new Set(Object.keys(m).map(base))].sort()).toEqual([...new Set(Object.keys(en).map(base))].sort());
	});

	it.each(others)('%s keeps every placeholder', (file: string) => {
		const m = load(file);
		for (const [k, v] of Object.entries(m)) {
			if (en[k] !== undefined) expect(placeholders(v), k).toEqual(placeholders(en[k]));
		}
	});

	it('names each language in its own language', () => {
		for (const f of ['en.json', ...others]) expect(load(f)._language).toBeTruthy();
	});

	it('defines every key the code uses', () => {
		const src = join(here, '../..');
		const missing: string[] = [];
		const scan = (d: string) => {
			for (const e of readdirSync(d, { withFileTypes: true })) {
				const p = join(d, e.name);
				if (e.isDirectory()) scan(p);
				else if (/\.(svelte|ts)$/.test(e.name) && !e.name.includes('.test.')) {
					for (const m of readFileSync(p, 'utf8').matchAll(/\bt\('([a-z_]+\.[a-z_]+)'/g)) {
						if (!(m[1] in en) && !(`${m[1]}_other` in en)) missing.push(`${e.name}: ${m[1]}`);
					}
				}
			}
		};
		scan(src);
		expect(missing).toEqual([]);
	});

	it('translates every error code the backend can send', () => {
		const backend = join(here, '../../../../backend/internal');
		const codes = new Set<string>();
		const scan = (d: string) => {
			for (const e of readdirSync(d, { withFileTypes: true })) {
				const p = join(d, e.name);
				if (e.isDirectory()) scan(p);
				else if (e.name.endsWith('.go') && !e.name.endsWith('_test.go')) {
					const src = readFileSync(p, 'utf8');
					for (const m of src.matchAll(/(?:fail\(http\.\w+, |invalid\(|Code: )"([a-z_]+)"/g)) codes.add(m[1]);
				}
			}
		};
		scan(backend);
		expect(codes.size).toBeGreaterThan(20);
		for (const c of codes) expect(en, `error code ${c}`).toHaveProperty(`err.${c}`);
	});
});
