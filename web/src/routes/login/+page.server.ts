import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { createApiClient } from '$lib/api-factory';
import { clearSessionCookie, getSessionToken, setSessionCookie } from '$lib/session';

export const load: PageServerLoad = ({ locals }) => {
	return { hasSession: locals.sessionToken !== null };
};

export const actions: Actions = {
	login: async ({ request, cookies, url, fetch, getClientAddress }) => {
		const form = await request.formData();
		const username = String(form.get('username') ?? '').trim();
		const password = String(form.get('password') ?? '');
		let clientIP = '';
		try {
			clientIP = getClientAddress() || '';
		} catch {
			// En algunos despliegues o pruebas no hay dirección de cliente disponible.
			clientIP = '';
		}

		if (!username || !password) {
			return fail(400, { username, message: 'Escribe tu usuario y contraseña.' });
		}

		const api = createApiClient(fetch);

		try {
			const session = await api.login(
				{ username, password },
				clientIP,
				request.headers.get('user-agent') ?? ''
			);
			setSessionCookie(cookies, session.token, session.expires_at, url.protocol === 'https:');
		} catch (error) {
			const status = typeof error === 'object' && error !== null && 'status' in error
				? Number(error.status)
				: 0;

			if (status === 401) {
				return fail(401, { username, message: 'Usuario o contraseña incorrectos.' });
			}
			if (status === 429) {
				return fail(429, {
					username,
					message: 'Demasiados intentos fallidos. Espera unos minutos antes de volver a intentarlo.'
				});
			}
			return fail(502, {
				username,
				message: 'No se pudo iniciar sesión. Inténtalo de nuevo en unos momentos.'
			});
		}

		redirect(303, '/');
	},

	logout: async ({ cookies, url, fetch }) => {
		const token = getSessionToken(cookies);
		if (token) {
			try {
				await createApiClient(fetch).logout(token);
			} catch {
				// Aunque el servidor no responda, la cookie local se borra igualmente.
				console.warn('No se pudo invalidar la sesión en el servidor.');
			}
		}
		clearSessionCookie(cookies, url.protocol === 'https:');
		redirect(303, '/login');
	}
};
