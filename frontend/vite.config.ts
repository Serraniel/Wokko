import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		tailwindcss(),
		// A static app served by the Go binary, so there is no frontend server to break.
		sveltekit({ adapter: adapter({ fallback: 'index.html', strict: false }) })
	],
	server: { proxy: { '/api': 'http://localhost:8080', '/media': 'http://localhost:8080', '/_allauth': 'http://localhost:8080' } },
	test: { include: ['src/**/*.test.ts'], environment: 'node' }
});
