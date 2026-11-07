// Writes the third-party notices linked from the settings page, from what is actually installed.
import { execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync } from 'node:fs';

const pkg = JSON.parse(readFileSync('package.json', 'utf8'));
const lines = ['Wokko is licensed under the EUPL-1.2. It includes the following third-party software.', '', 'Frontend', '--------'];
for (const name of Object.keys({ ...pkg.dependencies, ...pkg.devDependencies }).sort()) {
	const dep = JSON.parse(readFileSync(`node_modules/${name}/package.json`, 'utf8'));
	if (name.startsWith('@types/') || ['svelte-check', 'typescript', 'vitest', 'playwright-core'].includes(name)) continue;
	lines.push(`${name} ${dep.version} (${dep.license ?? 'see project'})`);
}
lines.push('', 'Backend', '-------');
let modules = '';
try {
	modules = execFileSync('go', ['list', '-m', 'all'], { cwd: '../backend', encoding: 'utf8' });
} catch {
	modules = readFileSync('../backend/go.mod', 'utf8').split('\n').filter((l) => /^\s+\S+\s+v/.test(l)).map((l) => l.replace('// indirect', '').trim()).join('\n');
}
for (const line of modules.split('\n').slice(0).filter(Boolean)) if (!line.startsWith('github.com/serraniel/wokko')) lines.push(line);
lines.push('', 'Data', '----', 'Starting nutrition values: USDA FoodData Central, SR Legacy (2018-04). Public domain (CC0).');
lines.push('', 'Each package is under its own license, found in its source repository.', '');
writeFileSync('build/third-party-notices.txt', lines.join('\n'));
