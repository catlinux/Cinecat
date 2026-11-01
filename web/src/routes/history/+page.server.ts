import { error, fail, redirect } from '@sveltejs/kit';
import { ApiError, type HistoryEntry } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import type { Actions, PageServerLoad } from './$types';

const LOAD_ERROR = 'No se pudo cargar tu historial. Inténtalo de nuevo.';
const DELETE_ERROR = 'No se pudo quitar la entrada del historial.';

function validItemID(value: FormDataEntryValue | null): number | null {
	const id = Number(value);
	return Number.isInteger(id) && id > 0 ? id : null;
}

export const load: PageServerLoad = async ({ locals, fetch }) => {
	const token = requireSessionToken(locals);
	const api = createApiClient(fetch);
	let entries: HistoryEntry[];
	try {
		entries = (await api.getHistory(token)).entries ?? [];
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, LOAD_ERROR);
	}
	return { entries };
};

export const actions: Actions = {
	delete: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const itemID = validItemID(form.get('item_id'));
		if (itemID === null) {
			return fail(400, { message: 'Entrada del historial no válida.' });
		}
		try {
			await createApiClient(fetch).deleteHistoryItem(itemID, token);
		} catch (reason) {
			if (reason instanceof ApiError) {
				if (reason.status === 401) {
					redirect(303, '/login');
				}
				if (reason.status === 404) {
					return fail(404, { message: 'El elemento ya no está disponible.' });
				}
			}
			return fail(502, { message: DELETE_ERROR });
		}
		return { success: true, message: 'Entrada quitada del historial.' };
	}
};
