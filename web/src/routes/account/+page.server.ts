import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';

export const load: PageServerLoad = ({ locals }) => {
	requireSessionToken(locals);
	return {};
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
	}
};
