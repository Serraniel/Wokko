import { mkdirSync, writeFileSync } from 'node:fs';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

// Notes which packages really end up in the build, for the third-party notices.
const shipped = new Set<string>();
const recordPackages = {
	name: 'wokko-record-packages',
	moduleParsed(info: { id: string }) {
		const m = info.id.match(/.*\/node_modules\/((?:@[^/]+\/)?[^/]+)\//);
		if (m) shipped.add(m[1]);
	},
	closeBundle() {
		mkdirSync('.svelte-kit', { recursive: true });
		writeFileSync('.svelte-kit/shipped-packages.json', JSON.stringify([...shipped].sort()));
	}
};

export default defineConfig({
	plugins: [
		recordPackages,
		tailwindcss(),
		// A static app served by the Go binary, so there is no frontend server to break.
		sveltekit({ adapter: adapter({ fallback: 'index.html', strict: false }) })
	],
	server: { proxy: { '/api': 'http://localhost:8080', '/media': 'http://localhost:8080', '/_allauth': 'http://localhost:8080' } },
	test: { include: ['src/**/*.test.ts'], environment: 'node' }
});
