import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { MediaItem } from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, locals, fetch }) => {
	const token = requireSessionToken(locals);
	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		error(404, 'Ítem no encontrado');
	}

	try {
		const item = await createApiClient(fetch).get<MediaItem>(`/api/v1/items/${id}`, token);
		return { item };
	} catch (err) {
		if (err instanceof ApiError) {
			if (err.status === 404) {
				error(404, 'Ítem no encontrado');
			}
			if (err.status === 401) {
				redirect(303, '/login');
			}
		}
		if (err instanceof ConfigurationError) {
			error(500, err.message);
		}
		error(502, 'No se pudo cargar el ítem.');
	}
};
