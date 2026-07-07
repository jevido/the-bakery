import { mdsvex } from 'mdsvex';
import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
	// Vite only exposes .env values via import.meta.env by default; server-side
	// code (auth.ts, db/index.ts) reads plain process.env so it also works
	// under CLI tools (drizzle-kit, better-auth generate) that run outside Vite.
	Object.assign(process.env, loadEnv(mode, process.cwd(), ''));

	return {
		plugins: [
			tailwindcss(),
			sveltekit({
				compilerOptions: {
					// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
					runes: ({ filename }) =>
						filename.split(/[/\\]/).includes('node_modules') ? undefined : true,
					experimental: { async: true }
				},
				adapter: adapter(),
				preprocess: [mdsvex({ extensions: ['.svx', '.md'] })],
				extensions: ['.svelte', '.svx', '.md'],
				experimental: {
					remoteFunctions: true,
					handleRenderingErrors: true,
					forkPreloads: true
				},
				typescript: {
					config: (config) => ({
						...config,
						include: [...config.include, '../drizzle.config.ts']
					})
				}
			})
		]
	};
});
