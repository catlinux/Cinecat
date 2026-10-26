// Vite ejecuta esta configuración en Node, pero el proyecto no instala
// @types/node; el import funciona igualmente en tiempo de ejecución.
// @ts-expect-error falta la declaración de node:fs
import { readFileSync } from 'node:fs';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// La autoridad de la versión es el VERSION de la raíz del repositorio. En
// Docker Vite se ejecuta desde /src/web, así que se prueba ../VERSION y, como
// respaldo, ./VERSION.
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
	plugins: [sveltekit()],
	define: {
		__APP_VERSION__: JSON.stringify(readAppVersion())
	}
});
