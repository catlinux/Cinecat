import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { ApiError, type Session } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { clearSessionCookie, requireSessionToken } from '$lib/session';

export const load: PageServerLoad = async ({ locals, fetch }) => {
	const token = requireSessionToken(locals);
	let sessions: Session[] | null = null;
	try {
		sessions = (await createApiClient(fetch).getSessions(token)).sessions ?? [];
	} catch {
		// La gestión de la contraseña sigue disponible aunque falle la lista.
		sessions = null;
	}
	return { sessions };
};

export const actions: Actions = {
	default: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();

		const currentPassword = String(form.get('current_password') ?? '');
		const newPassword = String(form.get('new_password') ?? '');
		const repeatPassword = String(form.get('repeat_password') ?? '');

		if (!currentPassword || !newPassword || !repeatPassword) {
			return fail(400, { message: 'Rellena los tres campos.' });
		}
		if (Array.from(newPassword).length < 8) {
			return fail(400, { message: 'La contraseña nueva debe tener al menos 8 caracteres.' });
		}
		if (newPassword !== repeatPassword) {
			return fail(400, { message: 'Las contraseñas nuevas no coinciden.' });
		}

		try {
			await createApiClient(fetch).changePassword(currentPassword, newPassword, token);
		} catch (error) {
			if (error instanceof ApiError) {
				if (error.status === 401) {
					return fail(401, {
						message:
							'La contraseña actual no es correcta o la sesión ha caducado. Revisa la contraseña o vuelve a iniciar sesión.'
					});
				}
				if (error.status === 400) {
					return fail(400, {
						message: error.message || 'La contraseña nueva no es válida.'
					});
				}
			}
			return fail(502, {
				message: 'No se pudo cambiar la contraseña. Inténtalo de nuevo en unos momentos.'
			});
		}

		return {
			success: true,
			message: 'Contraseña actualizada. Se han cerrado tus sesiones en otros dispositivos; la de este sigue abierta.'
		};
	},

	closeSession: async ({ request, locals, fetch, cookies, url }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = Number(form.get('id'));
		if (!Number.isInteger(id) || id <= 0) {
			return fail(400, { message: 'Sesión no válida.', closeSession: true });
		}

		const api = createApiClient(fetch);
		const currentValue = form.get('_current') ?? form.get('current');
		let wasCurrent = currentValue === '1' || currentValue === 'true';
		if (!form.has('_current') && !form.has('current')) {
			try {
				const response = await api.getSessions(token);
				wasCurrent = Boolean(
					(response.sessions ?? []).some((session) => session.id === id && session.current)
				);
			} catch (reason) {
				if (reason instanceof ApiError && reason.status === 401) {
					clearSessionCookie(cookies, url.protocol === 'https:');
					redirect(303, '/login');
				}
				// Si no se puede comprobar, se intenta cerrar igualmente.
			}
		}

		try {
			await api.closeSession(id, token);
		} catch (reason) {
			if (reason instanceof ApiError) {
				if (reason.status === 401) {
					clearSessionCookie(cookies, url.protocol === 'https:');
					redirect(303, '/login');
				}
				if (reason.status === 404) {
					return fail(404, {
						message: reason.message || 'Sesión no encontrada.',
						closeSession: true
					});
				}
			}
			return fail(502, {
				message: 'No se pudo cerrar la sesión. Inténtalo de nuevo.',
				closeSession: true
			});
		}

		if (wasCurrent) {
			clearSessionCookie(cookies, url.protocol === 'https:');
			redirect(303, '/login');
		}
		return { success: true, message: 'Sesión cerrada.', closeSession: true };
	}
};
