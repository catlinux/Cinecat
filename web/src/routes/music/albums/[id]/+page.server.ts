import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { parseMusicID } from '$lib/music';
import { requireSessionToken } from '$lib/session';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, locals, fetch }) => {
	const token = requireSessionToken(locals);
	const id = parseMusicID(params.id);
	if (id === null) {
		error(404, 'Álbum no encontrado');
	}

	try {
		const album = await createApiClient(fetch).getMusicAlbum(
			id,
			{ limit: 200, offset: 0 },
			token
		);
		return { album };
	} catch (cause) {
		if (cause instanceof ApiError) {
			if (cause.status === 401) {
				redirect(303, '/login');
			}
			if (cause.status === 404) {
				error(404, 'Álbum no encontrado');
			}
		}
		error(502, 'No se pudo cargar el álbum.');
	}
};
