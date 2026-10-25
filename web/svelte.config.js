import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// El cliente web se sirve desde un proceso Node detrás del proxy inverso
		// (Debian/Docker), igual que el servidor: no es una app estática.
		adapter: adapter()
	}
};

export default config;
