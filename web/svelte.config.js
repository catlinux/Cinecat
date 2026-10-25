import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// El cliente web se sirve desde un proceso Node detrás del proxy inverso
		// (Debian/Docker), igual que el servidor: no es una app estática.
		// La variable de entorno PROTOCOL_HEADER (ver docker-compose.yml) hace
		// que el servidor Node generado lea X-Forwarded-Proto de Caddy, para
		// que las cookies Secure se fijen bien aunque Node solo vea HTTP.
		adapter: adapter()
	}
};

export default config;
