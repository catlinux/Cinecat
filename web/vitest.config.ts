import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vitest/config';

// Misma fuente de versión que vite.config.ts, para que los tests vean la
// constante compilada.
function readAppVersion(): string {
	for (const relative of ['../VERSION', './VERSION']) {
		try {
			const version = readFileSync(new URL(relative, import.meta.url), 'utf8').trim();
			if (version) {
				return version;
			}
		} catch {
			// Se prueba el siguiente candidato.
		}
	}
	return 'desconocida';
}

export default defineConfig({
	plugins: [svelte()],
	define: {
		__APP_VERSION__: JSON.stringify(readAppVersion())
	},
	resolve: {
		alias: {
			$lib: fileURLToPath(new URL('./src/lib', import.meta.url)),
			'$env/dynamic/public': fileURLToPath(new URL('./src/lib/test-env.ts', import.meta.url)),
			'$app/forms': fileURLToPath(new URL('./src/lib/test-app-forms.ts', import.meta.url))
		}
	},
	test: {
		environment: 'node',
		include: ['src/**/*.test.ts']
	}
});
