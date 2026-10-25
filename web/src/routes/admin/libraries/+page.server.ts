import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';

const ADMIN_MESSAGE = 'Esta sección es solo para administradores.';

function actionFailure(
	reason: unknown,
	fallback: string,
	extra: Record<string, string> = {}
) {
	if (reason instanceof ApiError) {
		if (reason.status === 401) {
			redirect(303, '/login');
		}
		if (reason.status === 403) {
			return fail(403, { message: ADMIN_MESSAGE, ...extra });
		}
		if (reason.status === 400 || reason.status === 404) {
			return fail(reason.status, { message: reason.message || fallback, ...extra });
		}
	}
	return fail(502, { message: fallback, ...extra });
}

export const load: PageServerLoad = async ({ locals, fetch }) => {
	const token = requireSessionToken(locals);
	const api = createApiClient(fetch);

	let user;
	try {
		user = await api.getCurrentUser(token);
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, 'No se pudo comprobar tu cuenta. Inténtalo de nuevo.');
	}

	if (user.role !== 'admin') {
		error(403, ADMIN_MESSAGE);
	}

	try {
		const { libraries } = await api.listLibraries(token);
		return { libraries };
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, 'No se pudieron cargar las bibliotecas. Inténtalo de nuevo.');
	}
};

export const actions: Actions = {
	create: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const path = String(form.get('path') ?? '').trim();

		if (!name || !path) {
			return fail(400, {
				message: 'Escribe un nombre y una ruta para la biblioteca.',
				name,
				path
			});
		}

		try {
			await createApiClient(fetch).createLibrary(name, path, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo crear la biblioteca.', { name, path });
		}

		return { success: true, message: 'Biblioteca creada.' };
	},

	delete: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = Number(form.get('id'));

		if (!Number.isInteger(id) || id <= 0) {
			return fail(400, { message: 'Biblioteca no válida.' });
		}

		try {
			await createApiClient(fetch).deleteLibrary(id, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo borrar la biblioteca.');
		}

		return { success: true, message: 'Biblioteca borrada.' };
	},

	scan: async ({ locals, fetch }) => {
		const token = requireSessionToken(locals);

		try {
			const scan = await createApiClient(fetch).scanLibraries(token);
			return { success: true, message: 'Escaneo completado.', scan };
		} catch (reason) {
			return actionFailure(reason, 'No se pudo completar el escaneo.');
		}
	}
};
