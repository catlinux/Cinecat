import { error, fail, redirect } from '@sveltejs/kit';
import { ApiError, type TrashItem } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import type { Actions, PageServerLoad } from './$types';

const ADMIN_MESSAGE = 'Esta sección es solo para administradores.';
const LOAD_ERROR = 'No se pudo cargar la papelera. Inténtalo de nuevo.';
const RESTORE_DISABLED =
	'No se pudo restaurar. Activa «Permitir cambiar archivos» en Administración → Biblioteca.';

function validID(value: FormDataEntryValue | null): number | null {
	const id = Number(value);
	return Number.isInteger(id) && id > 0 ? id : null;
}

function trashFailure(reason: unknown, fallback: string, restore = false) {
	if (reason instanceof ApiError) {
		if (reason.status === 401) {
			redirect(303, '/login');
		}
		if (reason.status === 403) {
			return fail(403, { message: restore ? RESTORE_DISABLED : ADMIN_MESSAGE });
		}
		if (reason.status === 404) {
			return fail(404, { message: reason.message || 'El elemento ya no está en la papelera.' });
		}
		if (reason.status === 409 && restore) {
			return fail(409, {
				message:
					reason.message ||
					'No se pudo restaurar porque la ruta original ya está ocupada.'
			});
		}
		if (reason.status === 400) {
			return fail(400, { message: reason.message || fallback });
		}
	}
	return fail(502, { message: fallback });
}

export const load: PageServerLoad = async ({ locals, fetch, url }) => {
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

	let items: TrashItem[];
	try {
		const response = await api.listTrash(token);
		items = response.items ?? [];
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, LOAD_ERROR);
	}

	return {
		items,
		moved: url.searchParams.get('movido') || null
	};
};

export const actions: Actions = {
	restore: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		if (id === null) {
			return fail(400, { message: 'Elemento de la papelera no válido.' });
		}
		try {
			await createApiClient(fetch).restoreTrash(id, token);
		} catch (reason) {
			return trashFailure(reason, 'No se pudo restaurar el elemento.', true);
		}
		return { success: true, message: 'Elemento restaurado.' };
	},

	delete: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		if (id === null) {
			return fail(400, { message: 'Elemento de la papelera no válido.' });
		}
		try {
			await createApiClient(fetch).deleteTrash(id, token);
		} catch (reason) {
			return trashFailure(reason, 'No se pudo eliminar el elemento.');
		}
		return { success: true, message: 'Elemento eliminado definitivamente.' };
	},

	purge: async ({ locals, fetch }) => {
		const token = requireSessionToken(locals);
		try {
			const { purged } = await createApiClient(fetch).purgeTrash(token);
			return {
				success: true,
				message:
					purged === 0
						? 'No había elementos caducados.'
						: `${purged} ${purged === 1 ? 'elemento purgado' : 'elementos purgados'}.`
			};
		} catch (reason) {
			return trashFailure(reason, 'No se pudo vaciar lo caducado.');
		}
	}
};
