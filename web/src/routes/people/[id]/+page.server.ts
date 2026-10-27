import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { PersonMedia } from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, locals, fetch }) => {
	const token = requireSessionToken(locals);
	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		error(404, 'No encontramos a esta persona');
	}

	try {
		const response = await createApiClient(fetch).getPerson(id, token);
		return {
			person: response.person,
			inLibrary: response.in_library ?? [],
			notInLibrary: response.not_in_library ?? [],
			errorMessage: null
		};
	} catch (cause) {
		if (cause instanceof ApiError) {
			if (cause.status === 401) {
				redirect(303, '/login');
			}
			if (cause.status === 404) {
				error(404, 'No encontramos a esta persona');
			}
			if (cause.status === 502) {
				return {
					person: null,
					inLibrary: [] as PersonMedia[],
					notInLibrary: [] as PersonMedia[],
					errorMessage: 'TMDB no responde ahora; inténtalo más tarde.'
				};
			}
		}
		if (cause instanceof ConfigurationError) {
			error(500, cause.message);
		}
		error(502, 'No se pudo cargar la persona');
	}
};
