// Writes the frontend's part of the third-party notices and copies the changelog into the
// build. Licenses such as MIT require their full text to ship with the software, so this
// includes each license's text, for the packages that really are in the bundle. The backend
// adds its own part in backend/cmd/notices.
import { copyFileSync, existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const packages = JSON.parse(readFileSync('.svelte-kit/shipped-packages.json', 'utf8'));
// Fonts arrive through CSS, which the record of JavaScript modules doesn't see.
for (const css of ['@fontsource-variable/inter']) if (!packages.includes(css)) packages.push(css);

const out = [
	'Wokko is licensed under the EUPL-1.2. It includes the third-party software and data listed here, under their own licenses.',
	'',
	'Data',
	'====',
	'Starting nutrition values: USDA FoodData Central, SR Legacy (2018-04). Public domain (CC0 1.0).',
	'',
	'Frontend',
	'========'
];
const missing = [];
for (const name of packages.sort()) {
	const dir = join('node_modules', name);
	if (!existsSync(join(dir, 'package.json'))) continue;
	const pkg = JSON.parse(readFileSync(join(dir, 'package.json'), 'utf8'));
	const file = readdirSync(dir).find((f) => /^(licen[cs]e|copying)(\.|-|$)/i.test(f));
	out.push('', `${name} ${pkg.version} (${pkg.license ?? 'see below'})`, '-'.repeat(60));
	if (file) out.push(readFileSync(join(dir, file), 'utf8').trim());
	else if (pkg.license) out.push(`Licensed under ${pkg.license}. The package includes no separate license file.`);
	else missing.push(name);
}
if (missing.length) throw new Error(`No license found for: ${missing.join(', ')}`);
writeFileSync('build/third-party-notices.txt', out.join('\n') + '\n');
if (existsSync('../CHANGELOG.md')) copyFileSync('../CHANGELOG.md', 'build/changelog.md');
