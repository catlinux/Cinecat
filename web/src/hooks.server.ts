import type { Handle } from '@sveltejs/kit';
import { getSessionToken } from '$lib/session';

// La cookie httpOnly se lee en el servidor y queda disponible en locals para
// que los pasos posteriores protejan las rutas que lo necesiten.
export const handle: Handle = async ({ event, resolve }) => {
	event.locals.sessionToken = getSessionToken(event.cookies);
	return resolve(event);
};
