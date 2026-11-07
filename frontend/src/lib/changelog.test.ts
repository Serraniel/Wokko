import { describe, expect, it } from 'vitest';
import { formatChangelog } from './changelog.ts';

describe('formatChangelog', () => {
	it('renders release-please output', () => {
		const html = formatChangelog('# Changelog\n\n## [0.2.0](https://github.com/x/y/compare/a...b) (2026-11-08)\n\n### Features\n\n* **ui:** add printing ([abc](https://github.com/x/y/commit/abc))\n');
		expect(html).toContain('href="https://github.com/x/y/compare/a...b"');
		expect(html).toContain('<li><strong>ui:</strong> add printing');
		expect(html).not.toContain('>Changelog<');
	});
	it('cannot be used to add markup', () => {
		expect(formatChangelog('* <img src=x onerror=alert(1)>')).not.toContain('<img');
	});
});
