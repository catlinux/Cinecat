import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { ApiClient } from '$lib/api';
import { apiBaseUrl } from '$lib/config';
import { clearSessionCookie, getSessionToken, setSessionCookie } from '$lib/session';

export const load: PageServerLoad = ({ locals }) => {
	return { hasSession: locals.sessionToken !== null };
};

function createClient(fetchImpl: typeof fetch): ApiClient | null {
	if (!apiBaseUrl) {
		return null;
	}
	return new ApiClient({ baseUrl: apiBaseUrl, fetch: fetchImpl });
}

export const actions: Actions = {
	login: async ({ request, cookies, url, fetch }) => {
		const form = await request.formData();
		const username = String(form.get('username') ?? '').trim();
		const password = String(form.get('password') ?? '');

		if (!username || !password) {
			return fail(400, { username, message: 'Escribe tu usuario y contraseña.' });
		}

		const api = createClient(fetch);
		if (!api) {
			return fail(500, {
				username,
				message: 'Falta configurar PUBLIC_API_URL en el servidor.'
			});
		}

		try {
			const session = await api.login({ username, password });
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
				await createClient(fetch)?.logout(token);
			} catch {
				// Aunque el servidor no responda, la cookie local se borra igualmente.
				console.warn('No se pudo invalidar la sesión en el servidor.');
			}
		}
		clearSessionCookie(cookies, url.protocol === 'https:');
		redirect(303, '/login');
	}
};
