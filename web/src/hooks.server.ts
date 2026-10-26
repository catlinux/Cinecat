import type { Handle } from '@sveltejs/kit';
import { getSessionToken } from '$lib/session';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';

// La cookie httpOnly se lee en el servidor y queda disponible en locals para
// que los pasos posteriores protejan las rutas que lo necesiten.
export const handle: Handle = async ({ event, resolve }) => {
	event.locals.sessionToken = getSessionToken(event.cookies);
	return resolve(event, {
		transformPageChunk: ({ html }) => {
			const ui = event.locals.publicUI ?? DEFAULT_PUBLIC_UI;
			return html.replace(
				'<html lang="es">',
				`<html lang="es" data-theme-default="${ui.theme_default}" data-accent="${ui.accent}">`
			);
		}
	});
};
